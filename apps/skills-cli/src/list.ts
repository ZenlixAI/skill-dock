import { homedir } from 'os';
import type { AgentType } from './types.ts';
import { agents } from './agents.ts';
import { listInstalledView, type InstalledSkillView } from './installed-view.ts';
import { sanitizeMetadata } from './sanitize.ts';

const RESET = '\x1b[0m';
const BOLD = '\x1b[1m';
const DIM = '\x1b[38;5;102m';
const TEXT = '\x1b[38;5;145m';
const CYAN = '\x1b[36m';
const YELLOW = '\x1b[33m';

interface ListOptions {
  global?: boolean;
  project?: boolean;
  allScopes?: boolean;
  agent?: string[];
  json?: boolean;
}

/**
 * Shortens a path for display: replaces homedir with ~ and cwd with .
 */
function shortenPath(fullPath: string, cwd: string): string {
  const home = homedir();
  if (fullPath.startsWith(home)) {
    return fullPath.replace(home, '~');
  }
  if (fullPath.startsWith(cwd)) {
    return '.' + fullPath.slice(cwd.length);
  }
  return fullPath;
}

/**
 * Formats a list of items, truncating if too many
 */
function formatList(items: string[], maxShow: number = 5): string {
  if (items.length <= maxShow) {
    return items.join(', ');
  }
  const shown = items.slice(0, maxShow);
  const remaining = items.length - maxShow;
  return `${shown.join(', ')} +${remaining} more`;
}

function getDisplayName(record: InstalledSkillView): string {
  return record.packageName || record.displayName;
}

function getAgentNames(record: InstalledSkillView): string[] {
  return Array.from(
    new Set(
      record.agentLinks
        .filter((link) => link.kind !== 'missing')
        .map((link) => agents[link.agent].displayName)
    )
  );
}

function formatDisplayName(record: InstalledSkillView): string {
  const name = sanitizeMetadata(getDisplayName(record));
  return record.orphaned ? `${name} ${DIM}(orphaned)${RESET}` : name;
}

function matchesAllScopeIdentity(left: InstalledSkillView, right: InstalledSkillView): boolean {
  if (left.packageName && right.packageName) {
    return left.packageName === right.packageName;
  }

  return left.installName === right.installName;
}

export function parseListOptions(args: string[]): ListOptions {
  const options: ListOptions = {};

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '-g' || arg === '--global') {
      options.global = true;
    } else if (arg === '-p' || arg === '--project') {
      options.project = true;
    } else if (arg === '--all-scopes') {
      options.allScopes = true;
    } else if (arg === '--json') {
      options.json = true;
    } else if (arg === '-a' || arg === '--agent') {
      options.agent = options.agent || [];
      // Collect all following arguments until next flag
      while (i + 1 < args.length && !args[i + 1]!.startsWith('-')) {
        options.agent.push(args[++i]!);
      }
    }
  }

  return options;
}

export async function runList(args: string[]): Promise<void> {
  const options = parseListOptions(args);

  const allScopes = options.allScopes || (options.global === true && options.project === true);
  const globalScope = options.global === true;

  // Validate agent filter if provided
  let agentFilter: AgentType[] | undefined;
  if (options.agent && options.agent.length > 0) {
    const validAgents = Object.keys(agents);
    const invalidAgents = options.agent.filter((a) => !validAgents.includes(a));

    if (invalidAgents.length > 0) {
      console.log(`${YELLOW}Invalid agents: ${invalidAgents.join(', ')}${RESET}`);
      console.log(`${DIM}Valid agents: ${validAgents.join(', ')}${RESET}`);
      process.exit(1);
    }

    agentFilter = options.agent as AgentType[];
  }

  const installedSkills = await listInstalledView({
    scope: allScopes ? undefined : globalScope ? 'global' : 'project',
    agentFilter,
  });

  if (allScopes) {
    const grouped: Array<{
      record: InstalledSkillView;
      name: string;
      winner: 'project' | 'global';
      scopes: Array<{
        scope: 'project' | 'global';
        version: string | null;
        path: string;
        agents: string[];
        orphaned: boolean;
      }>;
    }> = [];

    for (const record of installedSkills) {
      const existing = grouped.find((group) => matchesAllScopeIdentity(group.record, record));
      const scopeEntry = {
        scope: record.scope,
        version: record.version || null,
        path: record.canonicalPath,
        agents: getAgentNames(record),
        orphaned: record.orphaned,
      };

      if (existing) {
        existing.scopes.push(scopeEntry);
        if (record.packageName && !existing.record.packageName) {
          existing.record = record;
        }
        if (getDisplayName(record).includes('/') && !existing.name.includes('/')) {
          existing.name = getDisplayName(record);
        }
        if (record.scope === 'project') {
          existing.winner = 'project';
        }
      } else {
        grouped.push({
          record,
          name: getDisplayName(record),
          winner: record.scope,
          scopes: [scopeEntry],
        });
      }
    }

    const values = grouped.sort((a, b) => a.name.localeCompare(b.name));

    if (options.json) {
      console.log(
        JSON.stringify(
          values.map(({ record: _, ...item }) => item),
          null,
          2
        )
      );
      return;
    }

    if (values.length === 0) {
      console.log(`${DIM}No installed skills found.${RESET}`);
      return;
    }

    console.log(`${BOLD}Installed Skills${RESET}`);
    console.log();

    for (const item of values) {
      console.log(`${CYAN}${sanitizeMetadata(item.name)}${RESET} ${DIM}(winner: ${item.winner})${RESET}`);
      for (const scopeEntry of item.scopes.sort((a, b) => a.scope.localeCompare(b.scope))) {
        const version = scopeEntry.version ? ` v${scopeEntry.version}` : '';
        const orphaned = scopeEntry.orphaned ? ` ${DIM}(orphaned)${RESET}` : '';
        console.log(
          `  ${TEXT}${scopeEntry.scope}${RESET}${DIM}${version}${RESET}${orphaned} ${DIM}${shortenPath(scopeEntry.path, process.cwd())}${RESET}`
        );
        const agentInfo =
          scopeEntry.agents.length > 0 ? formatList(scopeEntry.agents) : 'not linked';
        console.log(`    ${DIM}Agents:${RESET} ${agentInfo}`);
      }
      console.log();
    }
    return;
  }

  // JSON output mode: structured, no ANSI, untruncated agent lists
  if (options.json) {
    const jsonOutput = installedSkills.map((skill) => ({
      name: getDisplayName(skill),
      version: skill.version || null,
      path: skill.canonicalPath,
      scope: skill.scope,
      agents: getAgentNames(skill),
    }));
    console.log(JSON.stringify(jsonOutput, null, 2));
    return;
  }

  const cwd = process.cwd();
  const scopeLabel = globalScope ? 'Global' : 'Project';

  if (installedSkills.length === 0) {
    if (options.json) {
      console.log('[]');
      return;
    }
    console.log(`${DIM}No ${scopeLabel.toLowerCase()} skills found.${RESET}`);
    if (globalScope) {
      console.log(`${DIM}Try listing project skills without -g.${RESET}`);
    } else {
      console.log(`${DIM}Try listing global skills with -g.${RESET}`);
    }
    return;
  }

  function printSkill(skill: InstalledSkillView, indent: boolean = false): void {
    const prefix = indent ? '  ' : '';
    const shortPath = shortenPath(skill.canonicalPath, cwd);
    const agentNames = getAgentNames(skill);
    const agentInfo = agentNames.length > 0 ? formatList(agentNames) : `${YELLOW}not linked${RESET}`;
    const versionSuffix = skill.version ? ` ${DIM}v${skill.version}${RESET}` : '';
    console.log(
      `${prefix}${CYAN}${formatDisplayName(skill)}${RESET}${versionSuffix} ${DIM}${shortPath}${RESET}`
    );
    console.log(`${prefix}  ${DIM}Agents:${RESET} ${agentInfo}`);
  }

  console.log(`${BOLD}${scopeLabel} Skills${RESET}`);
  console.log();

  // Group skills by plugin
  const groupedSkills: Record<string, InstalledSkillView[]> = {};
  const ungroupedSkills: InstalledSkillView[] = [];

  for (const skill of installedSkills) {
    if (skill.pluginName) {
      const group = skill.pluginName;
      if (!groupedSkills[group]) {
        groupedSkills[group] = [];
      }
      groupedSkills[group].push(skill);
    } else {
      ungroupedSkills.push(skill);
    }
  }

  const hasGroups = Object.keys(groupedSkills).length > 0;

  if (hasGroups) {
    // Print groups sorted alphabetically
    const sortedGroups = Object.keys(groupedSkills).sort();
    for (const group of sortedGroups) {
      // Convert kebab-case to Title Case for display header
      const title = group
        .split('-')
        .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
        .join(' ');

      console.log(`${BOLD}${title}${RESET}`);
      const skills = groupedSkills[group];
      if (skills) {
        for (const skill of skills) {
          printSkill(skill, true);
        }
      }
      console.log();
    }

    // Print ungrouped skills if any exist
    if (ungroupedSkills.length > 0) {
      console.log(`${BOLD}General${RESET}`);
      for (const skill of ungroupedSkills) {
        printSkill(skill, true);
      }
      console.log();
    }
  } else {
    // No groups, print flat list as before
    for (const skill of installedSkills) {
      printSkill(skill);
    }
    console.log();
  }
}
