import { describe, expect, it } from 'vitest';
import { Topic } from '../lib/types';
import { selectExactTopic } from './topic-selection';

function topic(name: string): Topic {
  return { Name: name, Internal: false, PartitionCount: 1, ReplicationFactor: 1, UnderReplicated: 0, Partitions: [] };
}

describe('exact Topic selection', () => {
  it('does not confuse a Topic with a partial-name search result', () => {
    const exact = topic('orders');
    expect(selectExactTopic([topic('orders.retry'), exact, topic('old-orders')], 'orders')).toBe(exact);
  });

  it('returns undefined when the exact Topic is absent', () => {
    expect(selectExactTopic([topic('orders.retry')], 'orders')).toBeUndefined();
  });
});
