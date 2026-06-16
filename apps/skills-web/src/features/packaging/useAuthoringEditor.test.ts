import { act, renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { useAuthoringEditor } from './useAuthoringEditor';

describe('useAuthoringEditor', () => {
  it('applies raw files json into the editable file list and clears dirty state', () => {
    const { result } = renderHook(() => useAuthoringEditor('command'));

    act(() => {
      result.current.setRawInlineFiles('{\n  "docs/README.md": "hello\\n"\n}');
      result.current.setRawInlineFilesDirty(true);
    });

    act(() => {
      result.current.applyRawInlineFiles();
    });

    expect(result.current.rawInlineFilesDirty).toBe(false);
    expect(result.current.authoringError).toBeNull();
    expect(result.current.selectedAuthoringPath).toBe('docs/README.md');
    expect(result.current.authoringFiles).toEqual([
      {
        path: 'docs/README.md',
        content: 'hello\n',
        locked: false,
      },
    ]);
  });
});
