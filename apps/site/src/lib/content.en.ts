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
import type { SkillsShBrowseView } from './skills-directory-api';
import type { Locale } from './locale';

export interface NavItem {
  label: string;
  path: string;
}

export interface ReleasePoint {
  title: string;
  body: string;
  icon: LucideIcon;
}

export interface WorkflowStep {
  id: 'publish' | 'install' | 'govern';
  label: string;
  title: string;
  body: string;
  command: string;
  secondary?: string;
  flow: string[];
}

export interface ArchitectureNode {
  id: string;
  title: string;
  subtitle: string;
  detail: string;
  icon: LucideIcon;
}

export interface Capability {
  title: string;
  body: string;
  icon: LucideIcon;
  accent: 'green' | 'amber' | 'blue';
  visual: 'versions' | 'resolve' | 'policy' | 'audit' | 'storage' | 'delivery';
}

export interface FaqItem {
  question: string;
  answer: string;
}

export interface QuickStartStep {
  title: string;
  body: string;
  command: string;
}

export interface LanguageOption {
  id: Locale;
  label: string;
}

export interface LegalSection {
  heading: string;
  body: string;
}

export const en = {
  htmlLang: 'en',
  language: {
    label: 'Language',
    options: [
      { id: 'en', label: 'English' },
      { id: 'zh', label: '简体中文' },
    ] as LanguageOption[],
    switchLabel: (current: string) => `Switch language, current: ${current}`,
    current: 'Current language',
  },
  meta: {
    home: {
      title: 'Skill Dock · A self-hosted registry for agent skills',
      description:
        'Skill Dock is a self-hosted registry for agent skills. Store bundles in your npm registry, publish exact versions, resolve installs, and govern releases.',
    },
    registry: {
      title: 'Skills Directory · Skill Dock',
      description:
        'Browse the public skills.sh directory, compare install counts, and inspect a skill before installing it.',
    },
    privacy: {
      title: 'Privacy · Skill Dock',
      description: 'How the Skill Dock website handles site usage data.',
    },
    terms: {
      title: 'Terms · Skill Dock',
      description: 'Terms for using the Skill Dock website and related materials.',
    },
    notFound: {
      title: 'Page not found · Skill Dock',
      description: 'The requested page is not available. Return home or read the quick start.',
    },
  },
  shell: {
    brandTagline: 'agent skill registry',
    skipToContent: 'Skip to main content',
    homeLabel: 'Skill Dock home',
    primaryNavLabel: 'Primary',
    mobileNavLabel: 'Mobile',
    openMenu: 'Open menu',
    closeMenu: 'Close menu',
    githubLabel: 'Skill Dock on GitHub',
    quickStart: 'Quick start',
    docs: 'Docs',
    github: 'GitHub',
    nav: [
      { label: 'Registry', path: '/registry' },
      { label: 'How it works', path: '/#how-it-works' },
      { label: 'Architecture', path: '/#architecture' },
      { label: 'Quick start', path: '/#quick-start' },
      { label: 'FAQ', path: '/#faq' },
    ] as NavItem[],
    footer: {
      tagline: 'A self-hosted registry for agent skills.',
      productHeading: 'Product',
      projectHeading: 'Project',
      legalHeading: 'Legal',
      registry: 'Registry',
      howItWorks: 'How it works',
      architecture: 'Architecture',
      quickStart: 'Quick start',
      faq: 'FAQ',
      docs: 'Docs',
      license: 'License',
      privacy: 'Privacy',
      terms: 'Terms',
      openSource: 'Open source under MIT',
      copyright: '© 2026 Skill Dock',
    },
  },
  ui: {
    opensInNewTab: ' (opens in a new tab)',
    retry: 'Retry',
    theme: {
      label: 'Theme',
      system: 'System',
      light: 'Light',
      dark: 'Dark',
      trigger: (current: string) => `Theme: ${current}`,
      systemWith: (resolved: string) => `System (${resolved})`,
    },
    terminal: {
      copy: 'Copy',
      copied: 'Copied',
      copyCommand: (title: string) => `Copy ${title} command`,
      copiedCommand: 'Copied command',
      announcement: 'Command copied to clipboard.',
    },
  },
  hero: {
    eyebrow: 'Self-hosted · Open source',
    title: 'Agent skills, with a release process.',
    lede: 'Skill Dock is a self-hosted registry for agent skills. Bundles live in your npm registry. The catalog decides which release an agent gets and records every change to that decision.',
    quickStart: 'Quick start',
    viewSource: 'View the source',
    browseSkills: 'Browse skills',
    terminalTitle: 'start the catalog',
    terminalNote: 'Starts the catalog, Postgres, and a local npm registry.',
    card: {
      catalogLabel: 'catalog / release',
      published: 'published',
      integrity: 'integrity',
      manifest: 'manifest',
      policy: 'policy',
      resolved: 'resolved',
      resolvable: 'resolvable',
      agents: 'agents',
      extraAgents: '+53',
      resolveInstall: 'resolve install',
      resolvedIn: 'resolved in 38 ms',
      installTarget: 'Install target returned for 1.4.0',
      example: 'Example release record',
    },
  },
  agents: {
    eyebrow: 'Agent coverage',
    title: 'One CLI, 56 agent targets.',
    copy: 'The same bundle installs into Claude Code, Codex, Cursor, OpenCode, and the directories other agents read. Choose project or global scope.',
  },
  release: {
    eyebrow: 'Release discipline',
    title: 'A git ref is not a release.',
    lede: 'A skill can change how an agent works. It can also ship scripts. If the source is a branch, the same name can point at different files tomorrow. Skill Dock gives each bundle an exact version, an integrity hash, and a record of whether it is installable.',
    points: [
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
    ] as ReleasePoint[],
  },
  howItWorks: {
    eyebrow: 'How it works',
    title: 'Publish, resolve, govern.',
    lede: 'The catalog sits between the CLI and your npm registry. It decides what a version means and who can install it.',
    tablistLabel: 'Release workflow',
    flowLabel: (label: string) => `${label} flow`,
    orPrefix: 'Or: ',
    workflows: [
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
    ] as WorkflowStep[],
  },
  capabilities: {
    eyebrow: 'What the catalog owns',
    title: 'The catalog layer.',
    lede: 'Your npm registry already stores packages. The catalog adds the release decisions that packages do not carry.',
    items: [
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
    ] as Capability[],
    visuals: {
      published: 'published',
      deprecated: 'deprecated',
      active: 'active',
      yanked: 'yanked',
      blocked: 'blocked',
      retire: 'retire',
      auditMeta: 'alice · 2d ago',
      auditReason: 'reason: replaced by release-notes-v2',
      proxy: 'proxy',
      direct: 'direct',
      catalog: 'catalog',
      agent: 'agent',
    },
  },
  architecture: {
    eyebrow: 'Architecture',
    title: 'Three pieces, one release path.',
    lede: 'The catalog is a service you run. Postgres holds release records and governance data. Your npm registry holds the bundle bytes. The CLI runs on a developer machine or in CI.',
    flowLabel: 'Architecture flow',
    selectedLabel: 'selected piece',
    note: 'The catalog never stores bundle bytes. It stores the release record, the manifest digest, and the install policy. The npm registry stores the tarball.',
    nodes: [
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
        subtitle: 'self-hosted service',
        detail:
          'Owns release records, install resolution, lifecycle state, and audit events. It never stores bundle bytes.',
        icon: Server,
      },
      {
        id: 'postgres',
        title: 'Postgres',
        subtitle: 'catalog state',
        detail:
          'Stores release records, governance changes, audit events, and idempotency state.',
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
    ] as ArchitectureNode[],
  },
  quickStart: {
    eyebrow: 'Quick start',
    title: 'Run it locally.',
    lede: 'The Compose stack starts the catalog, Postgres, and a local npm registry. You can swap the registry for Nexus or Artifactory when you deploy.',
    tablistLabel: 'Quick start path',
    catalogTab: 'Catalog',
    cliTab: 'CLI',
    catalogSteps: [
      {
        title: 'Clone and configure',
        body: 'Replace the sample passwords and tokens in .env before you start the stack.',
        command: 'git clone https://github.com/ZenlixAI/skill-dock.git\ncd skill-dock\ncp .env.example .env',
      },
      {
        title: 'Start the stack',
        body: 'Compose starts the catalog, Postgres, and a local npm registry.',
        command: 'docker compose up --build -d\ncurl -fsS http://localhost:8080/readyz',
      },
    ] as QuickStartStep[],
    cliSteps: [
      {
        title: 'Build the CLI',
        body: 'The CLI lives in the same repository. Install it globally after the build.',
        command:
          'npx -y pnpm@10.17.1 --dir apps/skills-cli install\nnpx -y pnpm@10.17.1 --dir apps/skills-cli build\nnpm install -g ./apps/skills-cli',
      },
      {
        title: 'Point it at the catalog',
        body: 'Use the publisher token for authoring. Use the reader token for installs.',
        command:
          'export SKILLS_REGISTRY_URL=http://localhost:8080\nexport SKILLS_REGISTRY_TOKEN=<publisher-token>',
      },
      {
        title: 'Publish and install',
        body: 'The example uses the scoped package name the catalog expects.',
        command:
          'skills import-local ./my-skill \\\n  --package-name @acme.skill/my-skill \\\n  --version 1.4.0\n\nskills add @acme.skill/my-skill',
      },
    ] as QuickStartStep[],
    productionNotePrefix: 'For production, use ',
    productionNoteMiddle: ', set ',
    productionNoteSuffix: ' to your npm registry, and put the catalog behind TLS.',
  },
  faq: {
    eyebrow: 'FAQ',
    title: 'Questions that come up.',
    lede: 'Short answers to the questions that come up before a first deployment.',
    items: [
      {
        question: 'Does Skill Dock store the bundle?',
        answer:
          'No. Skill Dock stores release metadata and resolves tarball URLs. The bundle bytes stay in your npm registry.',
      },
      {
        question: 'Which npm registries work?',
        answer:
          'Any registry that supports the npm publish and package metadata APIs. Teams commonly use Nexus, Artifactory, or Verdaccio. The local Compose stack includes Verdaccio for evaluation.',
      },
      {
        question: 'Can developers keep installing from git?',
        answer:
          'Yes. skills add accepts a local path, a GitHub shorthand, or a git URL. skills sync refreshes those installs. Registry installs use skills update.',
      },
      {
        question: 'What happens when a version is yanked?',
        answer:
          'The tarball stays in npm. Skill Dock marks the version yanked and blocked. New installs fail with an explicit error, and the response can include a replacement recommendation.',
      },
      {
        question: 'How do I run it in production?',
        answer:
          'Run the catalog with Postgres, point SKILL_REGISTRY_ARTIFACT_READ_URL at your npm registry, and put the catalog behind TLS. The production Compose file is in zarf/compose/production.yml.',
      },
    ] as FaqItem[],
  },
  finalCta: {
    eyebrow: 'Start here',
    title: 'Start with the catalog.',
    body: 'The repository has a Compose stack, a production Compose file, and the CLI source. No hosted account is required.',
    quickStart: 'Quick start',
    openGithub: 'Open GitHub',
  },
  registry: {
    eyebrow: 'Registry · skills.sh',
    title: 'Skills Directory',
    lede: 'Browse the public skills.sh directory. Install any skill from its source repository.',
    searchLabel: 'Search skills',
    searchPlaceholder: 'Search skills.sh',
    clearSearch: 'Clear search',
    minCharsHint: 'Type at least 2 characters to search skills.sh.',
    viewsLabel: 'Browse view',
    views: [
      { value: 'all-time', label: 'All-time' },
      { value: 'trending', label: 'Trending' },
      { value: 'hot', label: 'Hot' },
    ] as Array<{ value: SkillsShBrowseView; label: string }>,
    viewDescriptions: {
      'all-time': 'Popular skills by all-time installs',
      trending: 'Trending skills from the last 24 hours',
      hot: 'Skills with the biggest short-term movement',
    } as Record<SkillsShBrowseView, string>,
    capabilityCheckErrorTitle: 'Could not check the skills directory',
    capabilityCheckErrorFallback: 'Could not check the skills directory.',
    disabledTitle: 'Skills directory is not enabled',
    disabledMessage: 'This deployment has disabled the public skills directory.',
    keepTypingTitle: 'Keep typing',
    loadErrorTitle: 'Could not load the skills directory',
    loadErrorFallback: 'The skills directory is unavailable.',
    resultsTitle: 'skills.sh',
    searchResultsFor: (query: string) => `Search results for “${query}”.`,
    communityNotice: 'A community directory, not reviewed or hosted by Skill Dock.',
    viewAll: 'View all on skills.sh',
    showing: (shown: string, total: string) => `Showing ${shown} of ${total}`,
    installs: (formatted: string) => `${formatted} installs`,
    official: 'Official',
    viewDetails: 'View details',
    emptyTitle: 'Nothing found',
    emptySearch: (query: string) => `No skills.sh results for “${query}”.`,
    emptyView: 'No skills in this view right now.',
    detailUnavailable: 'Could not load this skill.',
    formatLocale: 'en-US',
    drawer: {
      ariaLabel: 'External skill details',
      closeLabel: 'Close skill details',
      installation: 'Installation',
      installUnavailable: 'Install command unavailable.',
      refreshNote: 'Fetches from the source repository. Refresh with',
      refreshCommand: 'skills sync',
      refreshSuffix: '.',
      installsLabel: 'Installs',
      summary: 'Summary',
      renderedFrom: "Rendered from the skill's SKILL.md.",
      relatedSkills: 'Related skills',
      relatedInstalls: (formatted: string) => `${formatted} installs`,
      weeklySparkline: (values: string) => `Weekly installs: ${values}`,
      weeklyLabel: '8-week installs',
      viewOn: 'View on skills.sh',
      copyCommand: 'Copy command to clipboard',
      copied: 'Copied',
    },
  },
  legal: {
    eyebrow: 'Legal',
    back: 'Back to the product',
    privacy: {
      title: 'Privacy',
      lede: 'How the Skill Dock website handles site usage data.',
      articleLabel: 'Privacy content',
      sections: [
        {
          heading: 'What the site collects',
          body: 'The public site is a static build. It does not include an account system, analytics script, or advertising tracker. The hosting provider may keep standard request logs such as IP address, user agent, and requested path.',
        },
        {
          heading: 'Clipboard actions',
          body: 'Copy buttons write the selected command to your clipboard through the browser API. The site does not send the copied text anywhere.',
        },
        {
          heading: 'External links',
          body: 'Links to GitHub and the documentation open external sites. Their privacy policies apply once you leave this site.',
        },
        {
          heading: 'Contact',
          body: 'For questions about the project, open an issue in the ',
          linkLabel: 'GitHub repository',
          bodyAfter: '.',
        },
      ] as Array<
        LegalSection & { linkLabel?: string; bodyAfter?: string }
      >,
    },
    terms: {
      title: 'Terms',
      lede: 'Terms for using the Skill Dock website and related materials.',
      articleLabel: 'Terms content',
      sections: [
        {
          heading: 'Use of the site',
          body: 'The site provides product information, documentation links, and commands for running the software. You may use the site for evaluation and internal planning.',
        },
        {
          heading: 'Software license',
          body: 'The Skill Dock source code is released under the MIT License. The license in the ',
          linkLabel: 'repository',
          bodyAfter: ' governs your use of the software.',
        },
        {
          heading: 'Operational responsibility',
          body: 'You are responsible for the skills you publish, the registries you connect, and the access policies you apply. Review commands and configuration before you use them in a production environment.',
        },
        {
          heading: 'No warranty',
          body: 'The site and the software are provided as is, without warranties of any kind. See the MIT License for the full disclaimer.',
        },
      ] as Array<
        LegalSection & { linkLabel?: string; bodyAfter?: string }
      >,
    },
  },
  notFound: {
    eyebrow: '404',
    title: 'Page not found',
    lede: 'That route is not part of the site. Head back to the product page.',
    back: 'Return home',
  },
};

export type SiteContent = typeof en;
