import { afterEach, describe, expect, it, vi } from 'vitest';
import { fetchAuditData, track } from './telemetry.ts';

describe('offline telemetry defaults', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  it('does not send telemetry when no endpoint is configured', () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch');

    track({ event: 'find', query: 'pdf', resultCount: '0' });

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('does not perform audit lookups when no endpoint is configured', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch');

    await expect(fetchAuditData('local', ['pdf-skill'])).resolves.toBeNull();

    expect(fetchMock).not.toHaveBeenCalled();
  });
});
