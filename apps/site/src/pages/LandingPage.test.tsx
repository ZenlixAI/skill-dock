import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { LocaleProvider } from '../lib/i18n';
import { LandingPage } from './LandingPage';

afterEach(() => {
  cleanup();
});

describe('LandingPage', () => {
  it('introduces the catalog, the release path, and the quick start', async () => {
    const { container } = render(
      <MemoryRouter>
        <LandingPage />
      </MemoryRouter>
    );

    expect(
      screen.getByRole('heading', { name: 'Agent skills, with a release process.' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'A git ref is not a release.' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Publish, resolve, govern.' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Three pieces, one release path.' })
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Run it locally.' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Questions that come up.' })).toBeInTheDocument();

    expect(screen.getAllByText('docker compose up --build -d').length).toBeGreaterThan(0);
    expect(screen.getAllByText('skills publish ./my-skill').length).toBeGreaterThan(0);
    expect(screen.queryByRole('tabpanel', { name: 'Install' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('tab', { name: 'Install' }));
    expect(await screen.findByRole('tabpanel', { name: 'Install' })).toBeInTheDocument();

    expect(container.querySelectorAll('.faq-item')).toHaveLength(5);
    expect(screen.queryByText('MIT · Go · TypeScript')).not.toBeInTheDocument();
  });

  it('keeps the public copy focused on the product path', () => {
    render(
      <MemoryRouter>
        <LandingPage />
      </MemoryRouter>
    );

    expect(screen.queryByText(/one-for-one/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/skills\.sh/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/marketplace/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/coming soon/i)).not.toBeInTheDocument();
  });

  it('renders the Chinese version under /zh', async () => {
    render(
      <MemoryRouter initialEntries={['/zh']}>
        <LocaleProvider>
          <LandingPage />
        </LocaleProvider>
      </MemoryRouter>
    );

    expect(
      screen.getByRole('heading', { name: 'Agent 技能，也有发布流程。' })
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Git ref 不是 release。' })).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: '发布、解析、治理。' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: '三个部分，一条路径。' })
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '在本地运行。' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '常见问题。' })).toBeInTheDocument();

    expect(screen.getAllByText('docker compose up --build -d').length).toBeGreaterThan(0);
    expect(screen.getAllByText('skills publish ./my-skill').length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('tab', { name: '安装' }));
    expect(await screen.findByRole('tabpanel', { name: '安装' })).toBeInTheDocument();
  });
});
