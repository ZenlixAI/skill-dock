import { describe, expect, it } from 'vitest';
import {
  createDefaultAuthoringDraft,
  draftFilesToRecord,
  parseAuthoringJson,
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

  it('serializes files back into the request record shape', () => {
    const draft = createDefaultAuthoringDraft('workflow');

    expect(draftFilesToRecord(draft.files)).toEqual({
      'workflows/main.md': '1. First step\n2. Second step\n',
    });
  });
});
