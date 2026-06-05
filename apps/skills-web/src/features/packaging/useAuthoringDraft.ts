export type AuthoringTemplateId = 'prompt' | 'command' | 'workflow';

export type AuthoringFile = {
  path: string;
  content: string;
  locked: boolean;
};

export type AuthoringDraft = {
  template: AuthoringTemplateId;
  files: AuthoringFile[];
};

const AUTHORING_TEMPLATES: Record<AuthoringTemplateId, AuthoringFile[]> = {
  prompt: [{ path: 'prompts/main.md', content: 'Write the core prompt here.\n', locked: true }],
  command: [{ path: 'commands/run.txt', content: 'echo authored\n', locked: true }],
  workflow: [{ path: 'workflows/main.md', content: '1. First step\n2. Second step\n', locked: true }],
};

export function createDefaultAuthoringDraft(template: AuthoringTemplateId): AuthoringDraft {
  return {
    template,
    files: AUTHORING_TEMPLATES[template].map((file) => ({ ...file })),
  };
}

export function draftFilesToRecord(files: AuthoringFile[]) {
  return Object.fromEntries(files.map((file) => [file.path, file.content]));
}

export function parseAuthoringJson(raw: string) {
  const parsed = JSON.parse(raw) as Record<string, unknown>;
  return Object.fromEntries(
    Object.entries(parsed).map(([path, value]) => {
      if (typeof value !== 'string') {
        throw new Error(`Inline file ${path} 的内容必须是字符串。`);
      }
      return [path, value];
    })
  );
}

export function validateAuthoringPath(path: string, existing: AuthoringFile[]) {
  const trimmed = path.trim();
  if (trimmed === '') return '新文件路径不能为空。';
  if (trimmed.startsWith('/')) return '文件路径必须是相对路径。';
  if (trimmed === 'SKILL.md') return '根级 SKILL.md 由服务端生成，不能手动添加。';
  if (trimmed.split('/').some((segment) => segment === '..')) return '文件路径不能包含 ..。';
  if (existing.some((file) => file.path === trimmed)) return '文件路径不能重复。';
  return null;
}
