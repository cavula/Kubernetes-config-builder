import { useState } from 'react';

export function ExportActions({ text, filename, label = "Download", copyLabel = "Copy", load }: { text: string; filename: string; label?: string; copyLabel?: string; load?: () => Promise<string> }) {
  const [message, setMessage] = useState('');
  const [busy, setBusy] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(load ? await load() : text);
      setMessage('Copied to clipboard.');
    } catch {
      setMessage('Could not copy. Use Download to save the YAML.');
    }
  }

  async function download() {
    setBusy(true);
    try {
      const output = load ? await load() : text;
      const url = URL.createObjectURL(new Blob([output], { type: 'text/yaml;charset=utf-8' }));
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
    <div aria-label="Export active format">
      <button type="button" disabled={!text} onClick={copy}>{copyLabel}</button>
      <button type="button" disabled={!text || busy} onClick={download}>{label}</button>
      <span role="status">{message}</span>
    </div>
  );
}
