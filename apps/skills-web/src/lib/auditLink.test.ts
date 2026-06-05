import { describe, expect, it } from 'vitest';
import { buildAuditLink } from './auditLink';

describe('buildAuditLink', () => {
  it('returns the audit page without query params when no filters are provided', () => {
    expect(buildAuditLink({})).toBe('/workbench/audit');
  });

  it('includes only the provided filters in a stable order', () => {
    expect(
      buildAuditLink({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        outcome: 'failed',
        actor: 'svc-publisher',
      })
    ).toBe(
      '/workbench/audit?package=%40platform.skill%2Fpdf-skill&version=1.2.0&outcome=failed&actor=svc-publisher'
    );
  });

  it('omits blank values', () => {
    expect(
      buildAuditLink({
        packageName: '  ',
        version: '1.2.0',
        outcome: '',
      })
    ).toBe('/workbench/audit?version=1.2.0');
  });
});
