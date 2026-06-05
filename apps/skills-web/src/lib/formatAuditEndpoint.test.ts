import { describe, expect, it } from 'vitest';
import { formatAuditEndpoint } from './formatAuditEndpoint';

describe('formatAuditEndpoint', () => {
  it('decodes encoded package names in audit endpoints', () => {
    expect(
      formatAuditEndpoint('/api/v1/skills/%40company.skill%2Fdiagnose/versions/0.1.0/release-record')
    ).toBe('/api/v1/skills/@company.skill/diagnose/versions/0.1.0/release-record');
  });

  it('keeps malformed endpoints unchanged', () => {
    expect(formatAuditEndpoint('/api/v1/skills/%E0%A4%A')).toBe('/api/v1/skills/%E0%A4%A');
  });
});
