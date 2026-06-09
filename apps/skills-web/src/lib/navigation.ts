import { AppWindow, Blocks, FileClock, Search, ShieldAlert, Sparkles, Workflow } from 'lucide-react';

export interface ShellNavItem {
  to: string;
  label: string;
  icon: typeof Sparkles;
  requiredAction?: string;
}

export const portalNavItems: readonly ShellNavItem[] = [
  { to: '/', label: 'Overview', icon: Sparkles },
  { to: '/catalog', label: 'Catalog', icon: Search },
];

export const consoleNavItems: readonly ShellNavItem[] = [
  { to: '/workbench', label: 'Console', icon: AppWindow },
  { to: '/workbench/import', label: 'Import', icon: Workflow, requiredAction: 'packaging.prepare' },
  { to: '/workbench/authoring', label: 'Authoring', icon: Blocks, requiredAction: 'packaging.prepare' },
  {
    to: '/workbench/governance',
    label: 'Governance',
    icon: ShieldAlert,
    requiredAction: 'catalog.governance.skill.apply',
  },
  { to: '/workbench/audit', label: 'Audit', icon: FileClock },
];
