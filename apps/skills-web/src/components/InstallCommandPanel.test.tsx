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
        npmRegistry="https://npm.internal.example.com"
        installPolicy="resolvable"
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '复制安装命令' }));

    await waitFor(() =>
      expect(window.navigator.clipboard.writeText).toHaveBeenCalledWith(
        'npm install @platform.skill/agent-notes@1.2.3 --registry=https://npm.internal.example.com'
      )
    );
    expect(screen.getByText('已复制')).toBeInTheDocument();
  });

  it('disables copying and shows a blocked install warning', () => {
    render(
      <InstallCommandPanel
        title="安装这个版本"
        packageName="@platform.skill/agent-notes"
        version="1.2.3"
        npmRegistry="https://npm.internal.example.com"
        installPolicy="blocked"
      />
    );

    expect(screen.getByRole('button', { name: '复制安装命令' })).toBeDisabled();
    expect(screen.getByText('当前版本已被阻断，不应继续作为安装目标。')).toBeInTheDocument();
  });
});
