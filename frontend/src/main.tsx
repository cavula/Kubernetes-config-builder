import { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Builder } from './Builder';
import { kubernetesVersions, platform, selectConfiguration, tools } from './selection';
import './style.css';

function App() {
  const [configuration, setConfiguration] = useState(() =>
    selectConfiguration(platform.id, kubernetesVersions[0], tools[0].id),
  );

  return (
    <main>
      <h1>Kubernetes Config Builder</h1>
      <section aria-label="Configuration selection">
        <label>
          Platform
          <select value={platform.id} onChange={event => setConfiguration(
            selectConfiguration(event.target.value, configuration.kubernetesVersion, configuration.tool.id),
          )}>
            <option value={platform.id}>{platform.name}</option>
          </select>
        </label>
        <label>
          Kubernetes version
          <select value={configuration.kubernetesVersion} onChange={event => setConfiguration(
            selectConfiguration(platform.id, event.target.value, configuration.tool.id),
          )}>
            {kubernetesVersions.map(version => <option key={version}>{version}</option>)}
          </select>
        </label>
        <label>
          Tool
          <select value={configuration.tool.id} onChange={event => setConfiguration(
            selectConfiguration(platform.id, configuration.kubernetesVersion, event.target.value),
          )}>
            {tools.map(tool => <option key={tool.id} value={tool.id}>{tool.name}</option>)}
          </select>
        </label>
      </section>
      <p aria-live="polite">{configuration.tool.name} chart <strong>{configuration.tool.chartVersion}</strong></p>
      <Builder key={configuration.tool.id} tool={configuration.tool.id} chartVersion={configuration.tool.chartVersion} kubernetesVersion={configuration.kubernetesVersion} />
    </main>
  );
}

createRoot(document.getElementById('root')!).render(<App />);
