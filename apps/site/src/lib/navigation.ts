export interface ShellNavItem {
  to: string;
  label: string;
}

export const portalNavItems: readonly ShellNavItem[] = [
  { to: '/', label: 'Home' },
  { to: '/features', label: 'Features' },
  { to: '/docs', label: 'Docs' },
];
