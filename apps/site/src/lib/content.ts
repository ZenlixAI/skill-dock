import {
  Boxes,
  Database,
  FileCheck2,
  History,
  Package,
  Route,
  ScanLine,
  Server,
  ShieldCheck,
  TerminalSquare,
  Waypoints,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

export const REPO_URL = 'https://github.com/ZenlixAI/skill-dock';
export const DOCS_URL = `${REPO_URL}/tree/master/docs`;
export const LICENSE_URL = `${REPO_URL}/blob/master/LICENSE`;
export const ISSUES_URL = `${REPO_URL}/issues`;

export const AGENTS = [
  'Claude Code',
  'Codex',
  'Cursor',
  'OpenCode',
  'GitHub Copilot',
  'Gemini CLI',
  'Windsurf',
  'Cline',
  'Amp',
  'Goose',
  'Kiro',
  'Qwen Code',
  'Trae',
  'Zed',
  'Continue',
  'Roo',
  'Crush',
  'OpenHands',
  'Junie',
  'Mux',
] as const;

export interface ReleasePoint {
  title: string;
  body: string;
  icon: LucideIcon;
}

export const RELEASE_POINTS: ReleasePoint[] = [
  {
    icon: Package,
    title: 'Exact versions',
    body: 'Install 1.4.0, not whatever happens to be on main.',
  },
  {
    icon: ScanLine,
    title: 'Verified bytes',
    body: 'The CLI checks the tarball integrity and the SKILL.md digest before it writes files.',
  },
  {
    icon: FileCheck2,
    title: 'One release record',
    body: 'Published, deprecated, and yanked are catalog states, not npm dist-tags.',
  },
  {
    icon: History,
    title: 'An audit trail',
    body: 'Every lifecycle change stores the actor, reason, and time.',
  },
];

export interface WorkflowStep {
  id: 'publish' | 'install' | 'govern';
  label: string;
  title: string;
  body: string;
  command: string;
  secondary?: string;
  flow: string[];
}

export const WORKFLOWS: WorkflowStep[] = [
  {
    id: 'publish',
    label: 'Publish',
    title: 'Package the source, then write the release record.',
    body: 'The catalog validates SKILL.md, normalizes the source, builds the tarball, and publishes it to your npm registry. The version becomes installable only after the catalog writes its release record.',
    command: 'skills publish ./my-skill',
    secondary:
      'skills import-local ./my-skill --package-name @acme.skill/my-skill --version 1.4.0',
    flow: ['source', 'validate', 'tarball', 'npm', 'release record'],
  },
  {
    id: 'install',
    label: 'Install',
    title: 'Resolve a version, then verify it.',
    body: 'The CLI asks the catalog for an install target. The response pins the version, integrity, and manifest digest. The CLI downloads the tarball and checks all three before it writes to an agent directory.',
    command: 'skills add @acme.skill/my-skill --version 1.4.0',
    flow: ['catalog', 'install target', 'tarball', 'agent'],
  },
  {
    id: 'govern',
    label: 'Govern',
    title: 'Change access without moving bytes.',
    body: 'Retire a skill, deprecate a version, or yank it. The tarball stays in npm. The next install resolution follows the new policy.',
    command:
      'POST /api/v1/skills/@acme.skill%2Fmy-skill/versions/1.4.0/governance-changes',
    flow: ['reason', 'policy', 'audit', 'next resolve'],
  },
];

export interface ArchitectureNode {
  id: string;
  title: string;
  subtitle: string;
  detail: string;
  icon: LucideIcon;
}

export const ARCHITECTURE_NODES: ArchitectureNode[] = [
  {
    id: 'cli',
    title: 'skills CLI',
    subtitle: 'developer machine or CI',
    detail:
      'Resolves install targets, downloads tarballs, verifies integrity and the manifest digest, then writes skills into agent directories.',
    icon: TerminalSquare,
  },
  {
    id: 'catalog',
    title: 'Catalog',
    subtitle: 'one Go binary',
    detail:
      'Owns release records, install resolution, lifecycle state, and audit events. It never stores bundle bytes.',
    icon: Server,
  },
  {
    id: 'postgres',
    title: 'Postgres',
    subtitle: 'catalog state',
    detail:
      'Stores release records, governance changes, audit events, and idempotency state. SQLite is available for local evaluation.',
    icon: Database,
  },
  {
    id: 'npm',
    title: 'npm registry',
    subtitle: 'bundle storage',
    detail:
      'Stores tarballs and package metadata. Nexus, Artifactory, Verdaccio, and other npm-compatible registries work.',
    icon: Boxes,
  },
];

export interface Capability {
  title: string;
  body: string;
  icon: LucideIcon;
  accent: 'green' | 'amber' | 'blue';
  visual: 'versions' | 'resolve' | 'policy' | 'audit' | 'storage' | 'delivery';
}

export const CAPABILITIES: Capability[] = [
  {
    title: 'Release records',
    body: 'Each version is a package name and an exact semver. Published versions are immutable. The record carries integrity, manifest digest, and components.',
    icon: FileCheck2,
    accent: 'green',
    visual: 'versions',
  },
  {
    title: 'Install resolution',
    body: 'The server picks the latest resolvable version. It does not fall back to an older release or guess from dist-tags.',
    icon: Route,
    accent: 'blue',
    visual: 'resolve',
  },
  {
    title: 'Lifecycle',
    body: 'Retire a skill. Deprecate or yank a version. Set an install policy. Add a replacement recommendation.',
    icon: ShieldCheck,
    accent: 'amber',
    visual: 'policy',
  },
  {
    title: 'Audit',
    body: 'Governance changes and audit events record the actor, target, reason, and time.',
    icon: History,
    accent: 'green',
    visual: 'audit',
  },
  {
    title: 'npm storage',
    body: 'Bundles live in Nexus, Artifactory, Verdaccio, or another npm-compatible registry. Keep your retention, replication, and scanning.',
    icon: Boxes,
    accent: 'blue',
    visual: 'storage',
  },
  {
    title: 'Delivery',
    body: 'Fetch tarballs through the catalog proxy, or let clients go direct to the registry with their own npm credentials.',
    icon: Waypoints,
    accent: 'amber',
    visual: 'delivery',
  },
];

export interface FaqItem {
  question: string;
  answer: string;
}

export const FAQ: FaqItem[] = [
  {
    question: 'Does the catalog store the bundle?',
    answer:
      'No. It stores release metadata and resolves tarball URLs. The bundle bytes stay in your npm registry.',
  },
  {
    question: 'Which npm registries work?',
    answer:
      'A registry that supports the npm publish and package metadata APIs. The repo includes Verdaccio for local development. Production deployments usually point at Nexus, Artifactory, or another hosted npm repository.',
  },
  {
    question: 'Can developers keep installing from git?',
    answer:
      'Yes. skills add accepts a local path, a GitHub shorthand, or a git URL. skills sync refreshes those installs. Registry installs use skills update.',
  },
  {
    question: 'What does a yank do?',
    answer:
      'The tarball stays in npm. The catalog marks the version yanked and blocked. New installs fail with an explicit error, and the response can include a replacement recommendation.',
  },
  {
    question: 'Do I need Postgres?',
    answer:
      'Production uses Postgres. SQLite works for local evaluation and the default standalone binary.',
  },
  {
    question: 'Is there a web UI?',
    answer:
      'The catalog is API-first. Use the CLI, or call the API from your own internal tooling.',
  },
  {
    question: 'What license?',
    answer: 'MIT.',
  },
];

export const NAV_ITEMS = [
  { label: 'How it works', href: '/#how-it-works' },
  { label: 'Architecture', href: '/#architecture' },
  { label: 'Quick start', href: '/#quick-start' },
  { label: 'FAQ', href: '/#faq' },
] as const;

export const HERO_COMMAND = 'docker compose up --build -d';
