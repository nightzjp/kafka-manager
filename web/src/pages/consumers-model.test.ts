import { describe, expect, it } from 'vitest';
import { ConsumerGroup } from '../lib/types';
import { filterPartitions, sortConsumerGroups, summarizeConsumerGroups } from './consumers-model';

const groups: ConsumerGroup[] = [
  {
    Name: 'checkout-workers', State: 'Stable', Protocol: 'consumer', MemberCount: 2, TotalLag: 0,
    Partitions: [{ topic: 'checkout.events', partition: 0, currentOffset: 100, endOffset: 100, lag: 0 }],
  },
  {
    Name: 'analytics-workers', State: 'Empty', Protocol: 'consumer', MemberCount: 0, TotalLag: 27,
    Partitions: [
      { topic: 'orders.v1', partition: 0, currentOffset: 90, endOffset: 100, lag: 10 },
      { topic: 'orders.v1', partition: 1, currentOffset: 83, endOffset: 100, lag: 17 },
    ],
  },
];

describe('consumer group diagnostics model', () => {
  it('summarizes lag and affected groups', () => {
    expect(summarizeConsumerGroups(groups)).toEqual({ groups: 2, laggingGroups: 1, totalLag: 27, laggingPartitions: 2 });
  });

  it('sorts groups by lag without mutating the API response', () => {
    const sorted = sortConsumerGroups(groups, 'lag-desc');
    expect(sorted.map((group) => group.Name)).toEqual(['analytics-workers', 'checkout-workers']);
    expect(groups[0].Name).toBe('checkout-workers');
  });

  it('filters partitions by topic, partition and lag state', () => {
    expect(filterPartitions(groups[1].Partitions, 'orders.v1 / p1', false).map((item) => item.partition)).toEqual([1]);
    expect(filterPartitions(groups[0].Partitions, '', true)).toEqual([]);
  });
});
