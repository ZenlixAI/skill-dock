import { gunzipSync } from 'fflate';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { buildExternalImportArchive } from './externalImport';

function base64ToBytes(value: string): Uint8Array {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function listTarEntries(buffer: Uint8Array): string[] {
  const entries: string[] = [];
  let offset = 0;

  while (offset + 512 <= buffer.length) {
    const header = buffer.subarray(offset, offset + 512);
    const name = readNullTerminatedString(header.subarray(0, 100));
    if (!name) {
      break;
    }

    entries.push(name);

    const sizeText = readNullTerminatedString(header.subarray(124, 136)).trim();
    const size = sizeText ? Number.parseInt(sizeText, 8) : 0;
    const dataSize = Math.ceil(size / 512) * 512;
    offset += 512 + dataSize;
  }

  return entries;
}

function readNullTerminatedString(bytes: Uint8Array): string {
  const end = bytes.indexOf(0);
  const slice = end >= 0 ? bytes.subarray(0, end) : bytes;
  return new TextDecoder().decode(slice).replace(/\0+$/g, '');
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe('buildExternalImportArchive', () => {
  it('packages a GitHub skill subtree into upload_archive content', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify([
            {
              type: 'file',
              path: 'skills/engineering/zoom-out/SKILL.md',
              download_url: 'https://raw.githubusercontent.com/example/skills/main/skills/engineering/zoom-out/SKILL.md',
            },
            {
              type: 'dir',
              path: 'skills/engineering/zoom-out/commands',
              url: 'https://api.github.com/repos/example/skills/contents/skills/engineering/zoom-out/commands',
            },
          ]),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response('---\nname: Zoom Out\ndescription: Broader context\n---\n', { status: 200 })
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify([
            {
              type: 'file',
              path: 'skills/engineering/zoom-out/commands/run.txt',
              download_url: 'https://raw.githubusercontent.com/example/skills/main/skills/engineering/zoom-out/commands/run.txt',
            },
          ]),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(new Response('echo external\n', { status: 200 }));

    const result = await buildExternalImportArchive(
      'https://github.com/example/skills?path=skills/engineering/zoom-out'
    );

    expect(result.archiveBase64).not.toBe('');

    const tarball = gunzipSync(base64ToBytes(result.archiveBase64));
    expect(listTarEntries(tarball)).toEqual(['SKILL.md', 'commands/run.txt']);
  });

  it('surfaces a stable user-facing error when external acquisition fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network down'));

    await expect(
      buildExternalImportArchive('https://github.com/example/skills?path=skills/engineering/zoom-out')
    ).rejects.toThrow('无法获取外部 skill 源，请检查地址、仓库可见性或浏览器网络限制。');
  });
});
