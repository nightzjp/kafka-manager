import { describe, expect, it } from 'vitest';
import { connectionView } from './connection-state';
import { Cluster } from '../lib/types';

const online: Cluster = {
  id: 'dev', name: '开发环境', status: 'online', online: true, sampledAt: 1, latencyMs: 12,
  brokers: 3, topics: 4, partitions: 8, consumerGroups: 2, underReplicated: 0, totalLag: 0, readOnly: false,
};

describe('connectionView', () => {
  it('treats a missing initial snapshot as loading instead of offline', () => {
    expect(connectionView(undefined, 'loading')).toEqual({ tone: 'loading', label: '正在获取状态' });
  });

  it('distinguishes management service failures from Kafka failures', () => {
    expect(connectionView(online, 'error')).toEqual({ tone: 'error', label: '管理服务不可用' });
    expect(connectionView({ ...online, status: 'offline', online: false }, 'ready')).toEqual({ tone: 'offline', label: 'Kafka 连接中断' });
  });

  it('shows backend loading and online snapshots accurately', () => {
    expect(connectionView({ ...online, status: 'loading', online: false }, 'ready')).toEqual({ tone: 'loading', label: '正在连接 Kafka' });
    expect(connectionView(online, 'ready')).toEqual({ tone: 'online', label: '在线 · 12ms' });
  });

  it('supports snapshots from an older backend without status', () => {
    expect(connectionView({ ...online, status: undefined }, 'ready')).toEqual({ tone: 'online', label: '在线 · 12ms' });
  });
});
