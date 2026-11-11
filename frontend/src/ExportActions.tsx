import { useState } from 'react';

export function ExportActions({ text, filename }: { text: string; filename: string }) {
  const [message, setMessage] = useState('');

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setMessage('Copied to clipboard.');
    } catch {
      setMessage('Could not copy. Use Download to save the YAML.');
    }
  }

  function download() {
    const url = URL.createObjectURL(new Blob([text], { type: 'text/yaml;charset=utf-8' }));
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  return (
    <div aria-label="Export active format">
      <button type="button" disabled={!text} onClick={copy}>Copy</button>
      <button type="button" disabled={!text} onClick={download}>Download</button>
      <span role="status">{message}</span>
    </div>
  );
}
