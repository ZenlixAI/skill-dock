import { describe, expect, it } from 'vitest';
import {
  createAuthoringFilesFromRecord,
  createDefaultAuthoringDraft,
  draftFilesToRecord,
  hasUnsavedAuthoringChanges,
  parseAuthoringJson,
  parseOptionalAuthoringJson,
  resolvePrepareAuthoringFiles,
  validateAuthoringPath,
} from './useAuthoringDraft';

describe('authoring draft', () => {
  it('starts the command template with a locked commands/run.txt file', () => {
    const draft = createDefaultAuthoringDraft('command');

    expect(draft.files).toEqual([
      {
        path: 'commands/run.txt',
        content: 'echo authored\n',
        locked: true,
      },
    ]);
  });

  it('rejects SKILL.md and parent-path entries', () => {
    const files = createDefaultAuthoringDraft('command').files;

    expect(validateAuthoringPath('SKILL.md', files)).toBe('根级 SKILL.md 由服务端生成，不能手动添加。');
    expect(validateAuthoringPath('../bad.txt', files)).toBe('文件路径不能包含 ..。');
  });

  it('parses raw json into a file record', () => {
    expect(parseAuthoringJson('{\n  "docs/README.md": "hello\\n"\n}')).toEqual({
      'docs/README.md': 'hello\n',
    });
  });

  it('rejects non-object json values', () => {
    expect(() => parseAuthoringJson('"text"')).toThrow(
      'Inline files 必须是 JSON object，例如 {"commands/run.txt":"echo hi"}。'
    );
  });

  it('rejects array json values', () => {
    expect(() => parseAuthoringJson('[]')).toThrow(
      'Inline files 必须是 JSON object，例如 {"commands/run.txt":"echo hi"}。'
    );
  });

  it('returns undefined for blank optional raw json input', () => {
    expect(parseOptionalAuthoringJson('   ')).toBeUndefined();
  });

  it('serializes files back into the request record shape', () => {
    const draft = createDefaultAuthoringDraft('workflow');

    expect(draftFilesToRecord(draft.files)).toEqual({
      'workflows/main.md': '1. First step\n2. Second step\n',
    });
  });

  it('reconstructs files from a record and restores template-locked entries', () => {
    expect(
      createAuthoringFilesFromRecord(
        {
          'commands/run.txt': 'echo customized\n',
          'docs/README.md': 'hello\n',
        },
        'command'
      )
    ).toEqual([
      {
        path: 'commands/run.txt',
        content: 'echo customized\n',
        locked: true,
      },
      {
        path: 'docs/README.md',
        content: 'hello\n',
        locked: false,
      },
    ]);
  });

  it('treats dirty raw json as an unsaved authoring change', () => {
    const draft = createDefaultAuthoringDraft('prompt');

    expect(
      hasUnsavedAuthoringChanges({
        authoringFiles: draft.files,
        rawInlineFilesDirty: true,
        template: 'prompt',
      })
    ).toBe(true);
  });

  it('uses dirty raw json for prepare without requiring apply', () => {
    expect(
      resolvePrepareAuthoringFiles({
        authoringFiles: createDefaultAuthoringDraft('command').files,
        rawInlineFiles: '{\n  "docs/README.md": "draft\\n"\n}',
        rawInlineFilesDirty: true,
      })
    ).toEqual({
      'docs/README.md': 'draft\n',
    });
  });
});
