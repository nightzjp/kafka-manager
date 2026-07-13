import { ConsumerGroup, PartitionLag } from '../lib/types';

export type ConsumerSort = 'lag-desc' | 'name-asc' | 'members-desc';

export function summarizeConsumerGroups(groups: ConsumerGroup[]) {
  return groups.reduce((summary, group) => ({
    groups: summary.groups + 1,
    laggingGroups: summary.laggingGroups + (group.TotalLag > 0 ? 1 : 0),
    totalLag: summary.totalLag + group.TotalLag,
    laggingPartitions: summary.laggingPartitions + group.Partitions.filter((partition) => partition.lag > 0).length,
  }), { groups: 0, laggingGroups: 0, totalLag: 0, laggingPartitions: 0 });
}

export function sortConsumerGroups(groups: ConsumerGroup[], sort: ConsumerSort) {
  return [...groups].sort((left, right) => {
    if (sort === 'name-asc') return left.Name.localeCompare(right.Name);
    if (sort === 'members-desc') return right.MemberCount - left.MemberCount || right.TotalLag - left.TotalLag;
    return right.TotalLag - left.TotalLag || left.Name.localeCompare(right.Name);
  });
}

export function filterPartitions(partitions: PartitionLag[], search: string, lagOnly: boolean) {
  const term = search.trim().toLowerCase();
  return [...partitions]
    .filter((partition) => !lagOnly || partition.lag > 0)
    .filter((partition) => !term || `${partition.topic} / p${partition.partition}`.toLowerCase().includes(term))
    .sort((left, right) => right.lag - left.lag || left.topic.localeCompare(right.topic) || left.partition - right.partition);
}
