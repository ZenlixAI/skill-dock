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
          {report.normalized ? '已归一化导入' : '直接导入'}
        </StatusPill>
        <StatusPill tone="accent">{report.strictMode ? '严格模式' : '宽松模式'}</StatusPill>
      </div>
      <div className="import-report__metrics">
        <article className="console-signal console-signal--compact">
          <span>包身份</span>
          <strong>{report.rewrotePackageIdentity ? '已重写' : '保持稳定'}</strong>
        </article>
        <article className="console-signal console-signal--compact">
          <span>产物</span>
          <strong>{report.generatedPaths?.length || 0}</strong>
        </article>
      </div>
      <dl className="property-list">
        <div>
          <dt>来源类型</dt>
          <dd>{report.sourceType || '未知'}</dd>
        </div>
        <div>
          <dt>是否生成 package.json</dt>
          <dd>{String(report.generatedPackageJson)}</dd>
        </div>
        <div>
          <dt>是否生成 capability 占位内容</dt>
          <dd>{String(report.generatedCapabilityPlaceholder)}</dd>
        </div>
        <div>
          <dt>包身份</dt>
          <dd>{report.rewrotePackageIdentity ? '已重写' : '未变更'}</dd>
        </div>
        <div>
          <dt>生成路径</dt>
          <dd>{report.generatedPaths?.join(', ') || '无'}</dd>
        </div>
      </dl>
    </div>
  );
}
