# skills

> This directory is the `skills` CLI app inside the `skill_registry` monorepo. It is subtree-managed from `XmchxUp/skills`; keep monorepo-only changes minimal and documented.

The CLI for the open agent skills ecosystem.

<!-- agent-list:start -->

Supports **OpenCode**, **Claude Code**, **Codex**, **Cursor**, and [51 more](#supported-agents).

<!-- agent-list:end -->

[![skills.sh](https://skills.sh/b/vercel-labs/skills)](https://skills.sh/vercel-labs/skills)

## Install a Skill

```bash
npx skills add @platform.skill/pdf-skill
```

The CLI is registry-first. `skills add` only accepts a `package_name`.
Short alias: `npx skills a <package_name>`.

```bash
# Exact package install
npx skills add @platform.skill/pdf-skill

# Exact version
npx skills add @platform.skill/pdf-skill --version 1.2.0

# Temporary catalog override
npx skills add @platform.skill/pdf-skill --registry https://catalog.example.com
```

### Options

| Option                    | Description                                                                                                                                        |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `-g, --global`            | Install to user directory instead of project                                                                                                       |
| `-a, --agent <agents...>` | <!-- agent-names:start -->Target specific agents (e.g., `claude-code`, `codex`). See [Supported Agents](#supported-agents)<!-- agent-names:end --> |
| `--copy`                  | Copy files instead of symlinking to agent directories                                                                                              |
| `-y, --yes`               | Skip all confirmation prompts                                                                                                                      |
| `--verbose`               | Show per-agent install results instead of a compact summary                                                                                        |
| `--version <version>`     | Install an exact published version                                                                                                                 |
| `--registry <url>`        | Temporary catalog base URL override (default env: `SKILLS_REGISTRY_URL`)                                                                           |

### Examples

```bash
# Install to specific agents
npx skills add @platform.skill/pdf-skill -a claude-code -a opencode

# Non-interactive installation (CI/CD friendly)
npx skills add @platform.skill/pdf-skill -g -a claude-code -y
```

When `-y` is used without `--agent`, the CLI skips prompts and uses detected agents when available; if none are detected, it falls back to the universal `.agents/skills` agents instead of broadcasting to every known adapter.

### Catalog Auth

When talking to a private Catalog, the CLI can attach auth headers from env:

- `SKILLS_REGISTRY_TOKEN` -> `Authorization: Bearer <token>`
- `SKILLS_REGISTRY_ROLE` -> `X-Role: <role>` for local/dev registries such as the current Go prototype
- `skills publish` still requires publisher-capable credentials because it ends with `release-record`

### Installation Scope

| Scope       | Flag      | Location            | Use Case                                      |
| ----------- | --------- | ------------------- | --------------------------------------------- |
| **Project** | (default) | `./<agent>/skills/` | Committed with your project, shared with team |
| **Global**  | `-g`      | `~/<agent>/skills/` | Available across all projects                 |

### Installation Methods

When installing interactively, you can choose:

| Method                    | Description                                                                                 |
| ------------------------- | ------------------------------------------------------------------------------------------- |
| **Symlink** (Recommended) | Creates symlinks from each agent to a canonical copy. Single source of truth, easy updates. |
| **Copy**                  | Creates independent copies for each agent. Use when symlinks aren't supported.              |

## Other Commands

| Command                      | Description                                   |
| ---------------------------- | --------------------------------------------- |
| `npx skills add <package>`   | Add a registry skill package (alias: `a`)     |
| `npx skills list`            | List installed skills (alias: `ls`)           |
| `npx skills find [query]`    | Search for skills interactively or by keyword |
| `npx skills remove [skills]` | Remove installed skills from agents (alias: `rm`) |
| `npx skills update [skills]` | Update installed skills to latest versions    |
| `npx skills init [name]`     | Create a new skill scaffold                   |
| `npx skills inspect [path]`  | Validate a local skill directory              |
| `npx skills publish [path]`  | Validate, pack, and publish a local skill     |

### `skills list`

List all installed skills. Similar to `npm ls`.

```bash
# List all installed skills (project and global)
npx skills list

# List only global skills
npx skills ls -g

# Filter by specific agents
npx skills ls -a claude-code -a cursor
```

### `skills find`

Search for skills interactively or by keyword.

```bash
# Interactive search (fzf-style)
npx skills find

# Search by keyword
npx skills find typescript

# Machine-readable query output
npx skills find typescript --json
```

### `skills update`

```bash
# Update all skills (interactive scope prompt)
npx skills update

# Update a single installed package
npx skills update @platform.skill/pdf-skill

# Update multiple installed packages
npx skills update @platform.skill/pdf-skill @platform.skill/another-skill

# Update only global or project skills
npx skills update -g
npx skills update -p

# Non-interactive (auto-detects scope: project if in a project, else global)
npx skills update -y
```

| Option          | Description                                                               |
| --------------- | ------------------------------------------------------------------------- |
| `-g, --global`  | Only update global skills                                                 |
| `-p, --project` | Only update project skills                                                |
| `-y, --yes`     | Skip scope prompt (auto-detect: project if in a project dir, else global) |
| `[skills...]`   | Update specific skills by name instead of all                             |

### `skills init`

```bash
# Create SKILL.md in current directory
npx skills init

# Create a new skill in a subdirectory
npx skills init my-skill
```

Creates:

- `SKILL.md`
- `package.json`
- `commands/`

Default package identity:

- `@platform.skill/<skill-name>`

### `skills inspect`

```bash
# Validate the current directory
npx skills inspect

# Validate a specific path
npx skills inspect ./my-skill

# Use a specific catalog entry point for packaging validation
npx skills inspect ./my-skill --registry https://catalog.example.com
```

Without registry configuration, `inspect` uses the local authoring validator.
When `SKILLS_REGISTRY_URL` or `--registry <url>` is provided and the local directory
already has a resolvable `package.json` identity, it upgrades to the registry-aligned
packaging flow:

1. `POST /api/v1/skill-packaging/prepare`
2. `POST /api/v1/skill-packaging/validate`

| Option             | Description                                            |
| ------------------ | ------------------------------------------------------ |
| `--registry <url>` | Catalog base URL override for packaging validation     |

### `skills publish`

```bash
# Publish the current directory
npx skills publish

# Publish a specific path
npx skills publish ./my-skill

# Override the catalog entry point used for packaging and release record
npx skills publish ./my-skill --registry https://catalog.example.com
```

The publish command follows the registry-first flow:

1. `POST /api/v1/skill-packaging/prepare`
2. `POST /api/v1/skill-packaging/validate`
3. `POST /api/v1/skill-packaging/pack`
4. `npm publish <bundle>`
5. `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`

| Option             | Description                                                   |
| ------------------ | ------------------------------------------------------------- |
| `--registry <url>` | Catalog base URL override for packaging and release-record    |

### `skills remove`

Remove installed skills from agents.

```bash
# Remove interactively (select from installed skills)
npx skills remove

# Remove specific skill by name
npx skills remove web-design-guidelines

# Remove multiple skills
npx skills remove frontend-design web-design-guidelines

# Remove from global scope
npx skills remove --global web-design-guidelines

# Remove from specific agents only
npx skills remove --agent claude-code cursor my-skill

# Remove all installed skills without confirmation
npx skills remove --all

# Remove all skills from a specific agent
npx skills remove --skill '*' -a cursor

# Remove a specific skill from all agents
npx skills remove my-skill --agent '*'

# Use 'rm' alias
npx skills rm my-skill
```

| Option         | Description                                      |
| -------------- | ------------------------------------------------ |
| `-g, --global` | Remove from global scope (~/) instead of project |
| `-a, --agent`  | Remove from specific agents (use `'*'` for all)  |
| `-s, --skill`  | Specify skills to remove (use `'*'` for all)     |
| `-y, --yes`    | Skip confirmation prompts                        |
| `--all`        | Shorthand for `--skill '*' --agent '*' -y`       |

## What are Agent Skills?

Agent skills are reusable instruction sets that extend your coding agent's capabilities. They're defined in `SKILL.md`
files with YAML frontmatter containing a `name` and `description`.

Skills let agents perform specialized tasks like:

- Generating release notes from git history
- Creating PRs following your team's conventions
- Integrating with external tools (Linear, Notion, etc.)

Discover skills at **[skills.sh](https://skills.sh)**

## Supported Agents

Skills can be installed to any of these agents:

<!-- supported-agents:start -->

| Agent                                 | `--agent`                                | Project Path             | Global Path                     |
| ------------------------------------- | ---------------------------------------- | ------------------------ | ------------------------------- |
| AiderDesk                             | `aider-desk`                             | `.aider-desk/skills/`    | `~/.aider-desk/skills/`         |
| Amp, Kimi Code CLI, Replit, Universal | `amp`, `kimi-cli`, `replit`, `universal` | `.agents/skills/`        | `~/.config/agents/skills/`      |
| Antigravity                           | `antigravity`                            | `.agents/skills/`        | `~/.gemini/antigravity/skills/` |
| Augment                               | `augment`                                | `.augment/skills/`       | `~/.augment/skills/`            |
| IBM Bob                               | `bob`                                    | `.bob/skills/`           | `~/.bob/skills/`                |
| Claude Code                           | `claude-code`                            | `.claude/skills/`        | `~/.claude/skills/`             |
| OpenClaw                              | `openclaw`                               | `skills/`                | `~/.openclaw/skills/`           |
| Cline, Dexto, Warp                    | `cline`, `dexto`, `warp`                 | `.agents/skills/`        | `~/.agents/skills/`             |
| CodeArts Agent                        | `codearts-agent`                         | `.codeartsdoer/skills/`  | `~/.codeartsdoer/skills/`       |
| CodeBuddy                             | `codebuddy`                              | `.codebuddy/skills/`     | `~/.codebuddy/skills/`          |
| Codemaker                             | `codemaker`                              | `.codemaker/skills/`     | `~/.codemaker/skills/`          |
| Code Studio                           | `codestudio`                             | `.codestudio/skills/`    | `~/.codestudio/skills/`         |
| Codex                                 | `codex`                                  | `.agents/skills/`        | `~/.codex/skills/`              |
| Command Code                          | `command-code`                           | `.commandcode/skills/`   | `~/.commandcode/skills/`        |
| Continue                              | `continue`                               | `.continue/skills/`      | `~/.continue/skills/`           |
| Cortex Code                           | `cortex`                                 | `.cortex/skills/`        | `~/.snowflake/cortex/skills/`   |
| Crush                                 | `crush`                                  | `.crush/skills/`         | `~/.config/crush/skills/`       |
| Cursor                                | `cursor`                                 | `.agents/skills/`        | `~/.cursor/skills/`             |
| Deep Agents                           | `deepagents`                             | `.agents/skills/`        | `~/.deepagents/agent/skills/`   |
| Devin for Terminal                    | `devin`                                  | `.devin/skills/`         | `~/.config/devin/skills/`       |
| Droid                                 | `droid`                                  | `.factory/skills/`       | `~/.factory/skills/`            |
| Firebender                            | `firebender`                             | `.agents/skills/`        | `~/.firebender/skills/`         |
| ForgeCode                             | `forgecode`                              | `.forge/skills/`         | `~/.forge/skills/`              |
| Gemini CLI                            | `gemini-cli`                             | `.agents/skills/`        | `~/.gemini/skills/`             |
| GitHub Copilot                        | `github-copilot`                         | `.agents/skills/`        | `~/.copilot/skills/`            |
| Goose                                 | `goose`                                  | `.goose/skills/`         | `~/.config/goose/skills/`       |
| Hermes Agent                          | `hermes-agent`                           | `.hermes/skills/`        | `~/.hermes/skills/`             |
| Junie                                 | `junie`                                  | `.junie/skills/`         | `~/.junie/skills/`              |
| iFlow CLI                             | `iflow-cli`                              | `.iflow/skills/`         | `~/.iflow/skills/`              |
| Kilo Code                             | `kilo`                                   | `.kilocode/skills/`      | `~/.kilocode/skills/`           |
| Kiro CLI                              | `kiro-cli`                               | `.kiro/skills/`          | `~/.kiro/skills/`               |
| Kode                                  | `kode`                                   | `.kode/skills/`          | `~/.kode/skills/`               |
| MCPJam                                | `mcpjam`                                 | `.mcpjam/skills/`        | `~/.mcpjam/skills/`             |
| Mistral Vibe                          | `mistral-vibe`                           | `.vibe/skills/`          | `~/.vibe/skills/`               |
| Mux                                   | `mux`                                    | `.mux/skills/`           | `~/.mux/skills/`                |
| OpenCode                              | `opencode`                               | `.agents/skills/`        | `~/.config/opencode/skills/`    |
| OpenHands                             | `openhands`                              | `.openhands/skills/`     | `~/.openhands/skills/`          |
| Pi                                    | `pi`                                     | `.pi/skills/`            | `~/.pi/agent/skills/`           |
| Qoder                                 | `qoder`                                  | `.qoder/skills/`         | `~/.qoder/skills/`              |
| Qwen Code                             | `qwen-code`                              | `.qwen/skills/`          | `~/.qwen/skills/`               |
| Rovo Dev                              | `rovodev`                                | `.rovodev/skills/`       | `~/.rovodev/skills/`            |
| Roo Code                              | `roo`                                    | `.roo/skills/`           | `~/.roo/skills/`                |
| Tabnine CLI                           | `tabnine-cli`                            | `.tabnine/agent/skills/` | `~/.tabnine/agent/skills/`      |
| Trae                                  | `trae`                                   | `.trae/skills/`          | `~/.trae/skills/`               |
| Trae CN                               | `trae-cn`                                | `.trae/skills/`          | `~/.trae-cn/skills/`            |
| Windsurf                              | `windsurf`                               | `.windsurf/skills/`      | `~/.codeium/windsurf/skills/`   |
| Zencoder                              | `zencoder`                               | `.zencoder/skills/`      | `~/.zencoder/skills/`           |
| Neovate                               | `neovate`                                | `.neovate/skills/`       | `~/.neovate/skills/`            |
| Pochi                                 | `pochi`                                  | `.pochi/skills/`         | `~/.pochi/skills/`              |
| AdaL                                  | `adal`                                   | `.adal/skills/`          | `~/.adal/skills/`               |

<!-- supported-agents:end -->

> [!NOTE]
> **Kiro CLI users:** The default agent automatically loads skills from `.kiro/skills/` and `~/.kiro/skills/` — no
> configuration needed. If you use a **custom agent**, add skills to its `resources` in `.kiro/agents/<agent>.json`:
>
> ```json
> {
>   "resources": ["skill://.kiro/skills/**/SKILL.md"]
> }
> ```

The CLI automatically detects which coding agents you have installed. If none are detected, you'll be prompted to select
which agents to install to.

## Creating Skills

Skills are directories containing a `SKILL.md` file with YAML frontmatter:

```markdown
---
name: my-skill
description: What this skill does and when to use it
---

# My Skill

Instructions for the agent to follow when this skill is activated.

## When to Use

Describe the scenarios where this skill should be used.

## Steps

1. First, do this
2. Then, do that
```

### Required Fields

- `name`: Unique identifier (lowercase, hyphens allowed)
- `description`: Brief explanation of what the skill does

### Optional Fields

- `metadata.internal`: Set to `true` to hide the skill from normal discovery. Internal skills are only visible and
  installable when `INSTALL_INTERNAL_SKILLS=1` is set. Useful for work-in-progress skills or skills meant only for
  internal tooling.

```markdown
---
name: my-internal-skill
description: An internal skill not shown by default
metadata:
  internal: true
---
```

### Skill Discovery

The CLI searches for skills in these locations within a repository. Each
skill container directory is walked one level deep for the common flat
layout (`skills/<name>/SKILL.md`) and one extra level deep for catalog
layouts (`skills/<category>/<name>/SKILL.md`). A `SKILL.md` discovered at
the shallower level shadows anything nested below it. Use `--full-depth`
to also discover `SKILL.md` files outside these container directories
(e.g. under `examples/` or `tests/`).

<!-- skill-discovery:start -->

- Root directory (if it contains `SKILL.md`)
- `skills/`
- `skills/.curated/`
- `skills/.experimental/`
- `skills/.system/`
- `.aider-desk/skills/`
- `.agents/skills/`
- `.augment/skills/`
- `.bob/skills/`
- `.claude/skills/`
- `.codeartsdoer/skills/`
- `.codebuddy/skills/`
- `.codemaker/skills/`
- `.codestudio/skills/`
- `.commandcode/skills/`
- `.continue/skills/`
- `.cortex/skills/`
- `.crush/skills/`
- `.devin/skills/`
- `.factory/skills/`
- `.forge/skills/`
- `.goose/skills/`
- `.hermes/skills/`
- `.junie/skills/`
- `.iflow/skills/`
- `.kilocode/skills/`
- `.kiro/skills/`
- `.kode/skills/`
- `.mcpjam/skills/`
- `.vibe/skills/`
- `.mux/skills/`
- `.openhands/skills/`
- `.pi/skills/`
- `.qoder/skills/`
- `.qwen/skills/`
- `.rovodev/skills/`
- `.roo/skills/`
- `.tabnine/agent/skills/`
- `.trae/skills/`
- `.windsurf/skills/`
- `.zencoder/skills/`
- `.neovate/skills/`
- `.pochi/skills/`
- `.adal/skills/`
<!-- skill-discovery:end -->

### Plugin Manifest Discovery

If `.claude-plugin/marketplace.json` or `.claude-plugin/plugin.json` exists, skills declared in those files are also discovered:

```json
// .claude-plugin/marketplace.json
{
  "metadata": { "pluginRoot": "./plugins" },
  "plugins": [
    {
      "name": "my-plugin",
      "source": "my-plugin",
      "skills": ["./skills/review", "./skills/test"]
    }
  ]
}
```

This enables compatibility with the [Claude Code plugin marketplace](https://code.claude.com/docs/en/plugin-marketplaces) ecosystem. Skill paths declared in a manifest are searched at their declared depth and are not subject to the depth-2 catalog walk described above.

If no skills are found in standard locations, a recursive search is performed.

## Compatibility

Skills are generally compatible across agents since they follow a
shared [Agent Skills specification](https://agentskills.io). However, some features may be agent-specific:

| Feature         | OpenCode | OpenHands | Claude Code | Cline | CodeBuddy | Codex | Command Code | Kiro CLI | Cursor | Antigravity | Roo Code | Github Copilot | Amp | OpenClaw | Neovate | Pi  | Qoder | Zencoder |
| --------------- | -------- | --------- | ----------- | ----- | --------- | ----- | ------------ | -------- | ------ | ----------- | -------- | -------------- | --- | -------- | ------- | --- | ----- | -------- |
| Basic skills    | Yes      | Yes       | Yes         | Yes   | Yes       | Yes   | Yes          | Yes      | Yes    | Yes         | Yes      | Yes            | Yes | Yes      | Yes     | Yes | Yes   | Yes      |
| `allowed-tools` | Yes      | Yes       | Yes         | Yes   | Yes       | Yes   | Yes          | No       | Yes    | Yes         | Yes      | Yes            | Yes | Yes      | Yes     | Yes | Yes   | No       |
| `context: fork` | No       | No        | Yes         | No    | No        | No    | No           | No       | No     | No          | No       | No             | No  | No       | No      | No  | No    | No       |
| Hooks           | No       | No        | Yes         | Yes   | No        | No    | No           | Yes      | No     | No          | No       | No             | No  | No       | No      | No  | No    | No       |

## Troubleshooting

### "No skills found"

Ensure the repository contains valid `SKILL.md` files with both `name` and `description` in the frontmatter.

### Skill not loading in agent

- Verify the skill was installed to the correct path
- Check the agent's documentation for skill loading requirements
- Ensure the `SKILL.md` frontmatter is valid YAML

### Permission errors

Ensure you have write access to the target directory.

## Environment Variables

| Variable            | Description                              |
| ------------------- | ---------------------------------------- |
| `DISABLE_TELEMETRY` | Set to disable anonymous usage telemetry |
| `DO_NOT_TRACK`      | Alternative way to disable telemetry     |

## Telemetry

This CLI collects anonymous usage data to help improve the tool. No personal information is collected.

Telemetry is automatically disabled in CI environments.

## Related Links

- [Agent Skills Specification](https://agentskills.io)
- [Skills Directory](https://skills.sh)
- [Amp Skills Documentation](https://ampcode.com/manual#agent-skills)
- [Antigravity Skills Documentation](https://antigravity.google/docs/skills)
- [Factory AI / Droid Skills Documentation](https://docs.factory.ai/cli/configuration/skills)
- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
- [OpenClaw Skills Documentation](https://docs.openclaw.ai/tools/skills)
- [Cline Skills Documentation](https://docs.cline.bot/features/skills)
- [CodeBuddy Skills Documentation](https://www.codebuddy.ai/docs/ide/Features/Skills)
- [Codex Skills Documentation](https://developers.openai.com/codex/skills)
- [Command Code Skills Documentation](https://commandcode.ai/docs/skills)
- [Crush Skills Documentation](https://github.com/charmbracelet/crush?tab=readme-ov-file#agent-skills)
- [Cursor Skills Documentation](https://cursor.com/docs/context/skills)
- [Firebender Skills Documentation](https://docs.firebender.com/multi-agent/skills)
- [Gemini CLI Skills Documentation](https://geminicli.com/docs/cli/skills/)
- [GitHub Copilot Agent Skills](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills)
- [iFlow CLI Skills Documentation](https://platform.iflow.cn/en/cli/examples/skill)
- [Kimi Code CLI Skills Documentation](https://moonshotai.github.io/kimi-cli/en/customization/skills.html)
- [Kiro CLI Skills Documentation](https://kiro.dev/docs/cli/custom-agents/configuration-reference/#skill-resources)
- [Kode Skills Documentation](https://github.com/shareAI-lab/kode/blob/main/docs/skills.md)
- [OpenCode Skills Documentation](https://opencode.ai/docs/skills)
- [Qwen Code Skills Documentation](https://qwenlm.github.io/qwen-code-docs/en/users/features/skills/)
- [OpenHands Skills Documentation](https://docs.openhands.ai/modules/usage/how-to/using-skills)
- [Pi Skills Documentation](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/skills.md)
- [Qoder Skills Documentation](https://docs.qoder.com/cli/Skills)
- [Replit Skills Documentation](https://docs.replit.com/replitai/skills)
- [Roo Code Skills Documentation](https://docs.roocode.com/features/skills)
- [Trae Skills Documentation](https://docs.trae.ai/ide/skills)

## License

MIT
