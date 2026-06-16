import { useState } from 'react';
import {
  createAuthoringEditorState,
  getDefaultAuthoringPath,
  hasUnsavedAuthoringChanges,
  parseOptionalAuthoringJson,
  serializeAuthoringFiles,
  validateAuthoringPath,
  type AuthoringFile,
  type AuthoringTemplateId,
} from './useAuthoringDraft';

export function useAuthoringEditor(initialTemplate: AuthoringTemplateId) {
  const [authoringTemplate, setAuthoringTemplate] = useState<AuthoringTemplateId>(initialTemplate);
  const [authoringFiles, setAuthoringFiles] = useState<AuthoringFile[]>(
    () => createAuthoringEditorState(initialTemplate).draft.files
  );
  const [selectedAuthoringPath, setSelectedAuthoringPath] = useState(
    () => createAuthoringEditorState(initialTemplate).selectedPath
  );
  const [newAuthoringPath, setNewAuthoringPath] = useState('');
  const [authoringError, setAuthoringError] = useState<string | null>(null);
  const [showAdvancedFiles, setShowAdvancedFiles] = useState(false);
  const [rawInlineFiles, setRawInlineFiles] = useState('');
  const [rawInlineFilesDirty, setRawInlineFilesDirty] = useState(false);

  const selectedAuthoringFile =
    authoringFiles.find((file) => file.path === selectedAuthoringPath) || authoringFiles[0] || null;
  const advancedInlineFilesValue = rawInlineFilesDirty ? rawInlineFiles : serializeAuthoringFiles(authoringFiles);

  function resetAuthoringTemplate(nextTemplate: AuthoringTemplateId) {
    const hasUnsavedChanges = hasUnsavedAuthoringChanges({
      authoringFiles,
      rawInlineFilesDirty,
      template: authoringTemplate,
    });

    if (
      hasUnsavedChanges &&
      !window.confirm('切换模板会重建当前文件列表，未保存的作者输入会丢失。继续吗？')
    ) {
      return;
    }

    const nextAuthoringState = createAuthoringEditorState(nextTemplate);
    setAuthoringTemplate(nextTemplate);
    setAuthoringFiles(nextAuthoringState.draft.files);
    setSelectedAuthoringPath(nextAuthoringState.selectedPath);
    setAuthoringError(null);
    setRawInlineFiles('');
    setRawInlineFilesDirty(false);
    setNewAuthoringPath('');
  }

  function handleAddAuthoringFile() {
    const validationError = validateAuthoringPath(newAuthoringPath, authoringFiles);
    if (validationError) {
      setAuthoringError(validationError);
      return;
    }

    const trimmed = newAuthoringPath.trim();
    const nextFiles = [...authoringFiles, { path: trimmed, content: '', locked: false }];
    setAuthoringFiles(nextFiles);
    setSelectedAuthoringPath(trimmed);
    setAuthoringError(null);
    setNewAuthoringPath('');
    setRawInlineFilesDirty(false);
  }

  function handleDeleteAuthoringFile(path: string) {
    const target = authoringFiles.find((file) => file.path === path);
    if (!target || target.locked) {
      return;
    }

    const nextFiles = authoringFiles.filter((file) => file.path !== path);
    setAuthoringFiles(nextFiles);
    setAuthoringError(null);
    setRawInlineFilesDirty(false);

    if (selectedAuthoringPath === path) {
      setSelectedAuthoringPath(getDefaultAuthoringPath(nextFiles));
    }
  }

  function handleAuthoringContentChange(content: string) {
    const activePath = selectedAuthoringFile?.path;
    if (!activePath) {
      return;
    }

    setAuthoringFiles((current) =>
      current.map((file) => (file.path === activePath ? { ...file, content } : file))
    );
    setAuthoringError(null);
    setRawInlineFilesDirty(false);
  }

  function applyRawInlineFiles() {
    const nextAuthoringState = createAuthoringEditorState(
      authoringTemplate,
      parseOptionalAuthoringJson(rawInlineFiles) || {}
    );
    setAuthoringFiles(nextAuthoringState.draft.files);
    setSelectedAuthoringPath(nextAuthoringState.selectedPath);
    setAuthoringError(null);
    setRawInlineFilesDirty(false);
  }

  return {
    authoringTemplate,
    setAuthoringTemplate,
    authoringFiles,
    setAuthoringFiles,
    selectedAuthoringPath,
    setSelectedAuthoringPath,
    selectedAuthoringFile,
    newAuthoringPath,
    setNewAuthoringPath,
    authoringError,
    setAuthoringError,
    showAdvancedFiles,
    setShowAdvancedFiles,
    rawInlineFiles,
    setRawInlineFiles,
    rawInlineFilesDirty,
    setRawInlineFilesDirty,
    advancedInlineFilesValue,
    resetAuthoringTemplate,
    handleAddAuthoringFile,
    handleDeleteAuthoringFile,
    handleAuthoringContentChange,
    applyRawInlineFiles,
  };
}
