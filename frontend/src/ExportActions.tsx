import { useState } from 'react';

export function ExportActions({ text, filename, loadBundle }: {
  text: string;
  filename: string;
  loadBundle?: () => Promise<Blob>;
}) {
  const [message, setMessage] = useState('');
  const [busy, setBusy] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setMessage('Copied to clipboard.');
    } catch {
      setMessage('Could not copy. Use Download to save the configuration.');
    }
  }

  async function download() {
    setBusy(true);
    setMessage('');
    try {
      const output = loadBundle ? await loadBundle() : new Blob([text], { type: 'text/yaml;charset=utf-8' });
      const url = URL.createObjectURL(output);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Could not prepare the export.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="export-actions" aria-label="Export active format">
      <button type="button" disabled={!text} onClick={copy}>{loadBundle ? 'Copy values' : 'Copy'}</button>
      <button type="button" disabled={!text || busy} onClick={download}>{loadBundle ? 'Download Helm bundle' : 'Download'}</button>
      <span role="status">{message}</span>
    </div>
  );
}
