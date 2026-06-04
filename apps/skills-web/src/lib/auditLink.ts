type AuditLinkFilters = {
  packageName?: string | null;
  version?: string | null;
  outcome?: string | null;
  actor?: string | null;
};

function normalizeFilter(value?: string | null) {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

export function buildAuditLink({ packageName, version, outcome, actor }: AuditLinkFilters) {
  const searchParams = new URLSearchParams();
  const normalizedPackageName = normalizeFilter(packageName);
  const normalizedVersion = normalizeFilter(version);
  const normalizedOutcome = normalizeFilter(outcome);
  const normalizedActor = normalizeFilter(actor);

  if (normalizedPackageName) {
    searchParams.set('package', normalizedPackageName);
  }
  if (normalizedVersion) {
    searchParams.set('version', normalizedVersion);
  }
  if (normalizedOutcome) {
    searchParams.set('outcome', normalizedOutcome);
  }
  if (normalizedActor) {
    searchParams.set('actor', normalizedActor);
  }

  const query = searchParams.toString();
  return query ? `/audit?${query}` : '/audit';
}
