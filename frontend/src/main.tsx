import { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { presets, type Preset } from './builder-values';
import { Builder } from './Builder';
import { kubernetesVersions, platform, selectConfiguration, tools } from './selection';
import './style.css';

function App() {
  const [configuration, setConfiguration] = useState(() =>
    selectConfiguration(platform.id, kubernetesVersions[0], tools[0].id),
  );

  const [presetsByTool, setPresetsByTool] = useState<Record<string, Preset>>({});
  const activePreset = presetsByTool[configuration.tool.id] ?? 'minimal';

  return (
    <main>
      <h1>Kubernetes Config Builder</h1>
      <section aria-labelledby="choose-preset" className="step configuration-selection">
        <h2 id="choose-preset">1. Choose a preset</h2>
        <label>
          Platform
          <select value={platform.id} onChange={event => setConfiguration(
            selectConfiguration(event.target.value, configuration.kubernetesVersion, configuration.tool.id),
          )}>
            <option value={platform.id}>{platform.name}</option>
          </select>
          <small>Target cluster platform.</small>
        </label>
        <label>
          Kubernetes version
          <select value={configuration.kubernetesVersion} onChange={event => setConfiguration(
            selectConfiguration(platform.id, event.target.value, configuration.tool.id),
          )}>
            {kubernetesVersions.map(version => <option key={version}>{version}</option>)}
          </select>
          <small>Version running on your cluster.</small>
        </label>
        <label>
          Tool
          <select value={configuration.tool.id} onChange={event => setConfiguration(
            selectConfiguration(platform.id, configuration.kubernetesVersion, event.target.value),
          )}>
            {tools.map(tool => <option key={tool.id} value={tool.id}>{tool.name}</option>)}
          </select>
          <small>Chart to configure.</small>
        </label>
        <label>
          Preset
          <select value={activePreset} onChange={event => setPresetsByTool({ ...presetsByTool, [configuration.tool.id]: event.target.value as Preset })}>
            {Object.entries(presets)
              .map(([id, label]) => <option key={id} value={id}>{label}</option>)}
          </select>
          <small>{configuration.tool.id === 'prometheus'
            ? 'Start with the server alone or add monitoring components.'
            : activePreset === 'minimal' ? 'Core components only. Dex, notifications, and ApplicationSet are off.'
            : activePreset === 'recommended' ? 'Includes ApplicationSet. Dex and notifications are off.'
            : 'Includes Dex, notifications, and ApplicationSet.'}</small>
        </label>
        {configuration.tool.id === 'prometheus' && activePreset === 'recommended' && (
          <small className="preset-description">Includes Alertmanager, kube-state-metrics, and node exporter. Enable Pushgateway for short-lived service-level batch jobs.</small>
        )}
      </section>
      <Builder key={`${configuration.tool.id}/${activePreset}`} preset={activePreset} tool={configuration.tool.id} chartVersion={configuration.tool.chartVersion} kubernetesVersion={configuration.kubernetesVersion} />
    </main>
  );
}

createRoot(document.getElementById('root')!).render(<App />);
