import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { InstallCommandPanel } from './InstallCommandPanel';

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe('InstallCommandPanel', () => {
  it('copies the generated install command', async () => {
    Object.defineProperty(window.navigator, 'clipboard', {
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
      configurable: true,
    });

    render(
      <InstallCommandPanel
        title="安装这个技能"
        packageName="@platform.skill/agent-notes"
        version="1.2.3"
        npmRegistry="http://192.168.10.127:32081/repository/skills-npm/"
        installPolicy="resolvable"
        tarballUrl="https://npm.internal.example.com/@platform.skill/agent-notes/-/agent-notes-1.2.3.tgz"
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '复制安装命令' }));

    await waitFor(() =>
      expect(window.navigator.clipboard.writeText).toHaveBeenCalledWith(
        'SKILLS_REGISTRY_TOKEN=reader-token SKILLS_REGISTRY_URL=http://192.168.10.127:32081 npx --registry $SKILLS_REGISTRY_URL/repository/skills-npm/ skills add @platform.skill/agent-notes --version 1.2.3'
      )
    );
    expect(screen.getByDisplayValue(/SKILLS_REGISTRY_TOKEN=reader-token/)).toBeInTheDocument();
    expect(screen.getByText('已复制')).toBeInTheDocument();
    expect(screen.getByText('制品包地址')).toBeInTheDocument();
    expect(
      screen.getByText('http://192.168.10.127:32081/@platform.skill/agent-notes/-/agent-notes-1.2.3.tgz')
    ).toBeInTheDocument();
    expect(screen.getByText('完整性校验')).toBeInTheDocument();
    expect(screen.getByText('可直接安装')).toBeInTheDocument();
    expect(screen.queryByText('http://192.168.10.127:32081/repository/skills-npm/')).not.toBeInTheDocument();
  });

  it('shows localized failure feedback when copying the install command fails', async () => {
    Object.defineProperty(window.navigator, 'clipboard', {
      value: { writeText: vi.fn().mockRejectedValue(new Error('boom')) },
      configurable: true,
    });

    render(
      <InstallCommandPanel
        title="安装这个技能"
        packageName="@platform.skill/agent-notes"
        version="1.2.3"
        npmRegistry="http://192.168.10.127:32081/repository/skills-npm/"
        installPolicy="resolvable"
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '复制安装命令' }));

    await waitFor(() =>
      expect(screen.getByText('复制失败，请手动复制命令。')).toBeInTheDocument()
    );
  });

  it('disables copying and shows a blocked install warning', () => {
    render(
      <InstallCommandPanel
        title="安装这个版本"
        packageName="@platform.skill/agent-notes"
        version="1.2.3"
        npmRegistry="http://192.168.10.127:32081/repository/skills-npm/"
        installPolicy="blocked"
      />
    );

    expect(screen.getByRole('button', { name: '复制安装命令' })).toBeDisabled();
    expect(screen.getByText('当前版本已被阻断，不应继续作为安装目标。')).toBeInTheDocument();
  });

  it('shows Chinese fallback copy when install metadata is missing', () => {
    render(
      <InstallCommandPanel
        title="安装这个版本"
        packageName="@platform.skill/agent-notes"
        version="1.2.3"
        npmRegistry=""
        installPolicy="catalog_only"
      />
    );

    expect(screen.getAllByText('待补充仓库 Host').length).toBeGreaterThan(0);
    expect(
      screen.getByDisplayValue(
        'SKILLS_REGISTRY_TOKEN=reader-token SKILLS_REGISTRY_URL=<待补充仓库 Host> npx --registry $SKILLS_REGISTRY_URL/repository/skills-npm/ skills add @platform.skill/agent-notes --version 1.2.3'
      )
    ).toBeInTheDocument();
    expect(screen.getByText('当前未返回制品包地址。')).toBeInTheDocument();
    expect(screen.getByText('当前未返回完整性校验值。')).toBeInTheDocument();
    expect(screen.getByText('仅目录可见')).toBeInTheDocument();
  });
});
