import { describe, expect, it } from 'vitest';
import { buildRoutePath, parseRoutePath } from './routes';

describe('application routes', () => {
  it('round-trips a cluster page', () => {
    const route = { page: 'consumers' as const, clusterId: 'test-cluster' };
    expect(parseRoutePath(buildRoutePath(route))).toEqual(route);
  });

  it('round-trips Topic names and cluster IDs with reserved characters', () => {
    const route = { page: 'topics' as const, clusterId: '测试 / 内网', topicName: 'orders/中国.v1', topicTab: 'messages' as const };
    const path = buildRoutePath(route);
    expect(path).toBe('/clusters/%E6%B5%8B%E8%AF%95%20%2F%20%E5%86%85%E7%BD%91/topics/orders%2F%E4%B8%AD%E5%9B%BD.v1/messages');
    expect(parseRoutePath(path)).toEqual(route);
  });

  it('uses safe defaults for unknown paths and Topic tabs', () => {
    expect(parseRoutePath('/something/unknown')).toEqual({ page: 'dashboard' });
    expect(parseRoutePath('/clusters/dev/topics/orders/not-a-tab')).toEqual({ page: 'topics', clusterId: 'dev', topicName: 'orders', topicTab: 'overview' });
  });

  it('builds the root path before a cluster is known', () => {
    expect(buildRoutePath({ page: 'dashboard' })).toBe('/');
  });
});
