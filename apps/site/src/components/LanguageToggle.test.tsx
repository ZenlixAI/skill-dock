import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { LocaleProvider } from '../lib/i18n';
import { LanguageToggle } from './LanguageToggle';

afterEach(() => {
  cleanup();
});

function renderToggle(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <LocaleProvider>
        <LanguageToggle />
      </LocaleProvider>
    </MemoryRouter>
  );
}

describe('LanguageToggle', () => {
  it('links to the same page in Chinese and switches the interface', () => {
    renderToggle('/registry?view=trending#results');

    fireEvent.click(
      screen.getByRole('button', { name: 'Switch language, current: English' })
    );

    const chinese = screen.getByRole('menuitem', { name: '简体中文' });
    expect(chinese).toHaveAttribute('href', '/zh/registry?view=trending#results');

    fireEvent.click(chinese);

    expect(
      screen.getByRole('button', { name: '切换语言，当前：简体中文' })
    ).toBeInTheDocument();
    expect(screen.getByText('中文')).toBeInTheDocument();
  });

  it('links back to the English page from a Chinese route', () => {
    renderToggle('/zh/privacy');

    fireEvent.click(
      screen.getByRole('button', { name: '切换语言，当前：简体中文' })
    );

    expect(screen.getByRole('menuitem', { name: 'English' })).toHaveAttribute(
      'href',
      '/privacy'
    );
  });
});
