import { useState } from 'react';
import type { InstallPolicy } from '../api/types';
import { buildInstallCommand, getInstallPolicyLabel, getInstallRegistryLabel } from '../lib/installCommand';
import { StatusPill } from './StatusPill';
import { SurfaceCard } from './SurfaceCard';

interface InstallCommandPanelProps {
  title: string;
  packageName: string;
  version: string;
  npmRegistry: string;
  installPolicy: InstallPolicy;
  tarballUrl?: string;
  integrity?: string;
}

function getInstallTone(installPolicy: InstallPolicy) {
  if (installPolicy === 'resolvable') {
    return 'good';
  }

  if (installPolicy === 'catalog_only') {
    return 'warn';
  }

  return 'bad';
}

function getInstallLabel(installPolicy: InstallPolicy) {
  return getInstallPolicyLabel(installPolicy);
}

function getInstallGuidance(installPolicy: InstallPolicy) {
  if (installPolicy === 'resolvable') {
    return '当前版本可直接作为安装目标，复制命令后即可在你自己的终端执行。';
  }

  if (installPolicy === 'catalog_only') {
    return '当前版本保留在目录中用于审阅，但不建议继续作为默认安装目标。';
  }

  return '当前版本已被阻断，不应继续作为安装目标。';
}

export function InstallCommandPanel({
  title,
  packageName,
  version,
  npmRegistry,
  installPolicy,
  tarballUrl,
  integrity,
}: InstallCommandPanelProps) {
  const [copyState, setCopyState] = useState<'idle' | 'copied' | 'failed'>('idle');
  const registryLabel = getInstallRegistryLabel(npmRegistry);
  const command = buildInstallCommand({
    packageName,
    version,
    npmRegistry: registryLabel,
  });
  const installBlocked = installPolicy === 'blocked';

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(command);
      setCopyState('copied');
    } catch {
      setCopyState('failed');
    }
  }

  return (
    <SurfaceCard
      action={<StatusPill tone={getInstallTone(installPolicy)}>{getInstallLabel(installPolicy)}</StatusPill>}
      className="install-command-panel-card"
      description={getInstallGuidance(installPolicy)}
      title={title}
      variant="hero"
    >
      <div className="install-command-panel">
        <div className="pill-row">
          <StatusPill>{`${packageName}@${version}`}</StatusPill>
          <StatusPill>{registryLabel}</StatusPill>
        </div>
        <code>{command}</code>
        <dl className="install-command-panel__meta">
          <div>
            <dt>仓库地址</dt>
            <dd>{registryLabel}</dd>
          </div>
          <div>
            <dt>制品包地址</dt>
            <dd>{tarballUrl || '当前未返回制品包地址。'}</dd>
          </div>
          <div>
            <dt>完整性校验</dt>
            <dd>{integrity || '当前未返回完整性校验值。'}</dd>
          </div>
        </dl>
        <div className="list-card__actions">
          <button className="button button--primary" disabled={installBlocked} onClick={() => void handleCopy()} type="button">
            复制安装命令
          </button>
          {copyState === 'copied' ? (
            <span aria-live="polite" className="install-command-panel__feedback install-command-panel__feedback--success">
              已复制
            </span>
          ) : null}
          {copyState === 'failed' ? (
            <span aria-live="polite" className="install-command-panel__feedback install-command-panel__feedback--error">
              复制失败，请手动复制命令。
            </span>
          ) : null}
        </div>
      </div>
    </SurfaceCard>
  );
}
