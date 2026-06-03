import type { ImportReport } from '../api/types';
import { StatusPill } from './StatusPill';

interface ImportReportCardProps {
  report: ImportReport;
}

export function ImportReportCard({ report }: ImportReportCardProps) {
  return (
    <div className="import-report">
      <div className="import-report__top">
        <StatusPill tone={report.normalized ? 'warn' : 'good'}>
          {report.normalized ? 'Normalized import' : 'Direct import'}
        </StatusPill>
        <StatusPill tone="accent">{report.strictMode ? 'strict' : 'relaxed'}</StatusPill>
      </div>
      <dl className="property-list">
        <div>
          <dt>source type</dt>
          <dd>{report.sourceType || 'unknown'}</dd>
        </div>
        <div>
          <dt>generated package.json</dt>
          <dd>{String(report.generatedPackageJson)}</dd>
        </div>
        <div>
          <dt>placeholder capability</dt>
          <dd>{String(report.generatedCapabilityPlaceholder)}</dd>
        </div>
        <div>
          <dt>package identity</dt>
          <dd>{report.rewrotePackageIdentity ? 'rewritten' : 'unchanged'}</dd>
        </div>
        <div>
          <dt>generated paths</dt>
          <dd>{report.generatedPaths?.join(', ') || 'none'}</dd>
        </div>
      </dl>
    </div>
  );
}
