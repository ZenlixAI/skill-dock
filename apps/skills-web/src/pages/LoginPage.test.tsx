import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import * as client from '../api/client';
import { SessionProvider } from '../session/SessionProvider';
import { LoginPage } from './LoginPage';

vi.mock('../api/client', () => ({
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  loginSession: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderLoginPage(initialEntry = '/login?next=%2Fcatalog') {
  return render(
    <SessionProvider>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/catalog" element={<div>catalog destination</div>} />
          <Route path="/login" element={<LoginPage />} />
        </Routes>
      </MemoryRouter>
    </SessionProvider>
  );
}

describe('LoginPage', () => {
  it('renders the console login context and form controls', async () => {
    vi.mocked(client.getSession).mockResolvedValue(null);

    renderLoginPage();

    expect(await screen.findByText('控制台登录')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: '进入技能注册表控制台' })).toBeInTheDocument();
    expect(screen.getByText('导入、编写、治理和审计流程统一收口到这个控制台入口。')).toBeInTheDocument();
    expect(screen.getByLabelText('用户名')).toBeInTheDocument();
    expect(screen.getByLabelText('密码')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument();
  });

  it('announces login errors inline when submission fails', async () => {
    vi.mocked(client.getSession).mockResolvedValue(null);
    vi.mocked(client.loginSession).mockRejectedValue(new Error('用户名或密码不正确'));

    renderLoginPage();

    fireEvent.change(await screen.findByLabelText('用户名'), { target: { value: 'reader' } });
    fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'wrong-secret' } });
    fireEvent.click(screen.getByRole('button', { name: '登录' }));

    const error = await screen.findByRole('status');
    expect(error).toHaveTextContent('用户名或密码不正确');
    expect(error).toHaveAttribute('aria-live', 'polite');
    expect(screen.getByRole('button', { name: '登录' })).not.toBeDisabled();
  });

  it('redirects to next after a successful login', async () => {
    vi.mocked(client.getSession).mockResolvedValue(null);
    vi.mocked(client.loginSession).mockResolvedValue({
      role: 'reader',
      actor: 'dev@company',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });

    renderLoginPage();

    fireEvent.change(await screen.findByLabelText('用户名'), { target: { value: 'reader' } });
    fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'reader-secret' } });
    fireEvent.click(screen.getByRole('button', { name: '登录' }));

    expect(await screen.findByText('catalog destination')).toBeInTheDocument();
  });
});
