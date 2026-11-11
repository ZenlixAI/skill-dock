import { en } from './content.en';
import type { SiteContent } from './content.en';
import { zh } from './content.zh';
import type { Locale } from './locale';

export type {
  ArchitectureNode,
  Capability,
  FaqItem,
  NavItem,
  QuickStartStep,
  ReleasePoint,
  SiteContent,
  WorkflowStep,
} from './content.en';

export const SITE_URL = (import.meta.env.VITE_SITE_URL || 'https://skill-dock.pages.dev').replace(/\/+$/, '');
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

export const HERO_COMMAND = 'docker compose up --build -d';

export const CONTENT: Record<Locale, SiteContent> = { en, zh };

export function getContent(locale: Locale): SiteContent {
  return CONTENT[locale] ?? CONTENT.en;
}
