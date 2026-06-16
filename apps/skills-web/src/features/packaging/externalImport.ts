import { gzipSync } from 'fflate';

const IMPORT_REPORT_PATH = '.skill-packaging-import-report.json';
const IGNORED_SEGMENTS = new Set(['.git', '.dist']);
const GITHUB_API_ACCEPT = 'application/vnd.github+json';
const EXTERNAL_IMPORT_FETCH_ERROR = '无法获取外部 skill 源，请检查地址、仓库可见性或浏览器网络限制。';
const USER_FACING_ERRORS = new Set([
  EXTERNAL_IMPORT_FETCH_ERROR,
  '目前只支持 GitHub HTTP 地址导入。',
  'GitHub 地址必须至少包含 owner/repo。',
  'GitHub 地址必须带上 ?path=skills/... 指向具体 skill 目录。',
  '外部 skill 源里缺少根 SKILL.md。',
]);

interface GitHubLocation {
  owner: string;
  repo: string;
  subpath: string;
  ref?: string;
}

interface GitHubContentEntry {
  type: 'file' | 'dir';
  path: string;
  download_url?: string | null;
  url?: string;
}

export interface ExternalImportArchive {
  archiveBase64: string;
}

export async function buildExternalImportArchive(
  location: string,
  ref?: string
): Promise<ExternalImportArchive> {
  try {
    const source = parseGitHubLocation(location, ref);
    const files = await fetchGitHubSkillFiles(source);
    const tarBytes = createTarArchive(files);
    const gzipBytes = gzipSync(tarBytes);
    return { archiveBase64: bytesToBase64(gzipBytes) };
  } catch (error) {
    if (error instanceof Error && USER_FACING_ERRORS.has(error.message)) {
      throw error;
    }
    throw new Error(EXTERNAL_IMPORT_FETCH_ERROR);
  }
}

function parseGitHubLocation(location: string, ref?: string): GitHubLocation {
  let parsed: URL;
  try {
    parsed = new URL(location.trim());
  } catch {
    throw new Error(EXTERNAL_IMPORT_FETCH_ERROR);
  }

  if (parsed.hostname !== 'github.com') {
    throw new Error('目前只支持 GitHub HTTP 地址导入。');
  }

  const segments = parsed.pathname.split('/').filter(Boolean);
  if (segments.length < 2) {
    throw new Error('GitHub 地址必须至少包含 owner/repo。');
  }

  const subpath = parsed.searchParams.get('path')?.trim().replace(/^\/+|\/+$/g, '') || '';
  if (!subpath) {
    throw new Error('GitHub 地址必须带上 ?path=skills/... 指向具体 skill 目录。');
  }

  return {
    owner: segments[0]!,
    repo: segments[1]!.replace(/\.git$/, ''),
    subpath,
    ref: ref?.trim() || undefined,
  };
}

async function fetchGitHubSkillFiles(source: GitHubLocation): Promise<Record<string, string>> {
  const files: Record<string, string> = {};
  await walkGitHubPath(buildGitHubContentsURL(source, source.subpath), source, files);

  if (!Object.keys(files).includes('SKILL.md')) {
    throw new Error('外部 skill 源里缺少根 SKILL.md。');
  }

  return files;
}

async function walkGitHubPath(
  url: string,
  source: GitHubLocation,
  files: Record<string, string>
): Promise<void> {
  const response = await fetch(url, {
    headers: {
      Accept: GITHUB_API_ACCEPT,
    },
  });
  if (!response.ok) {
    throw new Error(EXTERNAL_IMPORT_FETCH_ERROR);
  }

  const payload = (await response.json()) as GitHubContentEntry[] | GitHubContentEntry;
  const entries = Array.isArray(payload) ? payload : [payload];

  for (const entry of entries) {
    if (!entry || typeof entry.path !== 'string') {
      continue;
    }

    if (entry.type === 'dir') {
      const nextUrl = appendRef(entry.url || buildGitHubContentsURL(source, entry.path), source.ref);
      await walkGitHubPath(nextUrl, source, files);
      continue;
    }

    if (entry.type !== 'file' || !entry.download_url) {
      continue;
    }

    const relativePath = trimSelectedPrefix(source.subpath, entry.path);
    if (!relativePath || shouldIgnorePackagingPath(relativePath)) {
      continue;
    }

    const fileResponse = await fetch(entry.download_url);
    if (!fileResponse.ok) {
      throw new Error(EXTERNAL_IMPORT_FETCH_ERROR);
    }
    files[relativePath] = await fileResponse.text();
  }
}

function buildGitHubContentsURL(source: GitHubLocation, path: string): string {
  const url = new URL(
    `https://api.github.com/repos/${encodeURIComponent(source.owner)}/${encodeURIComponent(source.repo)}/contents/${path
      .split('/')
      .map((segment) => encodeURIComponent(segment))
      .join('/')}`
  );
  if (source.ref) {
    url.searchParams.set('ref', source.ref);
  }
  return url.toString();
}

function appendRef(rawURL: string, ref?: string): string {
  if (!ref) {
    return rawURL;
  }
  const parsed = new URL(rawURL);
  parsed.searchParams.set('ref', ref);
  return parsed.toString();
}

function trimSelectedPrefix(prefix: string, fullPath: string): string {
  const normalizedPrefix = prefix.replace(/^\/+|\/+$/g, '');
  if (fullPath === normalizedPrefix) {
    return '';
  }
  if (!fullPath.startsWith(`${normalizedPrefix}/`)) {
    return fullPath;
  }
  return fullPath.slice(normalizedPrefix.length + 1);
}

function shouldIgnorePackagingPath(relativePath: string): boolean {
  const normalized = relativePath.trim().replace(/^\/+|\/+$/g, '');
  if (!normalized) {
    return false;
  }
  if (normalized === IMPORT_REPORT_PATH) {
    return true;
  }
  return normalized.split('/').some((segment) => IGNORED_SEGMENTS.has(segment));
}

function createTarArchive(files: Record<string, string>): Uint8Array {
  const encoder = new TextEncoder();
  const chunks: Uint8Array[] = [];

  for (const path of Object.keys(files).sort()) {
    const content = encoder.encode(files[path]!);
    chunks.push(createTarHeader(path, content.length));
    chunks.push(content);
    const remainder = content.length % 512;
    if (remainder !== 0) {
      chunks.push(new Uint8Array(512 - remainder));
    }
  }

  chunks.push(new Uint8Array(1024));

  const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  const archive = new Uint8Array(totalLength);
  let offset = 0;
  for (const chunk of chunks) {
    archive.set(chunk, offset);
    offset += chunk.length;
  }
  return archive;
}

function createTarHeader(path: string, size: number): Uint8Array {
  const header = new Uint8Array(512);
  writeString(header, 0, 100, path);
  writeOctal(header, 100, 8, 0o644);
  writeOctal(header, 108, 8, 0);
  writeOctal(header, 116, 8, 0);
  writeOctal(header, 124, 12, size);
  writeOctal(header, 136, 12, Math.floor(Date.now() / 1000));
  for (let index = 148; index < 156; index += 1) {
    header[index] = 0x20;
  }
  header[156] = '0'.charCodeAt(0);
  writeString(header, 257, 6, 'ustar');
  writeString(header, 263, 2, '00');

  let checksum = 0;
  for (const value of header) {
    checksum += value;
  }
  const checksumText = checksum.toString(8).padStart(6, '0');
  writeString(header, 148, 6, checksumText);
  header[154] = 0;
  header[155] = 0x20;

  return header;
}

function writeString(buffer: Uint8Array, offset: number, length: number, value: string): void {
  const bytes = new TextEncoder().encode(value);
  buffer.set(bytes.subarray(0, length), offset);
}

function writeOctal(buffer: Uint8Array, offset: number, length: number, value: number): void {
  const text = value.toString(8).padStart(length - 1, '0');
  writeString(buffer, offset, length - 1, text);
  buffer[offset + length - 1] = 0;
}

function bytesToBase64(bytes: Uint8Array): string {
  const globalBuffer = (globalThis as {
    Buffer?: {
      from(input: Uint8Array): { toString(encoding: 'base64'): string };
    };
  }).Buffer;
  if (globalBuffer) {
    return globalBuffer.from(bytes).toString('base64');
  }

  let binary = '';
  bytes.forEach((value) => {
    binary += String.fromCharCode(value);
  });
  return btoa(binary);
}
