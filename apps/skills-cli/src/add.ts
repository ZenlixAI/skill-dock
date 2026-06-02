import * as p from '@clack/prompts';
import pc from 'picocolors';
import { existsSync } from 'fs';
import { homedir } from 'os';
import { sep, join, dirname } from 'path';
import { searchMultiselect } from './prompts/search-multiselect.ts';

// Helper to check if a value is a cancel symbol (works with both clack and our custom prompts)
const isCancelled = (value: unknown): value is symbol => typeof value === 'symbol';

import {
  installSkillForAgent,
  isSkillInstalled,
  getCanonicalPath,
  type InstallMode,
} from './installer.ts';
import {
  detectInstalledAgents,
  agents,
  getUniversalAgents,
  getNonUniversalAgents,
  isUniversalAgent,
} from './agents.ts';
import {
  track,
  setVersion,
} from './telemetry.ts';
import { detectAgent, getAgentType } from './detect-agent.ts';
import {
  addSkillToLock,
  isPromptDismissed,
  dismissPrompt,
  getLastSelectedAgents,
  saveSelectedAgents,
} from './skill-lock.ts';
import type { AgentType } from './types.ts';
import { runRegistryInstallWorkflow } from './registry-install-workflow.ts';
import packageJson from '../package.json' with { type: 'json' };
export function initTelemetry(version: string): void {
  setVersion(version);
}

/**
 * Shortens a path for display: replaces homedir with ~ and cwd with .
 * Handles both Unix and Windows path separators.
 */
function shortenPath(fullPath: string, cwd: string): string {
  const home = homedir();
  // Ensure we match complete path segments by checking for separator after the prefix
  if (fullPath === home || fullPath.startsWith(home + sep)) {
    return '~' + fullPath.slice(home.length);
  }
  if (fullPath === cwd || fullPath.startsWith(cwd + sep)) {
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

/**
 * Splits agents into universal and non-universal (symlinked) groups.
 * Returns display names for each group.
 */
function splitAgentsByType(agentTypes: AgentType[]): {
  universal: string[];
  symlinked: string[];
} {
  const universal: string[] = [];
  const symlinked: string[] = [];

  for (const a of agentTypes) {
    if (isUniversalAgent(a)) {
      universal.push(agents[a].displayName);
    } else {
      symlinked.push(agents[a].displayName);
    }
  }

  return { universal, symlinked };
}

/**
 * Builds summary lines showing universal vs symlinked agents
 */
function buildAgentSummaryLines(targetAgents: AgentType[], installMode: InstallMode): string[] {
  const lines: string[] = [];
  const { universal, symlinked } = splitAgentsByType(targetAgents);

  if (installMode === 'symlink') {
    if (universal.length > 0) {
      lines.push(`  ${pc.green('universal:')} ${formatList(universal)}`);
    }
    if (symlinked.length > 0) {
      lines.push(`  ${pc.dim('symlink →')} ${formatList(symlinked)}`);
    }
  } else {
    // Copy mode - all agents get copies
    const allNames = targetAgents.map((a) => agents[a].displayName);
    lines.push(`  ${pc.dim('copy →')} ${formatList(allNames)}`);
  }

  return lines;
}

/**
 * Ensures universal agents are always included in the target agents list.
 * Used when -y flag is passed or when auto-selecting agents.
 */
function ensureUniversalAgents(targetAgents: AgentType[]): AgentType[] {
  const universalAgents = getUniversalAgents();
  const result = [...targetAgents];

  for (const ua of universalAgents) {
    if (!result.includes(ua)) {
      result.push(ua);
    }
  }

  return result;
}

/**
 * Builds result lines from installation results, splitting by universal vs symlinked
 */
function buildResultLines(
  results: Array<{
    agent: string;
    symlinkFailed?: boolean;
    skipped?: boolean;
  }>,
  targetAgents: AgentType[]
): string[] {
  const lines: string[] = [];

  // Split target agents by type
  const { universal, symlinked: symlinkAgents } = splitAgentsByType(targetAgents);

  // For symlink results, also track which ones actually succeeded vs failed
  // Exclude skipped agents (those whose config dir doesn't exist in the project)
  const successfulSymlinks = results
    .filter((r) => !r.symlinkFailed && !r.skipped && !universal.includes(r.agent))
    .map((r) => r.agent);
  const failedSymlinks = results.filter((r) => r.symlinkFailed && !r.skipped).map((r) => r.agent);

  if (universal.length > 0) {
    lines.push(`  ${pc.green('universal:')} ${formatList(universal)}`);
  }
  if (successfulSymlinks.length > 0) {
    lines.push(`  ${pc.dim('symlinked:')} ${formatList(successfulSymlinks)}`);
  }
  if (failedSymlinks.length > 0) {
    lines.push(`  ${pc.yellow('copied:')} ${formatList(failedSymlinks)}`);
  }

  return lines;
}

/**
 * Wrapper around p.multiselect that adds a hint for keyboard usage.
 * Accepts options with required labels (matching our usage pattern).
 */
function multiselect<Value>(opts: {
  message: string;
  options: Array<{ value: Value; label: string; hint?: string }>;
  initialValues?: Value[];
  required?: boolean;
}) {
  return p.multiselect({
    ...opts,
    // Cast is safe: our options always have labels, which satisfies p.Option requirements
    options: opts.options as p.Option<Value>[],
    message: `${opts.message} ${pc.dim('(space to toggle)')}`,
  }) as Promise<Value[] | symbol>;
}

/**
 * Prompts the user to select agents using interactive search.
 * Pre-selects the last used agents if available.
 * Saves the selection for future use.
 */
export async function promptForAgents(
  message: string,
  choices: Array<{ value: AgentType; label: string; hint?: string }>
): Promise<AgentType[] | symbol> {
  // Get last selected agents to pre-select
  let lastSelected: string[] | undefined;
  try {
    lastSelected = await getLastSelectedAgents();
  } catch {
    // Silently ignore errors reading lock file
  }

  const validAgents = choices.map((c) => c.value);

  // Default agents to pre-select when no valid history exists
  const defaultAgents: AgentType[] = ['claude-code', 'opencode', 'codex'];
  const defaultValues = defaultAgents.filter((a) => validAgents.includes(a));

  let initialValues: AgentType[] = [];

  if (lastSelected && lastSelected.length > 0) {
    // Filter stored agents against currently valid agents
    initialValues = lastSelected.filter((a) => validAgents.includes(a as AgentType)) as AgentType[];
  }

  // If no valid selection from history, use defaults
  if (initialValues.length === 0) {
    initialValues = defaultValues;
  }

  const selected = await searchMultiselect({
    message,
    items: choices,
    initialSelected: initialValues,
    required: true,
  });

  if (!isCancelled(selected)) {
    // Save selection for next time
    try {
      await saveSelectedAgents(selected as string[]);
    } catch {
      // Silently ignore errors writing lock file
    }
  }

  return selected as AgentType[] | symbol;
}

/**
 * Interactive agent selection using fuzzy search.
 * Shows universal agents as locked (always selected), and other agents as selectable.
 */
async function selectAgentsInteractive(options: {
  global?: boolean;
}): Promise<AgentType[] | symbol> {
  // Filter out agents that don't support global installation when --global is used
  const supportsGlobalFilter = (a: AgentType) => !options.global || agents[a].globalSkillsDir;

  const universalAgents = getUniversalAgents().filter(supportsGlobalFilter);
  const otherAgents = getNonUniversalAgents().filter(supportsGlobalFilter);

  // Universal agents shown as locked section
  const universalSection = {
    title: 'Universal (.agents/skills)',
    items: universalAgents.map((a) => ({
      value: a,
      label: agents[a].displayName,
    })),
  };

  // Other agents are selectable with their skillsDir as hint
  const otherChoices = otherAgents.map((a) => ({
    value: a,
    label: agents[a].displayName,
    hint: options.global ? agents[a].globalSkillsDir! : agents[a].skillsDir,
  }));

  // Get last selected agents (filter to only non-universal ones for initial selection)
  let lastSelected: string[] | undefined;
  try {
    lastSelected = await getLastSelectedAgents();
  } catch {
    // Silently ignore errors
  }

  const initialSelected = lastSelected
    ? (lastSelected.filter(
        (a) => otherAgents.includes(a as AgentType) && !universalAgents.includes(a as AgentType)
      ) as AgentType[])
    : [];

  const selected = await searchMultiselect({
    message: 'Which agents do you want to install to?',
    items: otherChoices,
    initialSelected,
    lockedSection: universalSection,
  });

  if (!isCancelled(selected)) {
    // Save selection (all agents including universal)
    try {
      await saveSelectedAgents(selected as string[]);
    } catch {
      // Silently ignore errors
    }
  }

  return selected as AgentType[] | symbol;
}

const version = packageJson.version;
setVersion(version);

export interface AddOptions {
  global?: boolean;
  agent?: string[];
  yes?: boolean;
  verbose?: boolean;
  skill?: string[];
  list?: boolean;
  all?: boolean;
  fullDepth?: boolean;
  copy?: boolean;
  dangerouslyAcceptOpenclawRisks?: boolean;
  version?: string;
  registry?: string;
}

const PACKAGE_NAME_RE = /^(?:@[a-z0-9][a-z0-9._-]*\/)?[a-z0-9][a-z0-9._-]*$/;
const EXACT_VERSION_RE =
  /^\d+\.\d+\.\d+(?:-[0-9A-Za-z]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/;

function isPackageName(value: string): boolean {
  return PACKAGE_NAME_RE.test(value);
}

function isExactVersion(value: string): boolean {
  return EXACT_VERSION_RE.test(value);
}

function showAddUsageError(message: string): never {
  console.log();
  console.log(pc.bgRed(pc.white(pc.bold(' ERROR '))) + ' ' + pc.red(message));
  console.log();
  console.log(pc.dim('  Usage:'));
  console.log(
    `    ${pc.cyan('npx skills add')} ${pc.yellow('<package_name>')} ${pc.dim('[options]')}`
  );
  console.log();
  console.log(pc.dim('  Example:'));
  console.log(`    ${pc.cyan('npx skills add')} ${pc.yellow('@platform.skill/pdf-skill')}`);
  console.log();
  process.exit(1);
}

function validateRegistryAddOptions(options: AddOptions): void {
  const unsupported: string[] = [];
  if (options.list) unsupported.push('--list');
  if (options.skill && options.skill.length > 0) unsupported.push('--skill');
  if (options.all) unsupported.push('--all');
  if (options.fullDepth) unsupported.push('--full-depth');
  if (options.dangerouslyAcceptOpenclawRisks) unsupported.push('--dangerously-accept-openclaw-risks');

  if (unsupported.length > 0) {
    throw new Error(
      `Registry installs do not support ${unsupported.join(', ')}. Use the package_name directly.`
    );
  }

  if (options.version && !isExactVersion(options.version)) {
    throw new Error(
      `The --version option only accepts an exact version. Received "${options.version}".`
    );
  }
}

function formatAgentDisplayList(agentTypes: AgentType[], maxShow: number = 6): string {
  return formatList(
    agentTypes.map((agentType) => agents[agentType].displayName),
    maxShow
  );
}

function resolveDefaultTargetAgents(installedAgents: AgentType[]): AgentType[] {
  if (installedAgents.length > 0) {
    return ensureUniversalAgents(installedAgents);
  }
  return getUniversalAgents();
}

async function resolveTargetAgentsForAdd(
  options: AddOptions,
  spinner: ReturnType<typeof p.spinner>
): Promise<AgentType[]> {
  let targetAgents: AgentType[];
  const validAgents = Object.keys(agents);

  if (options.agent?.includes('*')) {
    targetAgents = validAgents as AgentType[];
    p.log.info(`Installing to all ${targetAgents.length} agents`);
    return targetAgents;
  }

  if (options.agent && options.agent.length > 0) {
    const invalidAgents = options.agent.filter((a) => !validAgents.includes(a));
    if (invalidAgents.length > 0) {
      p.log.error(`Invalid agents: ${invalidAgents.join(', ')}`);
      p.log.info(`Valid agents: ${validAgents.join(', ')}`);
      process.exit(1);
    }

    return options.agent as AgentType[];
  }

  spinner.start('Loading agents...');
  const installedAgents = await detectInstalledAgents();
  spinner.stop(
    installedAgents.length === 0
      ? 'No installed agents detected'
      : `Detected ${installedAgents.length} installed agent${installedAgents.length === 1 ? '' : 's'}`
  );

  if (installedAgents.length === 0) {
    if (options.yes) {
      targetAgents = resolveDefaultTargetAgents([]);
      p.log.info(`Installing to default agents: ${formatAgentDisplayList(targetAgents)}`);
      return targetAgents;
    }

    p.log.info('Select agents to install skills to');
    const allAgentChoices = Object.entries(agents).map(([key, config]) => ({
      value: key as AgentType,
      label: config.displayName,
    }));

    const selected = await promptForAgents('Which agents do you want to install to?', allAgentChoices);
    if (p.isCancel(selected)) {
      p.cancel('Installation cancelled');
      process.exit(0);
    }

    return selected as AgentType[];
  }

  if (installedAgents.length === 1 || options.yes) {
    targetAgents = resolveDefaultTargetAgents(installedAgents);
    if (installedAgents.length === 1) {
      const firstAgent = installedAgents[0]!;
      p.log.info(`Installing to: ${pc.cyan(agents[firstAgent].displayName)}`);
    } else {
      p.log.info(`Installing to: ${formatAgentDisplayList(targetAgents)}`);
    }
    return targetAgents;
  }

  const selected = await selectAgentsInteractive({ global: options.global });
  if (p.isCancel(selected)) {
    p.cancel('Installation cancelled');
    process.exit(0);
  }

  return selected as AgentType[];
}

async function resolveScopeAndModeForAdd(
  options: AddOptions,
  targetAgents: AgentType[]
): Promise<{ installGlobally: boolean; installMode: InstallMode }> {
  let installGlobally = options.global ?? false;
  const supportsGlobal = targetAgents.some((a) => agents[a].globalSkillsDir !== undefined);

  if (options.global === undefined && !options.yes && supportsGlobal) {
    const scope = await p.select({
      message: 'Installation scope',
      options: [
        {
          value: false,
          label: 'Project',
          hint: 'Install in current directory (committed with your project)',
        },
        {
          value: true,
          label: 'Global',
          hint: 'Install in home directory (available across all projects)',
        },
      ],
    });

    if (p.isCancel(scope)) {
      p.cancel('Installation cancelled');
      process.exit(0);
    }

    installGlobally = scope as boolean;
  }

  let installMode: InstallMode = options.copy ? 'copy' : 'symlink';
  const uniqueDirs = new Set(targetAgents.map((a) => agents[a].skillsDir));

  if (!options.copy && !options.yes && uniqueDirs.size > 1) {
    const modeChoice = await p.select({
      message: 'Installation method',
      options: [
        {
          value: 'symlink',
          label: 'Symlink (Recommended)',
          hint: 'Single source of truth, easy updates',
        },
        { value: 'copy', label: 'Copy to all agents', hint: 'Independent copies for each agent' },
      ],
    });

    if (p.isCancel(modeChoice)) {
      p.cancel('Installation cancelled');
      process.exit(0);
    }

    installMode = modeChoice as InstallMode;
  } else if (uniqueDirs.size <= 1) {
    installMode = 'copy';
  }

  return { installGlobally, installMode };
}

function buildInstallResultLines(
  successful: Array<{
    agent: AgentType;
    result: {
      path: string;
      canonicalPath?: string;
    };
  }>,
  cwd: string,
  verbose: boolean
): string[] {
  if (verbose) {
    return successful.map(({ agent, result }) => {
      const path = result.canonicalPath || result.path;
      return `${pc.green('✓')} ${agents[agent].displayName} ${pc.dim(shortenPath(path, cwd))}`;
    });
  }

  const agentNames = successful.map(({ agent }) => agents[agent].displayName);
  const uniquePaths = Array.from(
    new Set(
      successful.map(({ result }) => shortenPath(result.canonicalPath || result.path, cwd))
    )
  );

  const lines = [`Agents: ${formatList(agentNames, 8)}`];
  if (uniquePaths.length === 1) {
    lines.push(`Location: ${uniquePaths[0]}`);
  } else {
    lines.push(`Locations: ${formatList(uniquePaths, 3)}`);
  }
  return lines;
}

export async function installRegistryPackage(
  packageName: string,
  options: AddOptions,
  spinner: ReturnType<typeof p.spinner>
): Promise<void> {
  validateRegistryAddOptions(options);
  const targetAgents = await resolveTargetAgentsForAdd(options, spinner);
  const { installGlobally, installMode } = await resolveScopeAndModeForAdd(options, targetAgents);
  const cwd = process.cwd();

  const summaryLines = [
    `${pc.cyan(getCanonicalPath(packageName, { global: installGlobally, cwd }))}`,
    ...buildAgentSummaryLines(targetAgents, installMode),
  ];

  console.log();
  p.note(summaryLines.join('\n'), 'Installation Summary');

  if (!options.yes) {
    const confirmed = await p.confirm({ message: 'Proceed with installation?' });
    if (p.isCancel(confirmed) || !confirmed) {
      p.cancel('Installation cancelled');
      process.exit(0);
    }
  }

  spinner.start('Installing skill...');
  const { successful, failed } = await runRegistryInstallWorkflow({
    packageName,
    version: options.version,
    registry: options.registry,
    targetAgents,
    installGlobally,
    installMode,
    cwd,
  });
  spinner.stop('Installation complete');

  const resultLines = buildInstallResultLines(successful, cwd, options.verbose === true);

  p.note(
    resultLines.join('\n'),
    pc.green(`Installed ${successful.length} target${successful.length === 1 ? '' : 's'}`)
  );

  if (failed.length > 0) {
    p.log.warn(pc.yellow(`Failed on ${failed.length} agent(s)`));
    for (const entry of failed) {
      p.log.message(
        `  ${pc.red('✗')} ${agents[entry.agent].displayName}: ${pc.dim(entry.result.error)}`
      );
    }
  }

  console.log();
  p.outro(
    pc.green('Done!') + pc.dim('  Review skills before use; they run with full agent permissions.')
  );
}

export async function runAdd(args: string[], options: AddOptions = {}): Promise<void> {
  const source = args[0];
  let installTipShown = false;

  const showInstallTip = (): void => {
    if (installTipShown) return;
    p.log.message(
      pc.dim('Tip: use the --yes (-y) and --global (-g) flags to install without prompts.')
    );
    installTipShown = true;
  };

  if (!source) {
    showAddUsageError('Missing required argument: package_name');
  }

  // Auto-enable non-interactive mode when running inside an AI agent
  const agentResult = await detectAgent();
  if (agentResult.isAgent) {
    options.yes = true;
    // Auto-select the detected agent + universal agents (unless user explicitly specified agents)
    if (!options.agent || options.agent.length === 0) {
      const mappedAgent = getAgentType(agentResult.agent.name);
      if (mappedAgent) {
        options.agent = ensureUniversalAgents([mappedAgent]);
      }
    }
  }

  console.log();
  if (!agentResult.isAgent) {
    p.intro(pc.bgCyan(pc.black(' skills ')));
  }

  if (agentResult.isAgent) {
    p.log.info(
      pc.bgCyan(pc.black(pc.bold(` ${agentResult.agent.name} `))) +
        ' ' +
        'Agent detected — installing non-interactively'
    );
  } else if (!process.stdin.isTTY) {
    showInstallTip();
  }

  try {
    const spinner = p.spinner();

    if (!isPackageName(source)) {
      showAddUsageError(
        'This command only accepts a registry package_name. Legacy local path, owner/repo, and URL installs are disabled.'
      );
    }

    await installRegistryPackage(source, options, spinner);
  } catch (error) {
    p.log.error(error instanceof Error ? error.message : 'Unknown error occurred');
    showInstallTip();
    p.outro(pc.red('Installation failed'));
    process.exit(1);
  }
}

/**
 * Prompt user to install the find-skills skill after their first installation.
 */
async function promptForFindSkills(
  options?: AddOptions,
  targetAgents?: AgentType[]
): Promise<void> {
  const registryPackage = process.env.SKILLS_FIND_SKILLS_PACKAGE;
  if (!registryPackage) return;

  // Skip if already dismissed or not in interactive mode
  if (!process.stdin.isTTY) return;
  if (options?.yes) return;

  try {
    const dismissed = await isPromptDismissed('findSkillsPrompt');
    if (dismissed) return;

    // Check if find-skills is already installed
    const findSkillsInstalled = await isSkillInstalled(registryPackage, 'claude-code', {
      global: true,
    });
    if (findSkillsInstalled) {
      // Mark as dismissed so we don't check again
      await dismissPrompt('findSkillsPrompt');
      return;
    }

    console.log();
    p.log.message(pc.dim("One-time prompt - you won't be asked again if you dismiss."));
    const install = await p.confirm({
      message: `Install ${pc.cyan(registryPackage)}? It helps your agent discover and suggest skills.`,
    });

    if (p.isCancel(install)) {
      await dismissPrompt('findSkillsPrompt');
      return;
    }

    if (install) {
      // Install find-skills to the same agents the user selected, excluding replit
      await dismissPrompt('findSkillsPrompt');

      // Filter out replit from target agents
      const findSkillsAgents = targetAgents?.filter((a) => a !== 'replit');

      // Skip if no valid agents remain after filtering
      if (!findSkillsAgents || findSkillsAgents.length === 0) {
        return;
      }

      console.log();
      p.log.step('Installing find-skills skill...');

      try {
        await runAdd([registryPackage], {
          global: true,
          yes: true,
          agent: findSkillsAgents,
        });
      } catch {
        p.log.warn('Failed to install find-skills. You can try again with:');
        p.log.message(pc.dim(`  npx skills add ${registryPackage} -g -y --all`));
      }
    } else {
      // User declined - dismiss the prompt
      await dismissPrompt('findSkillsPrompt');
      p.log.message(pc.dim(`You can install it later with: npx skills add ${registryPackage}`));
    }
  } catch {
    // Don't fail the main installation if prompt fails
  }
}

// Parse command line options from args array
export function parseAddOptions(args: string[]): { source: string[]; options: AddOptions } {
  const options: AddOptions = {};
  const source: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === '-g' || arg === '--global') {
      options.global = true;
    } else if (arg === '-y' || arg === '--yes') {
      options.yes = true;
    } else if (arg === '--verbose') {
      options.verbose = true;
    } else if (arg === '--version') {
      if (i + 1 < args.length) {
        options.version = args[++i];
      }
    } else if (arg === '--registry') {
      if (i + 1 < args.length) {
        options.registry = args[++i];
      }
    } else if (arg === '-l' || arg === '--list') {
      options.list = true;
    } else if (arg === '--all') {
      options.all = true;
    } else if (arg === '-a' || arg === '--agent') {
      options.agent = options.agent || [];
      i++;
      let nextArg = args[i];
      while (i < args.length && nextArg && !nextArg.startsWith('-')) {
        options.agent.push(nextArg);
        i++;
        nextArg = args[i];
      }
      i--; // Back up one since the loop will increment
    } else if (arg === '-s' || arg === '--skill') {
      options.skill = options.skill || [];
      i++;
      let nextArg = args[i];
      while (i < args.length && nextArg && !nextArg.startsWith('-')) {
        options.skill.push(nextArg);
        i++;
        nextArg = args[i];
      }
      i--; // Back up one since the loop will increment
    } else if (arg === '--full-depth') {
      options.fullDepth = true;
    } else if (arg === '--copy') {
      options.copy = true;
    } else if (arg === '--dangerously-accept-openclaw-risks') {
      options.dangerouslyAcceptOpenclawRisks = true;
    } else if (arg && !arg.startsWith('-')) {
      source.push(arg);
    }
  }

  return { source, options };
}
