#!/usr/bin/env node

import module from 'node:module';
import { writeFileSync, readFileSync, existsSync, mkdirSync } from 'fs';
import { basename, join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { runAdd, parseAddOptions, initTelemetry } from './add.ts';
import { runFind } from './find.ts';
import { runList } from './list.ts';
import { removeCommand, parseRemoveOptions } from './remove.ts';
import { flushTelemetry } from './telemetry.ts';
import { runInspect } from './inspect.ts';
import { runImportLocal } from './import-local.ts';
import { runPublish } from './publish.ts';
import { runUpdate } from './update.ts';

if (module.enableCompileCache && !process.env.NODE_DISABLE_COMPILE_CACHE) {
  try {
    module.enableCompileCache();
  } catch {
    // Ignore errors so CLI startup still works on runtimes without compile-cache support.
  }
}

const __dirname = dirname(fileURLToPath(import.meta.url));

function getVersion(): string {
  try {
    const pkgPath = join(__dirname, '..', 'package.json');
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf-8'));
    return pkg.version;
  } catch {
    return '0.0.0';
  }
}

const VERSION = getVersion();
initTelemetry(VERSION);

const RESET = '\x1b[0m';
const BOLD = '\x1b[1m';
// 256-color grays - visible on both light and dark backgrounds
const DIM = '\x1b[38;5;102m'; // darker gray for secondary text
const TEXT = '\x1b[38;5;145m'; // lighter gray for primary text

function showHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills <command> [options]

${BOLD}Manage Skills:${RESET}
  add <package_name>   Add a skill package from the registry (alias: a)
                       e.g. @company.skill/pdf-skill
  remove [skills]      Remove installed skills
  list, ls             List installed skills
  find [query]         Search for skills (supports --registry, --json)

${BOLD}Updates:${RESET}
  update [skills...]   Update skills to latest versions

${BOLD}Update Options:${RESET}
  -g, --global           Update global skills only
  -p, --project          Update project skills only
  -y, --yes              Skip scope prompt (auto-detect: project if in a project, else global)

${BOLD}Project:${RESET}
  init [name]          Initialize a skill (creates <name>/SKILL.md or ./SKILL.md)
  inspect [path]       Validate a local skill directory
  publish [path]       Validate, pack, and publish a local skill directory
  import-local [path]  Upload a local source directory for import + publish

${BOLD}Add Options:${RESET}
  -g, --global           Install skill globally (user-level) instead of project-level
  -a, --agent <agents>   Specify agents to install to (use '*' for all agents)
  -y, --yes              Skip confirmation prompts
  --registry <url>       Catalog base URL override (default env: SKILLS_REGISTRY_URL)
  --version <version>    Install an exact version from the registry
  --copy                 Copy files instead of symlinking to agent directories
  --verbose              Show per-agent install results instead of a compact summary

${BOLD}Remove Options:${RESET}
  -g, --global           Remove from global scope
  -a, --agent <agents>   Remove from specific agents (use '*' for all agents)
  -s, --skill <skills>   Specify skills to remove (use '*' for all skills)
  -y, --yes              Skip confirmation prompts
  --all                  Shorthand for --skill '*' --agent '*' -y

${BOLD}List Options:${RESET}
  -g, --global           List global skills (default: project)
  -p, --project          List project skills explicitly
  --all-scopes           Show project and global installs together with winner
  -a, --agent <agents>   Filter by specific agents
  --json                 Output as JSON (machine-readable, no ANSI codes)

${BOLD}Find Options:${RESET}
  --registry <url>       Catalog base URL override (default env: SKILLS_REGISTRY_URL)
  --json                 Output query results as JSON

${BOLD}Inspect / Publish Options:${RESET}
  --registry <url>       Catalog base URL override for registry-backed packaging flows
  --strict               Disable imported workspace normalization

${BOLD}Options:${RESET}
  --help, -h        Show this help message
  --version, -v     Show version number

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills add @company.skill/pdf-skill
  ${DIM}$${RESET} skills add @company.skill/pdf-skill -g
  ${DIM}$${RESET} skills add @company.skill/pdf-skill --version 1.2.0
  ${DIM}$${RESET} skills add @company.skill/pdf-skill --registry https://catalog.example.com
  ${DIM}$${RESET} skills remove                        ${DIM}# interactive remove${RESET}
  ${DIM}$${RESET} skills remove web-design             ${DIM}# remove by name${RESET}
  ${DIM}$${RESET} skills rm --global frontend-design
  ${DIM}$${RESET} skills list                          ${DIM}# list project skills${RESET}
  ${DIM}$${RESET} skills ls -g                         ${DIM}# list global skills${RESET}
  ${DIM}$${RESET} skills ls --all-scopes               ${DIM}# compare project/global installs${RESET}
  ${DIM}$${RESET} skills ls -a claude-code             ${DIM}# filter by agent${RESET}
  ${DIM}$${RESET} skills ls --json                      ${DIM}# JSON output${RESET}
  ${DIM}$${RESET} skills find                          ${DIM}# interactive search${RESET}
  ${DIM}$${RESET} skills find typescript               ${DIM}# search by keyword${RESET}
  ${DIM}$${RESET} skills update
  ${DIM}$${RESET} skills update my-skill             ${DIM}# update a single skill${RESET}
  ${DIM}$${RESET} skills update -g                    ${DIM}# update global skills only${RESET}
  ${DIM}$${RESET} skills init my-skill
  ${DIM}$${RESET} skills inspect
  ${DIM}$${RESET} skills publish
  ${DIM}$${RESET} skills import-local ./legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0
  ${DIM}$${RESET} skills find pdf --json                ${DIM}# machine-readable search results${RESET}
`);
}

function showRemoveHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills remove [skills...] [options]

${BOLD}Description:${RESET}
  Remove installed skills from agents. If no skill names are provided,
  an interactive selection menu will be shown.

${BOLD}Arguments:${RESET}
  skills            Optional skill names to remove (space-separated)

${BOLD}Options:${RESET}
  -g, --global       Remove from global scope (~/) instead of project scope
  -a, --agent        Remove from specific agents (use '*' for all agents)
  -s, --skill        Specify skills to remove (use '*' for all skills)
  -y, --yes          Skip confirmation prompts
  --all              Shorthand for --skill '*' --agent '*' -y

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills remove                           ${DIM}# interactive selection${RESET}
  ${DIM}$${RESET} skills remove my-skill                   ${DIM}# remove specific skill${RESET}
  ${DIM}$${RESET} skills remove skill1 skill2 -y           ${DIM}# remove multiple skills${RESET}
  ${DIM}$${RESET} skills remove --global my-skill          ${DIM}# remove from global scope${RESET}
  ${DIM}$${RESET} skills rm --agent claude-code my-skill   ${DIM}# remove from specific agent${RESET}
  ${DIM}$${RESET} skills remove --all                      ${DIM}# remove all skills${RESET}
  ${DIM}$${RESET} skills remove --skill '*' -a cursor      ${DIM}# remove all skills from cursor${RESET}
`);
}

function showAddHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills add <package_name> [options]

${BOLD}Description:${RESET}
  Add a skill package from the registry.

${BOLD}Arguments:${RESET}
  package_name      Registry package identity, e.g. @company.skill/pdf-skill

${BOLD}Options:${RESET}
  -g, --global           Install skill globally (user-level) instead of project-level
  -a, --agent <agents>   Specify agents to install to (use '*' for all agents)
  -y, --yes              Skip confirmation prompts
  --registry <url>       Catalog base URL override (default env: SKILLS_REGISTRY_URL)
  --version <version>    Install an exact version from the registry
  --copy                 Copy files instead of symlinking to agent directories

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills add @company.skill/pdf-skill
  ${DIM}$${RESET} skills add @company.skill/pdf-skill --version 1.2.0
  ${DIM}$${RESET} skills add @company.skill/pdf-skill --registry https://catalog.example.com
`);
}

function showListHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills list [options]

${BOLD}Description:${RESET}
  List installed skills.

${BOLD}Options:${RESET}
  -g, --global           List global skills (default: project)
  -p, --project          List project skills explicitly
  --all-scopes           Show project and global installs together with winner
  -a, --agent <agents>   Filter by specific agents
  --json                 Output as JSON (machine-readable, no ANSI codes)

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills list
  ${DIM}$${RESET} skills ls -g
  ${DIM}$${RESET} skills ls -a claude-code
  ${DIM}$${RESET} skills ls --all-scopes
`);
}

function showFindHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills find [query] [options]

${BOLD}Description:${RESET}
  Search for skills interactively or by keyword.

${BOLD}Options:${RESET}
  --registry <url>       Catalog base URL override (default env: SKILLS_REGISTRY_URL)
  --json                 Output query results as JSON

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills find
  ${DIM}$${RESET} skills find typescript
  ${DIM}$${RESET} skills find pdf --json
`);
}

function showUpdateHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills update [skills...] [options]

${BOLD}Description:${RESET}
  Update installed skills to latest versions.

${BOLD}Arguments:${RESET}
  skills            Optional package_name values to update

${BOLD}Options:${RESET}
  -g, --global           Update global skills only
  -p, --project          Update project skills only
  -y, --yes              Skip scope prompt

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills update
  ${DIM}$${RESET} skills update @company.skill/pdf-skill
  ${DIM}$${RESET} skills update -g
`);
}

function showInspectHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills inspect [path] [options]

${BOLD}Description:${RESET}
  Validate a local skill directory using the built-in inspection flow.

${BOLD}Arguments:${RESET}
  path              Optional path to a local skill directory (default: current directory)

${BOLD}Options:${RESET}
  --registry <url>  Catalog base URL override for registry-backed packaging validation
  --strict          Disable imported workspace normalization during prepare

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills inspect
  ${DIM}$${RESET} skills inspect ./my-skill
  ${DIM}$${RESET} skills inspect ./my-skill --strict
  ${DIM}$${RESET} skills inspect ./my-skill --registry https://catalog.example.com
`);
}

function showPublishHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills publish [path] [options]

${BOLD}Description:${RESET}
  Validate, pack, and publish a local skill directory.

${BOLD}Arguments:${RESET}
  path              Optional path to a local skill directory (default: current directory)

${BOLD}Options:${RESET}
  --registry <url>  Catalog base URL override for packaging and release-record calls
  --strict          Disable imported workspace normalization during prepare

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills publish
  ${DIM}$${RESET} skills publish ./my-skill
  ${DIM}$${RESET} skills publish ./my-skill --strict
  ${DIM}$${RESET} skills publish ./my-skill --registry https://catalog.example.com
`);
}

function showImportLocalHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills import-local [path] [options]

${BOLD}Description:${RESET}
  Upload a local source directory to Catalog for normalization/import, then publish it.

${BOLD}Arguments:${RESET}
  path                   Optional path to a local source directory (default: current directory)

${BOLD}Options:${RESET}
  --registry <url>       Catalog base URL override
  --package-name <name>  Override/inject package_name when package.json is missing
  --version <version>    Override/inject version when package.json is missing
  --publish-registry     Registry recorded into package metadata and release-record
  --npm-registry         Registry passed to npm publish
  --strict               Disable imported workspace normalization during prepare

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills import-local ./legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0
  ${DIM}$${RESET} skills import-local ./legacy-skill --registry https://catalog.example.com
`);
}

function showInitHelp(): void {
  console.log(`
${BOLD}Usage:${RESET} skills init [name]

${BOLD}Description:${RESET}
  Initialize a skill scaffold in the current directory or a new subdirectory.

${BOLD}Arguments:${RESET}
  name              Optional skill directory and default skill name

${BOLD}Examples:${RESET}
  ${DIM}$${RESET} skills init
  ${DIM}$${RESET} skills init my-skill

${BOLD}Output:${RESET}
  Creates:
  - SKILL.md
  - package.json
  - commands/

  Default package identity:
  - @company.skill/my-skill
`);
}

function runInit(args: string[]): void {
  const cwd = process.cwd();
  const skillName = args[0] || basename(cwd);
  const hasName = args[0] !== undefined;

  const skillDir = hasName ? join(cwd, skillName) : cwd;
  const skillFile = join(skillDir, 'SKILL.md');
  const packageJsonFile = join(skillDir, 'package.json');
  const commandsDir = join(skillDir, 'commands');
  const displayPath = hasName ? `${skillName}/SKILL.md` : 'SKILL.md';
  const displayPackageJsonPath = hasName ? `${skillName}/package.json` : 'package.json';
  const displayCommandsPath = hasName ? `${skillName}/commands/` : 'commands/';

  if (existsSync(skillFile)) {
    console.log(`${TEXT}Skill already exists at ${DIM}${displayPath}${RESET}`);
    return;
  }

  if (hasName) {
    mkdirSync(skillDir, { recursive: true });
  }
  mkdirSync(commandsDir, { recursive: true });

  const skillContent = `---
name: ${skillName}
description: A brief description of what this skill does
---

# ${skillName}

Instructions for the agent to follow when this skill is activated.

## When to use

Describe when this skill should be used.

## Instructions

1. First step
2. Second step
3. Additional steps as needed
`;

  const packageJsonContent = JSON.stringify(
    {
      name: `@company.skill/${skillName}`,
      version: '0.1.0',
    },
    null,
    2
  );

  writeFileSync(skillFile, skillContent);
  writeFileSync(packageJsonFile, `${packageJsonContent}\n`);

  console.log(`${TEXT}Initialized skill: ${DIM}${skillName}${RESET}`);
  console.log();
  console.log(`${DIM}Created:${RESET}`);
  console.log(`  ${displayPath}`);
  console.log(`  ${displayPackageJsonPath}`);
  console.log(`  ${displayCommandsPath}`);
  console.log();
  console.log(`${DIM}Next:${RESET}`);
  console.log(`  - Edit ${TEXT}${displayPath}${RESET}`);
  console.log(`  - Run ${TEXT}skills inspect${RESET}`);
  console.log(`  - Run ${TEXT}skills publish${RESET}`);
  console.log();
}

// ============================================
// Main
// ============================================

async function main(): Promise<void> {
  const args = process.argv.slice(2);

  if (args.length === 0) {
    showHelp();
    return;
  }

  const command = args[0];
  const restArgs = args.slice(1);

  switch (command) {
    case 'find':
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showFindHelp();
        break;
      }
      await runFind(restArgs);
      break;
    case 'init':
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showInitHelp();
        break;
      }
      runInit(restArgs);
      break;
    case 'inspect':
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showInspectHelp();
        break;
      }
      await runInspect(restArgs);
      break;
    case 'publish':
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showPublishHelp();
        break;
      }
      await runPublish(restArgs);
      break;
    case 'import-local':
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showImportLocalHelp();
        break;
      }
      await runImportLocal(restArgs);
      break;
    case 'a':
    case 'add': {
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showAddHelp();
        break;
      }
      const { source: addSource, options: addOpts } = parseAddOptions(restArgs);
      await runAdd(addSource, addOpts);
      break;
    }
    case 'remove':
    case 'rm':
      // Check for --help or -h flag
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showRemoveHelp();
        break;
      }
      const { skills, options: removeOptions } = parseRemoveOptions(restArgs);
      await removeCommand(skills, removeOptions);
      break;
    case 'list':
    case 'ls':
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showListHelp();
        break;
      }
      await runList(restArgs);
      break;
    case 'update':
      if (restArgs.includes('--help') || restArgs.includes('-h')) {
        showUpdateHelp();
        break;
      }
      await runUpdate(restArgs);
      break;
    case '--help':
    case '-h':
      showHelp();
      break;
    case '--version':
    case '-v':
      console.log(VERSION);
      break;

    default:
      console.log(`Unknown command: ${command}`);
      console.log(`Run ${BOLD}skills --help${RESET} for usage.`);
      process.exitCode = 1;
  }
}

main()
  .catch((error) => {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  })
  .finally(() => flushTelemetry().then(() => process.exit(process.exitCode ?? 0)));
