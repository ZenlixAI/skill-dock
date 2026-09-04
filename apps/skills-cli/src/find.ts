import * as readline from 'readline';
import { runAdd } from './add.ts';
import { CatalogError, searchCatalogSkillsPage } from './catalog.ts';
import { sanitizeMetadata } from './sanitize.ts';
import { track } from './telemetry.ts';
import { isRunningInAgent } from './detect-agent.ts';

const RESET = '\x1b[0m';
const BOLD = '\x1b[1m';
const DIM = '\x1b[38;5;102m';
const TEXT = '\x1b[38;5;145m';
const CYAN = '\x1b[36m';
const MAX_QUERY_RESULTS_DISPLAY = 6;

export interface SearchSkill {
  packageName: string;
  displayName: string;
  description: string;
  latestVersion?: string;
  sourceKind?: string;
  originRegistryUrl?: string;
  originRepositoryUrl?: string;
  learnMoreUrl?: string;
  installable?: boolean;
  hosted?: boolean;
  mirrored?: boolean;
}

interface FindOptions {
  query: string;
  registry?: string;
  includeExternal?: boolean;
  source?: 'catalog' | 'skills.sh' | 'npm' | 'git' | 'local' | 'all';
  scope?: string;
  origin?: string;
  json?: boolean;
}

interface SearchSkillsResult {
  items: SearchSkill[];
  hasMoreCatalogResults: boolean;
}

export function parseFindOptions(args: string[]): FindOptions {
  const queryParts: string[] = [];
  let registry: string | undefined;
  let includeExternal = false;
  let source: FindOptions['source'];
  let scope: string | undefined;
  let origin: string | undefined;
  let json = false;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--registry' && i + 1 < args.length) {
      registry = args[++i];
      continue;
    }
    if (arg === '--include-external') {
      includeExternal = true;
      continue;
    }
    if (arg === '--source' && i + 1 < args.length) {
      const value = args[++i] as FindOptions['source'];
      source = value;
      continue;
    }
    if (arg === '--scope' && i + 1 < args.length) {
      scope = args[++i];
      continue;
    }
    if (arg === '--origin' && i + 1 < args.length) {
      origin = args[++i];
      continue;
    }
    if (arg === '--json') {
      json = true;
      continue;
    }

    queryParts.push(arg!);
  }

  return {
    query: queryParts.join(' ').trim(),
    registry,
    json,
    ...(includeExternal ? { includeExternal } : {}),
    ...(source ? { source } : {}),
    ...(scope ? { scope } : {}),
    ...(origin ? { origin } : {}),
  };
}

async function searchSkillsCatalog(query: string, options: FindOptions): Promise<SearchSkillsResult> {
  const results: SearchSkill[] = [];
  let cursor: string | undefined;
  let hasMoreCatalogResults = false;

  for (let page = 0; page < 3; page++) {
    const response = await searchCatalogSkillsPage(query, {
      registry: options.registry,
      pageSize: 10,
      cursor,
      ...(options.includeExternal ? { includeExternal: true } : {}),
      ...(options.source ? { source: options.source } : {}),
      ...(options.scope ? { scope: options.scope } : {}),
      ...(options.origin ? { origin: options.origin } : {}),
    });

    results.push(
      ...response.items.map((item) => ({
        packageName: sanitizeMetadata(item.packageName),
        displayName: sanitizeMetadata(item.displayName),
        description: sanitizeMetadata(item.description || ''),
        latestVersion: item.latestVersion ? sanitizeMetadata(item.latestVersion) : undefined,
        sourceKind: item.sourceKind || 'catalog',
        originRegistryUrl: item.originRegistryUrl,
        originRepositoryUrl: item.originRepositoryUrl,
        learnMoreUrl: item.learnMoreUrl,
        installable: item.installable ?? true,
        hosted: item.hosted ?? true,
        mirrored: item.mirrored ?? false,
      }))
    );

    if (!response.pageInfo.hasNextPage || !response.pageInfo.nextCursor) {
      hasMoreCatalogResults = false;
      break;
    }
    cursor = response.pageInfo.nextCursor;
    hasMoreCatalogResults = true;
  }

  return { items: results, hasMoreCatalogResults };
}

export async function searchSkillsAPI(query: string, registry?: string): Promise<SearchSkill[]> {
  const result = await searchSkillsCatalog(query, { registry });
  return result.items;
}

async function searchSkills(query: string, options: FindOptions): Promise<SearchSkillsResult> {
  return searchSkillsCatalog(query, options);
}

const HIDE_CURSOR = '\x1b[?25l';
const SHOW_CURSOR = '\x1b[?25h';
const CLEAR_DOWN = '\x1b[J';
const MOVE_UP = (n: number) => `\x1b[${n}A`;
const MOVE_TO_COL = (n: number) => `\x1b[${n}G`;

async function runSearchPrompt(initialQuery = '', options: FindOptions): Promise<SearchSkill | null> {
  let results: SearchSkill[] = [];
  let selectedIndex = 0;
  let query = initialQuery;
  let loading = false;
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let lastRenderedLines = 0;

  if (process.stdin.isTTY) {
    process.stdin.setRawMode(true);
  }

  readline.emitKeypressEvents(process.stdin);
  process.stdin.resume();
  process.stdout.write(HIDE_CURSOR);

  function render(): void {
    if (lastRenderedLines > 0) {
      process.stdout.write(MOVE_UP(lastRenderedLines) + MOVE_TO_COL(1));
    }

    process.stdout.write(CLEAR_DOWN);

    const lines: string[] = [];
    lines.push(`${TEXT}Search skills:${RESET} ${query}${BOLD}_${RESET}`);
    lines.push('');

    if (!query || query.length < 2) {
      lines.push(`${DIM}Start typing to search (min 2 chars)${RESET}`);
    } else if (results.length === 0 && loading) {
      lines.push(`${DIM}Searching...${RESET}`);
    } else if (results.length === 0) {
      lines.push(`${DIM}No skills found${RESET}`);
    } else {
      for (const [index, skill] of results.slice(0, 8).entries()) {
        const isSelected = index === selectedIndex;
        const arrow = isSelected ? `${BOLD}>${RESET}` : ' ';
        const label = isSelected
          ? `${BOLD}${skill.displayName}${RESET}`
          : `${TEXT}${skill.displayName}${RESET}`;
        const pkg = `${DIM}${skill.packageName}${RESET}`;
        const version = skill.latestVersion ? ` ${CYAN}v${skill.latestVersion}${RESET}` : '';
        const source = skill.sourceKind && skill.sourceKind !== 'catalog' ? ` ${DIM}[${skill.sourceKind}]${RESET}` : '';
        lines.push(`  ${arrow} ${label} ${pkg}${source}${version}`);
        if (skill.description) {
          lines.push(`    ${DIM}${skill.description}${RESET}`);
        }
      }
    }

    lines.push('');
    lines.push(`${DIM}up/down navigate | enter select | esc cancel${RESET}`);

    for (const line of lines) {
      process.stdout.write(line + '\n');
    }

    lastRenderedLines = lines.length;
  }

  function triggerSearch(nextQuery: string): void {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }

    loading = false;

    if (!nextQuery || nextQuery.length < 2) {
      results = [];
      selectedIndex = 0;
      render();
      return;
    }

    loading = true;
    render();

    debounceTimer = setTimeout(async () => {
      try {
        results = (await searchSkills(query, options)).items;
        selectedIndex = 0;
      } catch {
        results = [];
      } finally {
        loading = false;
        debounceTimer = null;
        render();
      }
    }, Math.max(150, 350 - nextQuery.length * 50));
  }

  if (initialQuery) {
    triggerSearch(initialQuery);
  }
  render();

  return new Promise((resolve) => {
    function cleanup(): void {
      process.stdin.removeListener('keypress', handleKeypress);
      if (process.stdin.isTTY) {
        process.stdin.setRawMode(false);
      }
      process.stdout.write(SHOW_CURSOR);
      process.stdin.pause();
    }

    function handleKeypress(_ch: string | undefined, key: readline.Key): void {
      if (!key) return;

      if (key.name === 'escape' || (key.ctrl && key.name === 'c')) {
        cleanup();
        resolve(null);
        return;
      }

      if (key.name === 'return') {
        cleanup();
        resolve(results[selectedIndex] || null);
        return;
      }

      if (key.name === 'up') {
        selectedIndex = Math.max(0, selectedIndex - 1);
        render();
        return;
      }

      if (key.name === 'down') {
        selectedIndex = Math.min(Math.max(0, results.length - 1), selectedIndex + 1);
        render();
        return;
      }

      if (key.name === 'backspace') {
        if (query.length > 0) {
          query = query.slice(0, -1);
          triggerSearch(query);
        }
        return;
      }

      if (key.sequence && !key.ctrl && !key.meta && key.sequence.length === 1) {
        const char = key.sequence;
        if (char >= ' ' && char <= '~') {
          query += char;
          triggerSearch(query);
        }
      }
    }

    process.stdin.on('keypress', handleKeypress);
  });
}

export async function runFind(args: string[]): Promise<void> {
  const options = parseFindOptions(args);
  const query = options.query;
  const isNonInteractive = !process.stdin.isTTY;
  const agentTip = `${DIM}Tip: if running in a coding agent, follow these steps:${RESET}
${DIM}  1) npx skills find [query]${RESET}
${DIM}  2) npx skills add <package_name>${RESET}`;

  if (query) {
    let searchResult: SearchSkillsResult;
    try {
      searchResult = await searchSkills(query, options);
    } catch (error) {
      process.exitCode = 1;
      if (options.json) {
        console.log(
          JSON.stringify(
            {
              error: {
                message: formatFindError(error),
              },
            },
            null,
            2
          )
        );
      } else {
        console.log(`${DIM}Find failed:${RESET} ${formatFindError(error)}`);
      }
      return;
    }
    const results = searchResult.items;

    track({
      event: 'find',
      query,
      resultCount: String(results.length),
    });

    if (results.length === 0) {
      if (options.json) {
        console.log(
          JSON.stringify(
            {
              items: [],
              total: 0,
              truncated: false,
            },
            null,
            2
          )
        );
      } else {
        console.log(`${DIM}No skills found for "${query}".${RESET}`);
      }
      return;
    }

    if (options.json) {
      console.log(
        JSON.stringify(
          {
            items: results,
            total: results.length,
            truncated: searchResult.hasMoreCatalogResults,
          },
          null,
          2
        )
      );
      return;
    }

    console.log(`${DIM}Install with:${RESET} npx skills add <package_name>`);
    console.log();

    for (const skill of results.slice(0, MAX_QUERY_RESULTS_DISPLAY)) {
      const version = skill.latestVersion ? ` ${CYAN}v${skill.latestVersion}${RESET}` : '';
      const source = skill.sourceKind && skill.sourceKind !== 'catalog' ? ` ${DIM}[${skill.sourceKind}]${RESET}` : '';
      console.log(`${TEXT}${skill.packageName}${RESET}${source}${version}`);
      if (skill.description) {
        console.log(`${DIM}└ ${skill.description}${RESET}`);
      }
      console.log();
    }
    if (results.length > MAX_QUERY_RESULTS_DISPLAY || searchResult.hasMoreCatalogResults) {
      const totalLabel = searchResult.hasMoreCatalogResults
        ? `at least ${results.length}`
        : String(results.length);
      console.log(
        `${DIM}Showing first ${MAX_QUERY_RESULTS_DISPLAY} of ${totalLabel} results. Refine your query to narrow the list.${RESET}`
      );
    }
    return;
  }

  if (isNonInteractive || (await isRunningInAgent())) {
    console.log(agentTip);
    console.log();
    console.log(`${DIM}Usage: npx skills find <query> [--registry <url>] [--include-external] [--source <catalog|skills.sh|npm|all>] [--json]${RESET}`);
    console.log(`${DIM}Default catalog env:${RESET} SKILLS_REGISTRY_URL`);
    return;
  }

  const selected = await runSearchPrompt(query, options);

  track({
    event: 'find',
    query: '',
    resultCount: selected ? '1' : '0',
    interactive: '1',
  });

  if (!selected) {
    console.log(`${DIM}Search cancelled${RESET}`);
    console.log();
    return;
  }

  console.log();
  console.log(
    `${TEXT}Installing ${BOLD}${selected.displayName}${RESET} from ${DIM}${selected.packageName}${RESET}...`
  );
  console.log();

  await runAdd(
    [selected.packageName],
    options.registry ? { registry: options.registry } : {}
  );

  console.log();
  console.log(`${DIM}Discover more skills in the configured catalog.${RESET}`);
  console.log();
}

function formatFindError(error: unknown): string {
  if (error instanceof CatalogError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'Unknown error';
}
