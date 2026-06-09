import { AppWindow, Blocks, FileClock, Search, ShieldAlert, Sparkles, Workflow } from 'lucide-react';

export interface ShellNavItem {
  to: string;
  label: string;
  icon: typeof Sparkles;
  requiredAction?: string;
}

export const portalNavItems: readonly ShellNavItem[] = [
  { to: '/', label: '概览', icon: Sparkles },
  { to: '/catalog', label: '目录', icon: Search },
];

export const consoleNavItems: readonly ShellNavItem[] = [
  { to: '/workbench', label: '控制台', icon: AppWindow },
  { to: '/workbench/import', label: '导入', icon: Workflow, requiredAction: 'packaging.prepare' },
  { to: '/workbench/authoring', label: '编写', icon: Blocks, requiredAction: 'packaging.prepare' },
  {
    to: '/workbench/governance',
    label: '治理',
    icon: ShieldAlert,
    requiredAction: 'catalog.governance.skill.apply',
  },
  { to: '/workbench/audit', label: '审计', icon: FileClock },
];
