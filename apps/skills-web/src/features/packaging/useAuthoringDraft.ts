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

export type AuthoringEditorState = {
  draft: AuthoringDraft;
  selectedPath: string;
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

export function createAuthoringFilesFromRecord(
  files: Record<string, string>,
  template: AuthoringTemplateId
): AuthoringFile[] {
  const templateLockedPaths = new Set(
    AUTHORING_TEMPLATES[template].map((file) => file.path)
  );

  return Object.entries(files).map(([path, content]) => ({
    path,
    content,
    locked: templateLockedPaths.has(path),
  }));
}

export function getDefaultAuthoringPath(files: AuthoringFile[]) {
  return files[0]?.path || '';
}

export function createAuthoringEditorState(
  template: AuthoringTemplateId,
  files?: Record<string, string>
): AuthoringEditorState {
  const draft =
    files === undefined
      ? createDefaultAuthoringDraft(template)
      : {
          template,
          files: createAuthoringFilesFromRecord(files, template),
        };

  return {
    draft,
    selectedPath: getDefaultAuthoringPath(draft.files),
  };
}

export function authoringFilesMatchTemplate(
  files: AuthoringFile[],
  template: AuthoringTemplateId
) {
  const templateFiles = AUTHORING_TEMPLATES[template];
  if (files.length !== templateFiles.length) {
    return false;
  }

  return files.every((file, index) => {
    const templateFile = templateFiles[index];
    return (
      templateFile !== undefined &&
      file.path === templateFile.path &&
      file.content === templateFile.content &&
      file.locked === templateFile.locked
    );
  });
}

export function hasUnsavedAuthoringChanges({
  authoringFiles,
  rawInlineFilesDirty,
  template,
}: {
  authoringFiles: AuthoringFile[];
  rawInlineFilesDirty: boolean;
  template: AuthoringTemplateId;
}) {
  return rawInlineFilesDirty || !authoringFilesMatchTemplate(authoringFiles, template);
}

export function serializeAuthoringFiles(files: AuthoringFile[]) {
  return JSON.stringify(draftFilesToRecord(files), null, 2);
}

export function parseAuthoringJson(raw: string) {
  const parsed = JSON.parse(raw.trim()) as unknown;
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Inline files 必须是 JSON object，例如 {"commands/run.txt":"echo hi"}。');
  }

  return Object.fromEntries(
    Object.entries(parsed).map(([path, value]) => {
      if (typeof value !== 'string') {
        throw new Error(`Inline file ${path} 的内容必须是字符串。`);
      }
      return [path, value];
    })
  );
}

export function parseOptionalAuthoringJson(raw: string) {
  if (raw.trim() === '') {
    return undefined;
  }

  return parseAuthoringJson(raw);
}

export function resolvePrepareAuthoringFiles({
  authoringFiles,
  rawInlineFiles,
  rawInlineFilesDirty,
}: {
  authoringFiles: AuthoringFile[];
  rawInlineFiles: string;
  rawInlineFilesDirty: boolean;
}) {
  if (!rawInlineFilesDirty) {
    return draftFilesToRecord(authoringFiles);
  }

  return parseOptionalAuthoringJson(rawInlineFiles) ?? draftFilesToRecord(authoringFiles);
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
