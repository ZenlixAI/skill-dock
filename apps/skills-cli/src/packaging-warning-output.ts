import type { PackagingImportReport, PackagingWarningDetail } from './catalog.ts';

function fallbackWarningDetail(code: string): PackagingWarningDetail {
  return {
    code,
    title: code,
    message: code,
  };
}

export function formatPackagingWarnings(
  warnings: string[] = [],
  warningDetails: PackagingWarningDetail[] = []
): PackagingWarningDetail[] {
  if (warnings.length === 0) {
    return [];
  }

  const byCode = new Map(warningDetails.map((detail) => [detail.code, detail]));
  return warnings.map((code) => byCode.get(code) ?? fallbackWarningDetail(code));
}

export function formatImportReportSummary(report?: PackagingImportReport): string[] {
  if (!report) {
    return [];
  }

  const lines: string[] = [];
  const status = report.normalized ? 'normalized import' : 'no normalization needed';
  lines.push(`status: ${status}`);

  if (report.sourceType) {
    lines.push(`source_type: ${report.sourceType}`);
  }
  lines.push(`strict_mode: ${report.strictMode ? 'true' : 'false'}`);

  if (report.generatedPaths && report.generatedPaths.length > 0) {
    lines.push(`generated_paths: ${report.generatedPaths.join(', ')}`);
  }
  if (report.rewrotePackageIdentity) {
    lines.push('package_identity: rewritten');
  }
  if (report.warningCodes && report.warningCodes.length > 0) {
    lines.push(`warning_codes: ${report.warningCodes.join(', ')}`);
  }
  return lines;
}
