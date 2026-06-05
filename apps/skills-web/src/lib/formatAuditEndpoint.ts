export function formatAuditEndpoint(endpoint: string) {
  const trimmed = endpoint.trim();
  if (trimmed === '') {
    return '—';
  }

  try {
    const [path, query] = trimmed.split('?', 2);
    const decodedPath = decodeURIComponent(path);
    return query ? `${decodedPath}?${query}` : decodedPath;
  } catch {
    return trimmed;
  }
}
