import { useEffect, useRef } from 'react';
import * as monaco from 'monaco-editor';
import EditorWorker from 'monaco-editor/editor/editor.worker.js?worker';

self.MonacoEnvironment = { getWorker: () => new EditorWorker() };

type Content = { text: string; notes: Map<number, string>; folds?: LineRange[] };
export type LineRange = { start: number; end: number };

function updateModel(model: monaco.editor.ITextModel, text: string) {
  const current = model.getValue();
  if (current === text) return;
  if (!current || !text) {
    model.setValue(text);
    return;
  }
  let start = 0;
  let oldEnd = current.length;
  let newEnd = text.length;
  while (start < oldEnd && start < newEnd && current[start] === text[start]) start++;
  while (oldEnd > start && newEnd > start && current[oldEnd - 1] === text[newEnd - 1]) {
    oldEnd--;
    newEnd--;
  }
  if (Math.max(oldEnd - start, newEnd - start) > 64 * 1024) {
    model.setValue(text);
    return;
  }
  model.applyEdits([{
    range: monaco.Range.fromPositions(model.getPositionAt(start), model.getPositionAt(oldEnd)),
    text: text.slice(start, newEnd),
  }]);
}

export function Preview({ current, previous, focusLine, focusRange, highlightRanges, explainField }: {
  current: Content;
  previous: Content;
  focusLine?: number;
  focusRange?: LineRange;
  highlightRanges?: LineRange[];
  explainField?: (text: string, line: number, signal: AbortSignal) => Promise<string>;
}) {
  const container = useRef<HTMLDivElement>(null);
  const models = useRef<{ original: monaco.editor.ITextModel; modified: monaco.editor.ITextModel }>(null);
  const editorRef = useRef<monaco.editor.IStandaloneDiffEditor>(null);
  const foldingRanges = useRef(current.folds);
  const hoverNotes = useRef({ current: current.notes, previous: previous.notes, explainField });

  useEffect(() => {
    const original = monaco.editor.createModel('', 'yaml');
    const modified = monaco.editor.createModel('', 'yaml');
    const editor = monaco.editor.createDiffEditor(container.current!, {
      automaticLayout: true,
      readOnly: true,
      renderSideBySide: false,
      minimap: { enabled: false },
      diffWordWrap: 'on',
      stickyScroll: { enabled: true },
      accessibilityVerbose: true,
    });
    const folding = foldingRanges.current && monaco.languages.registerFoldingRangeProvider('yaml', {
      provideFoldingRanges(model) {
        return model === modified ? foldingRanges.current : undefined;
      },
    });
    const hover = monaco.languages.registerHoverProvider('yaml', {
      async provideHover(model, position, token) {
        const { current, previous, explainField } = hoverNotes.current;
        const notes = model === modified ? current : model === original ? previous : undefined;
        if (!notes) return;
        const line = position.lineNumber;
        let note = notes.get(line);
        if (!note && explainField && model.getLineContent(line).trim()) {
          const version = model.getVersionId();
          const controller = new AbortController();
          const cancellation = token.onCancellationRequested(() => controller.abort());
          try {
            note = await explainField(model.getValue(), line, controller.signal);
          } catch {
            return;
          } finally {
            cancellation.dispose();
          }
          if (token.isCancellationRequested || model.isDisposed() || model.getVersionId() !== version) return;
          if (note) notes.set(line, note);
        }
        if (note) return { contents: [{ value: note.replace(/[\\`*_{}\[\]()<>#+.!|]/g, '\\$&') }] };
      },
    });
    editor.setModel({ original, modified });
    editor.getOriginalEditor().updateOptions({ lineNumbers: 'off' });
    editorRef.current = editor;
    models.current = { original, modified };
    return () => {
      hover.dispose();
      folding?.dispose();
      editor.dispose();
      original.dispose();
      modified.dispose();
      models.current = null;
      editorRef.current = null;
    };
  }, []);

  useEffect(() => {
    hoverNotes.current = { current: current.notes, previous: previous.notes, explainField };
    foldingRanges.current = current.folds;
    const { original, modified } = models.current!;
    updateModel(original, previous.text);
    updateModel(modified, current.text);
  }, [current, previous, explainField]);

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

  useEffect(() => {
    if (!current.folds?.length) return;
    const modified = editorRef.current!.getModifiedEditor();
    modified.updateOptions({ folding: true, showFoldingControls: 'always' });
    void modified.getAction('editor.fold')?.run({
      selectionLines: current.folds.map(({ start }) => start - 1),
    });
  }, [current.text, current.folds]);

  return <div className="preview" ref={container} />;
}
