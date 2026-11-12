import { useEffect, useRef } from 'react';
import * as monaco from 'monaco-editor';
import EditorWorker from 'monaco-editor/editor/editor.worker.js?worker';

self.MonacoEnvironment = { getWorker: () => new EditorWorker() };

type Content = { text: string; notes: Map<number, string> };
export type LineRange = { start: number; end: number };

export function Preview({ current, previous, focusLine, focusRange, highlightRanges }: {
  current: Content;
  previous: Content;
  focusLine?: number;
  focusRange?: LineRange;
  highlightRanges?: LineRange[];
}) {
  const container = useRef<HTMLDivElement>(null);
  const models = useRef<{ original: monaco.editor.ITextModel; modified: monaco.editor.ITextModel }>(null);
  const editorRef = useRef<monaco.editor.IStandaloneDiffEditor>(null);
  const currentNotes = useRef(current.notes);

  useEffect(() => {
    const original = monaco.editor.createModel('', 'yaml');
    const modified = monaco.editor.createModel('', 'yaml');
    const editor = monaco.editor.createDiffEditor(container.current!, {
      automaticLayout: true,
      readOnly: true,
      renderSideBySide: false,
      minimap: { enabled: false },
      diffWordWrap: 'on',
      accessibilityVerbose: true,
    });
    const hover = monaco.languages.registerHoverProvider('yaml', {
      provideHover(model, position) {
        const note = currentNotes.current.get(position.lineNumber);
        if (model === modified && note) return { contents: [{ value: note }] };
      },
    });
    editor.setModel({ original, modified });
    editorRef.current = editor;
    models.current = { original, modified };
    return () => {
      hover.dispose();
      editor.dispose();
      original.dispose();
      modified.dispose();
      models.current = null;
      editorRef.current = null;
    };
  }, []);

  useEffect(() => {
    currentNotes.current = current.notes;
    const { original, modified } = models.current!;
    if (original.getValue() !== previous.text) original.setValue(previous.text);
    if (modified.getValue() !== current.text) modified.setValue(current.text);
  }, [current, previous]);

  useEffect(() => {
    const editor = editorRef.current!;
    const modified = editor.getModifiedEditor();
    const ranges = highlightRanges ?? [];
    const decorations = modified.createDecorationsCollection(
      ranges.map(({ start, end }) => ({
        range: new monaco.Range(start, 1, end, 1),
        options: { isWholeLine: true, className: 'comparison-value-line' },
      })),
    );
    if (focusLine !== undefined) {
      decorations.append([{
        range: new monaco.Range(focusLine, 1, focusLine, 1),
        options: { isWholeLine: true, className: 'comparison-value-line' },
      }]);
      modified.revealLineInCenter(focusLine);
    }
    if (focusRange) {
      const range = new monaco.Range(focusRange.start, 1, focusRange.end, 1);
      modified.revealRangeInCenter(range);
      modified.setPosition({ lineNumber: focusRange.start, column: 1 });
    }
    return () => decorations.clear();
  }, [current.text, previous.text, focusLine, focusRange, highlightRanges]);

  return <div className="preview" ref={container} />;
}
