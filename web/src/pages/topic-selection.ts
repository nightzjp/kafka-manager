import { Topic } from '../lib/types';

export function selectExactTopic(items: Topic[], name: string): Topic | undefined {
  return items.find((item) => item.Name === name);
}
