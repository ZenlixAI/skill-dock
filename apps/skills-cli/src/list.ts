import { basename } from 'path';
import { homedir } from 'os';
import type { AgentType } from './types.ts';
import { agents } from './agents.ts';
import { listInstalledSkills, sanitizeName, type InstalledSkill } from './installer.ts';
import { sanitizeMetadata } from './sanitize.ts';
import { getAllLockedSkills } from './skill-lock.ts';
import { readLocalLock } from './local-lock.ts';

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

interface DisplayMetadata {
  name: string;
  version?: string;
  pluginName?: string;
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

function getDisplayMetadataForSkill(
  skill: InstalledSkill,
  displayMetadataBySanitized: Map<string, DisplayMetadata>
): DisplayMetadata | undefined {
  return (
    displayMetadataBySanitized.get(sanitizeName(skill.name)) ||
    displayMetadataBySanitized.get(basename(skill.canonicalPath))
  );
}

async function getDisplayMetadataByScope(global: boolean): Promise<Map<string, DisplayMetadata>> {
  const displayMetadataBySanitized = new Map<string, DisplayMetadata>();

  if (global) {
    const lockedSkills = await getAllLockedSkills();
    for (const [key, value] of Object.entries(lockedSkills)) {
      displayMetadataBySanitized.set(sanitizeName(key), {
        name: key,
        version: value.version,
        pluginName: value.pluginName,
      });
    }
  } else {
    const localLock = await readLocalLock();
    for (const [key, value] of Object.entries(localLock.skills)) {
      displayMetadataBySanitized.set(sanitizeName(key), {
        name: key,
        version: value.version,
      });
    }
  }

  return displayMetadataBySanitized;
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
  const scope = allScopes ? undefined : options.global === true ? true : false;

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

  const installedSkills = await listInstalledSkills({
    global: scope,
    agentFilter,
  });

  const projectDisplayMetadata = await getDisplayMetadataByScope(false);
  const globalDisplayMetadata = await getDisplayMetadataByScope(true);
  const displayMetadataBySanitized = scope ? globalDisplayMetadata : projectDisplayMetadata;

  if (allScopes) {
    const entries = installedSkills.map((skill) => {
      const metadata = getDisplayMetadataForSkill(
        skill,
        skill.scope === 'global' ? globalDisplayMetadata : projectDisplayMetadata
      );

      return {
        name: metadata?.name || skill.name,
        version: metadata?.version || null,
        scope: skill.scope,
        path: skill.canonicalPath,
        agents: skill.agents.map((a) => agents[a].displayName),
      };
    });

    const grouped = new Map<
      string,
      {
        name: string;
        winner: 'project' | 'global';
        scopes: Array<{
          scope: 'project' | 'global';
          version: string | null;
          path: string;
          agents: string[];
        }>;
      }
    >();

    for (const entry of entries) {
      const existing = grouped.get(entry.name);
      if (existing) {
        existing.scopes.push(entry);
        if (entry.scope === 'project') {
          existing.winner = 'project';
        }
      } else {
        grouped.set(entry.name, {
          name: entry.name,
          winner: entry.scope,
          scopes: [entry],
        });
      }
    }

    const values = Array.from(grouped.values()).sort((a, b) => a.name.localeCompare(b.name));

    if (options.json) {
      console.log(JSON.stringify(values, null, 2));
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
        console.log(
          `  ${TEXT}${scopeEntry.scope}${RESET}${DIM}${version}${RESET} ${DIM}${shortenPath(scopeEntry.path, process.cwd())}${RESET}`
        );
        const agentInfo = scopeEntry.agents.length > 0 ? formatList(scopeEntry.agents) : 'not linked';
        console.log(`    ${DIM}Agents:${RESET} ${agentInfo}`);
      }
      console.log();
    }
    return;
  }

  // JSON output mode: structured, no ANSI, untruncated agent lists
  if (options.json) {
    const jsonOutput = installedSkills.map((skill) => ({
      name: getDisplayMetadataForSkill(skill, displayMetadataBySanitized)?.name || skill.name,
      version: getDisplayMetadataForSkill(skill, displayMetadataBySanitized)?.version || null,
      path: skill.canonicalPath,
      scope: skill.scope,
      agents: skill.agents.map((a) => agents[a].displayName),
    }));
    console.log(JSON.stringify(jsonOutput, null, 2));
    return;
  }

  // Fetch lock entries to get plugin grouping info
  const lockedSkills = await getAllLockedSkills();

  const cwd = process.cwd();
  const scopeLabel = scope ? 'Global' : 'Project';

  if (installedSkills.length === 0) {
    if (options.json) {
      console.log('[]');
      return;
    }
    console.log(`${DIM}No ${scopeLabel.toLowerCase()} skills found.${RESET}`);
    if (scope) {
      console.log(`${DIM}Try listing project skills without -g.${RESET}`);
    } else {
      console.log(`${DIM}Try listing global skills with -g.${RESET}`);
    }
    return;
  }

  function printSkill(skill: InstalledSkill, indent: boolean = false): void {
    const prefix = indent ? '  ' : '';
    const shortPath = shortenPath(skill.canonicalPath, cwd);
    const agentNames = skill.agents.map((a) => agents[a].displayName);
    const agentInfo =
      skill.agents.length > 0 ? formatList(agentNames) : `${YELLOW}not linked${RESET}`;
    const displayMetadata = getDisplayMetadataForSkill(skill, displayMetadataBySanitized);
    const displayName = displayMetadata?.name || skill.name;
    const versionSuffix = displayMetadata?.version ? ` ${DIM}v${displayMetadata.version}${RESET}` : '';
    console.log(
      `${prefix}${CYAN}${sanitizeMetadata(displayName)}${RESET}${versionSuffix} ${DIM}${shortPath}${RESET}`
    );
    console.log(`${prefix}  ${DIM}Agents:${RESET} ${agentInfo}`);
  }

  console.log(`${BOLD}${scopeLabel} Skills${RESET}`);
  console.log();

  // Group skills by plugin
  const groupedSkills: Record<string, InstalledSkill[]> = {};
  const ungroupedSkills: InstalledSkill[] = [];

  for (const skill of installedSkills) {
    const displayMetadata = getDisplayMetadataForSkill(skill, displayMetadataBySanitized);
    if (displayMetadata?.pluginName) {
      const group = displayMetadata.pluginName;
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
