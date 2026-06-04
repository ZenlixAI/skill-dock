export function normalizeRegistryUrl(value?: string): string | undefined {
  if (typeof value !== 'string') {
    return undefined;
  }
  const trimmed = value.trim();
  if (trimmed === '') {
    return undefined;
  }
  return trimmed.replace(/\/+$/, '');
}
