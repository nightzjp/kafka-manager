import { Cluster } from '../lib/types';

export type DashboardLoadState = 'idle' | 'loading' | 'ready' | 'error';
export type ConnectionView = {
  tone: 'loading' | 'online' | 'offline' | 'error';
  label: string;
};

export function connectionView(cluster: Cluster | undefined, loadState: DashboardLoadState): ConnectionView {
  if (loadState === 'error') return { tone: 'error', label: '管理服务不可用' };
  if (!cluster) return { tone: 'loading', label: '正在获取状态' };
  const status = cluster.status || (cluster.online ? 'online' : 'offline');
  if (status === 'loading') return { tone: 'loading', label: '正在连接 Kafka' };
  if (status === 'offline') return { tone: 'offline', label: 'Kafka 连接中断' };
  return { tone: 'online', label: `在线 · ${cluster.latencyMs}ms` };
}
