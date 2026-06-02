#!/usr/bin/env python3
"""Batch import skills into the catalog using the packaging + publish flow."""

from __future__ import annotations

import argparse
import base64
import json
import os
import re
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any

PACKAGE_NAME_RE = re.compile(r"^(?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$")
EXACT_VERSION_RE = re.compile(
    r"^\d+\.\d+\.\d+(?:-[0-9A-Za-z]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$"
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Batch import existing skills through Catalog packaging APIs and npm publish."
    )
    parser.add_argument("--manifest", help="JSON manifest file describing skills to import.")
    parser.add_argument("--root-dir", help="Directory that contains skill folders to scan.")
    parser.add_argument(
        "--recursive",
        action="store_true",
        help="Recursively scan --root-dir for directories that contain a root SKILL.md.",
    )
    parser.add_argument(
        "--catalog-url",
        default=os.environ.get("SKILLS_REGISTRY_URL"),
        help="Catalog base URL. Defaults to SKILLS_REGISTRY_URL.",
    )
    parser.add_argument(
        "--catalog-token",
        default=os.environ.get("SKILLS_REGISTRY_TOKEN"),
        help="Catalog bearer token. Defaults to SKILLS_REGISTRY_TOKEN.",
    )
    parser.add_argument(
        "--catalog-role",
        default=os.environ.get("SKILLS_REGISTRY_ROLE"),
        help="Optional X-Role header value.",
    )
    parser.add_argument(
        "--npm-registry",
        help="Optional registry passed to `npm publish --registry ...`.",
    )
    parser.add_argument(
        "--publish-registry",
        help="Registry recorded into package metadata and release-record. Defaults to --npm-registry.",
    )
    parser.add_argument(
        "--source-type",
        default="imported",
        help="Default metadata.sourceType for prepared bundles. Defaults to imported.",
    )
    parser.add_argument(
        "--package-scope",
        default="@platform.skill/",
        help="Scope prefix used when inferring package names. Defaults to @platform.skill/.",
    )
    parser.add_argument(
        "--default-version",
        default="0.1.0",
        help="Version used when inferring versions during root-dir scan. Defaults to 0.1.0.",
    )
    parser.add_argument(
        "--infer-targets",
        action="store_true",
        help="Allow manifest entries without packageName/version to infer them from local directory names.",
    )
    parser.add_argument(
        "--continue-on-error",
        action="store_true",
        help="Keep importing remaining entries after a failure.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Resolve the import plan and print it without calling Catalog or npm publish.",
    )
    args = parser.parse_args()

    if not args.manifest and not args.root_dir:
        parser.error("one of --manifest or --root-dir is required")
    if not args.catalog_url and not args.dry_run:
        parser.error("--catalog-url is required unless --dry-run is used")
    if args.package_scope and not args.package_scope.endswith("/"):
        parser.error("--package-scope must end with '/'")
    return args


def build_headers(args: argparse.Namespace) -> dict[str, str]:
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
    }
    if args.catalog_token:
        headers["Authorization"] = f"Bearer {args.catalog_token}"
    if args.catalog_role:
        headers["X-Role"] = args.catalog_role
    return headers


def request_json(
    method: str,
    url: str,
    payload: dict[str, Any],
    headers: dict[str, str],
) -> dict[str, Any]:
    request = urllib.request.Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers=headers,
        method=method,
    )
    try:
        with urllib.request.urlopen(request) as response:
            body = response.read().decode("utf-8")
            return json.loads(body) if body else {}
    except urllib.error.HTTPError as error:
        body = error.read().decode("utf-8", errors="replace")
        try:
            parsed = json.loads(body)
            error_body = parsed.get("error") if isinstance(parsed, dict) else None
            if isinstance(error_body, dict):
                code = error_body.get("code", "catalog_request_failed")
                message = error_body.get("message", f"Catalog request failed with status {error.code}")
                raise RuntimeError(f"{code}: {message}") from None
        except json.JSONDecodeError:
            pass
        raise RuntimeError(f"Catalog request failed with status {error.code}: {body}") from None
    except urllib.error.URLError as error:
        raise RuntimeError(f"Catalog request failed: {error.reason}") from None


def parse_frontmatter(skill_md_path: Path) -> dict[str, str]:
    content = skill_md_path.read_text(encoding="utf-8")
    if not content.startswith("---\n"):
        return {}
    end = content.find("\n---\n", 4)
    if end == -1:
        return {}
    frontmatter = content[4:end]
    result: dict[str, str] = {}
    for line in frontmatter.splitlines():
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        result[key.strip()] = value.strip().strip("'\"")
    return result


def load_package_json(directory: Path) -> dict[str, Any]:
    package_json_path = directory / "package.json"
    if not package_json_path.exists():
        return {}
    return json.loads(package_json_path.read_text(encoding="utf-8"))


def slugify_package_name(directory_name: str) -> str:
    value = re.sub(r"[^a-z0-9._-]+", "-", directory_name.lower()).strip("._-")
    if not value:
        raise ValueError(f"cannot infer package name from directory {directory_name!r}")
    return value


def inspect_local_skill(directory: Path) -> dict[str, Any]:
    skill_md_path = directory / "SKILL.md"
    if not skill_md_path.exists():
        raise ValueError(f"{directory} is missing SKILL.md")

    frontmatter = parse_frontmatter(skill_md_path)
    package_json = load_package_json(directory)

    publish_registry = None
    publish_config = package_json.get("publishConfig")
    if isinstance(publish_config, dict):
        publish_registry = publish_config.get("registry")

    skill_meta = package_json.get("skill")
    source_type = None
    if isinstance(skill_meta, dict):
        source_type = skill_meta.get("sourceType")

    return {
        "displayName": frontmatter.get("name"),
        "description": frontmatter.get("description"),
        "packageName": package_json.get("name") if isinstance(package_json.get("name"), str) else None,
        "version": package_json.get("version") if isinstance(package_json.get("version"), str) else None,
        "publishRegistry": publish_registry if isinstance(publish_registry, str) else None,
        "sourceType": source_type if isinstance(source_type, str) else None,
    }


def scan_root_dir(root_dir: Path, recursive: bool) -> list[dict[str, Any]]:
    skills: list[dict[str, Any]] = []
    seen: set[Path] = set()

    def add_candidate(candidate: Path) -> None:
        if candidate in seen:
            return
        if (candidate / "SKILL.md").exists():
            skills.append({"path": str(candidate)})
            seen.add(candidate)

    add_candidate(root_dir)
    if recursive:
        for current_root, dirnames, _ in os.walk(root_dir):
            current_path = Path(current_root)
            add_candidate(current_path)
            dirnames[:] = [name for name in dirnames if name not in {".git", "node_modules", ".venv"}]
    else:
        for child in root_dir.iterdir():
            if child.is_dir():
                add_candidate(child)
    return skills


def load_manifest(manifest_path: Path) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    payload = json.loads(manifest_path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise ValueError("manifest must be a JSON object")
    defaults = payload.get("defaults", {})
    if defaults is None:
        defaults = {}
    if not isinstance(defaults, dict):
        raise ValueError("manifest.defaults must be an object when provided")
    skills = payload.get("skills", [])
    if not isinstance(skills, list):
        raise ValueError("manifest.skills must be an array")
    normalized: list[dict[str, Any]] = []
    for item in skills:
        if not isinstance(item, dict):
            raise ValueError("each manifest.skills entry must be an object")
        merged = dict(defaults)
        merged.update(item)
        normalized.append(merged)
    return defaults, normalized


def validate_package_name(value: str) -> str:
    if not PACKAGE_NAME_RE.match(value):
        raise ValueError(f"invalid packageName: {value}")
    return value


def validate_version(value: str) -> str:
    if not EXACT_VERSION_RE.match(value):
        raise ValueError(f"invalid version: {value}")
    return value


def normalize_entry(
    raw: dict[str, Any],
    args: argparse.Namespace,
    allow_infer_targets: bool,
) -> dict[str, Any]:
    source = raw.get("source")
    path_value = raw.get("path")
    if source is not None and path_value is not None:
        raise ValueError("entry cannot contain both path and source")

    if path_value is not None:
        directory = Path(str(path_value)).expanduser().resolve()
        inspection = inspect_local_skill(directory)
        infer_targets = allow_infer_targets or bool(raw.get("inferTarget"))
        package_name = raw.get("packageName") or inspection.get("packageName")
        version = raw.get("version") or inspection.get("version")
        if not package_name and infer_targets:
            package_name = f"{args.package_scope}{slugify_package_name(directory.name)}"
        if not version and infer_targets:
            version = args.default_version
        if not package_name:
            raise ValueError(f"{directory}: packageName is required")
        if not version:
            raise ValueError(f"{directory}: version is required")

        display_name = raw.get("displayName") or inspection.get("displayName")
        description = raw.get("description") or inspection.get("description")
        if not display_name:
            raise ValueError(f"{directory}: displayName is required")
        if not description:
            raise ValueError(f"{directory}: description is required")

        publish_registry = (
            raw.get("publishRegistry")
            or inspection.get("publishRegistry")
            or args.publish_registry
            or args.npm_registry
        )
        npm_cwd = raw.get("npmCwd") or str(directory)
        return {
            "label": str(directory),
            "source": {
                "type": "local",
                "location": str(directory),
            },
            "target": {
                "packageName": validate_package_name(str(package_name)),
                "version": validate_version(str(version)),
            },
            "metadata": {
                "displayName": str(display_name),
                "description": str(description),
                "sourceType": str(raw.get("sourceType") or inspection.get("sourceType") or args.source_type),
            },
            "publishRegistry": str(publish_registry) if publish_registry else None,
            "npmCwd": str(Path(str(npm_cwd)).expanduser().resolve()),
        }

    if not isinstance(source, dict):
        raise ValueError("entry must contain either path or source")
    source_type = source.get("type")
    if source_type not in {"git", "platform_authoring"}:
        raise ValueError("source.type must be git or platform_authoring when path is not used")

    package_name = raw.get("packageName")
    version = raw.get("version")
    display_name = raw.get("displayName")
    description = raw.get("description")
    if not package_name or not version or not display_name or not description:
        raise ValueError("non-local entries require packageName, version, displayName, and description")

    publish_registry = raw.get("publishRegistry") or args.publish_registry or args.npm_registry
    npm_cwd = raw.get("npmCwd") or os.getcwd()
    normalized_source = {"type": source_type}
    for key in ("location", "ref", "files"):
        if key in source:
            normalized_source[key] = source[key]

    return {
        "label": str(source.get("location") or source_type),
        "source": normalized_source,
        "target": {
            "packageName": validate_package_name(str(package_name)),
            "version": validate_version(str(version)),
        },
        "metadata": {
            "displayName": str(display_name),
            "description": str(description),
            "sourceType": str(raw.get("sourceType") or args.source_type),
        },
        "publishRegistry": str(publish_registry) if publish_registry else None,
        "npmCwd": str(Path(str(npm_cwd)).expanduser().resolve()),
    }


def resolve_entries(args: argparse.Namespace) -> list[dict[str, Any]]:
    entries: list[dict[str, Any]] = []
    if args.manifest:
        _, manifest_entries = load_manifest(Path(args.manifest).expanduser().resolve())
        for raw in manifest_entries:
            entries.append(normalize_entry(raw, args, allow_infer_targets=args.infer_targets))
    if args.root_dir:
        scanned_entries = scan_root_dir(Path(args.root_dir).expanduser().resolve(), args.recursive)
        for raw in scanned_entries:
            entries.append(normalize_entry(raw, args, allow_infer_targets=True))
    if not entries:
        raise ValueError("no skills found to import")
    return entries


def call_prepare(entry: dict[str, Any], args: argparse.Namespace, headers: dict[str, str]) -> dict[str, Any]:
    options: dict[str, Any] = {}
    if entry["publishRegistry"]:
        options["publishRegistry"] = entry["publishRegistry"]
    return request_json(
        "POST",
        f"{args.catalog_url.rstrip('/')}/api/v1/skill-packaging/prepare",
        {
            "idempotencyKey": f"batch-import-{int(time.time() * 1000)}-{entry['target']['packageName']}",
            "source": entry["source"],
            "target": entry["target"],
            "metadata": entry["metadata"],
            "options": options,
        },
        headers,
    )


def call_validate(
    workspace_path: str,
    target: dict[str, Any],
    args: argparse.Namespace,
    headers: dict[str, str],
) -> dict[str, Any]:
    return request_json(
        "POST",
        f"{args.catalog_url.rstrip('/')}/api/v1/skill-packaging/validate",
        {
            "workspacePath": workspace_path,
            "target": target,
        },
        headers,
    )


def call_pack(
    workspace_path: str,
    target: dict[str, Any],
    args: argparse.Namespace,
    headers: dict[str, str],
) -> dict[str, Any]:
    return request_json(
        "POST",
        f"{args.catalog_url.rstrip('/')}/api/v1/skill-packaging/pack",
        {
            "workspacePath": workspace_path,
            "target": target,
            "includeBundle": True,
        },
        headers,
    )


def call_release_record(
    target: dict[str, Any],
    publish_registry: str | None,
    pack_payload: dict[str, Any],
    args: argparse.Namespace,
    headers: dict[str, str],
) -> None:
    package_name = urllib.parse.quote(target["packageName"], safe="")
    version = urllib.parse.quote(target["version"], safe="")
    body: dict[str, Any] = {
        "manifestDigest": pack_payload.get("manifestDigest"),
        "components": pack_payload.get("components") or [],
    }
    if publish_registry:
        body["publishRegistry"] = publish_registry
    request_json(
        "PUT",
        f"{args.catalog_url.rstrip('/')}/api/v1/skills/{package_name}/versions/{version}/release-record",
        body,
        headers,
    )


def materialize_bundle(pack_payload: dict[str, Any], target: dict[str, Any]) -> tuple[Path, tempfile.TemporaryDirectory[str]]:
    bundle_base64 = pack_payload.get("bundleBase64")
    if not isinstance(bundle_base64, str) or not bundle_base64:
        raise RuntimeError("packaging API did not return bundleBase64")
    temp_dir = tempfile.TemporaryDirectory(prefix="skills-batch-import-")
    bundle_name = Path(
        pack_payload.get("bundlePath")
        or f"{target['packageName'].replace('@', '').replace('/', '-')}-{target['version']}.tgz"
    ).name
    bundle_path = Path(temp_dir.name) / bundle_name
    bundle_path.write_bytes(base64.b64decode(bundle_base64))
    return bundle_path, temp_dir


def publish_bundle(bundle_path: Path, npm_cwd: str, npm_registry: str | None) -> None:
    command = ["npm", "publish", str(bundle_path)]
    if npm_registry:
        command.extend(["--registry", npm_registry])
    result = subprocess.run(
        command,
        cwd=npm_cwd,
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        stderr = result.stderr.strip()
        stdout = result.stdout.strip()
        message = stderr or stdout or "npm publish failed"
        raise RuntimeError(message)


def print_plan(entries: list[dict[str, Any]]) -> None:
    payload = []
    for entry in entries:
        payload.append(
            {
                "label": entry["label"],
                "source": entry["source"],
                "target": entry["target"],
                "metadata": entry["metadata"],
                "publishRegistry": entry["publishRegistry"],
                "npmCwd": entry["npmCwd"],
            }
        )
    print(json.dumps({"skills": payload}, ensure_ascii=False, indent=2))


def import_entry(
    index: int,
    total: int,
    entry: dict[str, Any],
    args: argparse.Namespace,
    headers: dict[str, str],
) -> dict[str, Any]:
    prefix = f"[{index}/{total}]"
    print(f"{prefix} prepare {entry['target']['packageName']} from {entry['label']}")
    prepare_payload = call_prepare(entry, args, headers)
    if prepare_payload.get("errors"):
        raise RuntimeError(f"prepare returned errors: {prepare_payload['errors']}")

    workspace_path = prepare_payload.get("workspacePath")
    if not isinstance(workspace_path, str) or not workspace_path:
        raise RuntimeError("prepare did not return workspacePath")

    validate_payload = call_validate(workspace_path, entry["target"], args, headers)
    if not validate_payload.get("ok"):
        raise RuntimeError(f"validate failed: {validate_payload.get('errors') or []}")

    pack_payload = call_pack(workspace_path, entry["target"], args, headers)
    if pack_payload.get("errors"):
        raise RuntimeError(f"pack returned errors: {pack_payload['errors']}")

    bundle_path, temp_dir = materialize_bundle(pack_payload, entry["target"])
    try:
        publish_bundle(bundle_path, entry["npmCwd"], entry["publishRegistry"] or args.npm_registry)
        call_release_record(entry["target"], entry["publishRegistry"], pack_payload, args, headers)
    finally:
        temp_dir.cleanup()

    manifest_digest = pack_payload.get("manifestDigest")
    components = pack_payload.get("components") or []
    print(f"{prefix} ok {entry['target']['packageName']}@{entry['target']['version']}")
    return {
        "packageName": entry["target"]["packageName"],
        "version": entry["target"]["version"],
        "manifestDigest": manifest_digest,
        "components": components,
    }


def main() -> int:
    args = parse_args()
    try:
        entries = resolve_entries(args)
    except Exception as error:
        print(f"error: {error}", file=sys.stderr)
        return 2

    if args.dry_run:
        print_plan(entries)
        return 0

    headers = build_headers(args)
    successes: list[dict[str, Any]] = []
    failures: list[dict[str, str]] = []
    total = len(entries)

    for index, entry in enumerate(entries, start=1):
        try:
            successes.append(import_entry(index, total, entry, args, headers))
        except Exception as error:
            message = str(error)
            failures.append(
                {
                    "label": entry["label"],
                    "packageName": entry["target"]["packageName"],
                    "version": entry["target"]["version"],
                    "error": message,
                }
            )
            print(
                f"[{index}/{total}] failed {entry['target']['packageName']}@{entry['target']['version']}: {message}",
                file=sys.stderr,
            )
            if not args.continue_on_error:
                break

    summary = {
        "total": total,
        "succeeded": len(successes),
        "failed": len(failures),
        "successes": successes,
        "failures": failures,
    }
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if not failures else 1


if __name__ == "__main__":
    sys.exit(main())
