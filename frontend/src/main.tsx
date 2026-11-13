import { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { presets, type Preset } from './builder-values';
import { UsageBuilder } from './UsageBuilder';
import { Builder } from './Builder';
import { kubernetesVersions, platform, selectConfiguration, tools } from './selection';
import './style.css';

function App() {
  const [configuration, setConfiguration] = useState(() =>
    selectConfiguration(platform.id, kubernetesVersions[0], 'argo-cd'),
  );

  const [mode, setMode] = useState<'use' | 'install'>('use');
  const [presetsByTool, setPresetsByTool] = useState<Record<string, Preset>>({});
  const activePreset = presetsByTool[configuration.tool.id] ?? 'minimal';

  return (
    <main>
      <h1>Kubernetes Config Builder</h1>
      <nav className="workflow-options" aria-label="Tool workflow">
        <button type="button" aria-pressed={mode === 'use'} onClick={() => setMode('use')}>Use the tool</button>
        <button type="button" aria-pressed={mode === 'install'} onClick={() => setMode('install')}>Install the tool</button>
      </nav>
      <section aria-labelledby="choose-preset" className="step configuration-selection">
        <h2 id="choose-preset">{mode === 'install' ? '1. Choose a preset' : 'Choose a tool'}</h2>
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
          <small>{mode === 'install' ? 'Chart to configure.' : 'Tool to generate usage YAML for.'}</small>
        </label>
        {mode === 'install' && <label>
          Preset
          <select value={activePreset} onChange={event => setPresetsByTool({ ...presetsByTool, [configuration.tool.id]: event.target.value as Preset })}>
            {Object.entries(presets).filter(([id]) => configuration.tool.id !== 'argo-cd' || id !== 'recommended')
              .map(([id, label]) => <option key={id} value={id}>{label}</option>)}
          </select>
          <small>{configuration.tool.id === 'prometheus'
            ? 'Start with the server alone or add monitoring components.'
            : activePreset === 'minimal' ? 'Core components only. Dex, notifications, and ApplicationSet are off.'
            : 'Includes Dex, notifications, and ApplicationSet.'}</small>
        </label>}
        {mode === 'install' && configuration.tool.id === 'prometheus' && activePreset === 'recommended' && (
          <small className="preset-description">Includes Alertmanager, kube-state-metrics, and node exporter. Enable Pushgateway for short-lived service-level batch jobs.</small>
        )}
      </section>
      {mode === 'install' ? <Builder key={`${configuration.tool.id}/${activePreset}`} preset={activePreset} tool={configuration.tool.id} chartVersion={configuration.tool.chartVersion} kubernetesVersion={configuration.kubernetesVersion} />
        : <UsageBuilder key={configuration.tool.id} tool={configuration.tool.id} kubernetesVersion={configuration.kubernetesVersion} />}
    </main>
  );
}

createRoot(document.getElementById('root')!).render(<App />);
