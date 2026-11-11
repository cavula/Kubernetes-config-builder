export const platform = { id: 'k3s', name: 'K3s' } as const;
export const kubernetesVersions = ['1.37', '1.36', '1.35'] as const;
export const tools = [
  { id: 'prometheus', name: 'Prometheus', chartVersion: '29.45.1' },
  { id: 'argo-cd', name: 'Argo CD', chartVersion: '10.17.1' },
] as const;

export function selectConfiguration(platformId: string, versionId: string, toolId: string) {
  const kubernetesVersion = kubernetesVersions.find(version => version === versionId);
  const tool = tools.find(entry => entry.id === toolId);

  if (platformId !== platform.id || !kubernetesVersion || !tool) {
    throw new RangeError('Unsupported platform, Kubernetes version, or tool.');
  }

  return { platform, kubernetesVersion, tool };
}
