export interface ReleaseHandoff {
  packageName: string;
  version: string;
  publishRegistry?: string;
  manifestDigest?: string;
  components?: string[];
}

export function buildReleaseHandoffPath(handoff: ReleaseHandoff, basePath?: string) {
  const path =
    basePath ||
    `/skills/${encodeURIComponent(handoff.packageName)}/versions/${encodeURIComponent(handoff.version)}`;
  const params = new URLSearchParams();

  if (handoff.publishRegistry) {
    params.set('publishRegistry', handoff.publishRegistry);
  }
  if (handoff.manifestDigest) {
    params.set('manifestDigest', handoff.manifestDigest);
  }
  if (handoff.components?.length) {
    params.set('components', handoff.components.join(','));
  }
  const query = params.toString();
  return query ? `${path}?${query}` : path;
}

export function readReleaseHandoff(searchParams: URLSearchParams, fallback?: { packageName?: string; version?: string }) {
  const packageName = fallback?.packageName || '';
  const version = fallback?.version || '';

  return {
    packageName,
    version,
    publishRegistry: searchParams.get('publishRegistry') || undefined,
    manifestDigest: searchParams.get('manifestDigest') || undefined,
    components: searchParams.get('components')?.split(',').map((item) => item.trim()).filter(Boolean) || [],
  };
}
