import { useEffect, useRef } from 'react';
import * as monaco from 'monaco-editor';
import EditorWorker from 'monaco-editor/editor/editor.worker.js?worker';

self.MonacoEnvironment = { getWorker: () => new EditorWorker() };

type Content = { text: string; notes: Map<number, string> };

export function Preview({ current, previous }: { current: Content; previous: Content }) {
  const container = useRef<HTMLDivElement>(null);
  const models = useRef<{ original: monaco.editor.ITextModel; modified: monaco.editor.ITextModel }>(null);
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
    models.current = { original, modified };
    return () => {
      hover.dispose();
      editor.dispose();
      original.dispose();
      modified.dispose();
      models.current = null;
    };
  }, []);

  useEffect(() => {
    currentNotes.current = current.notes;
    const { original, modified } = models.current!;
    if (original.getValue() !== previous.text) original.setValue(previous.text);
    if (modified.getValue() !== current.text) modified.setValue(current.text);
  }, [current, previous]);

  return <div className="preview" ref={container} />;
}
