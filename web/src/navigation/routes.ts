export type Page = 'dashboard' | 'topics' | 'messages' | 'consumers' | 'settings' | 'audit';
export type TopicTab = 'overview' | 'messages' | 'partitions' | 'config';

export type AppRoute = {
  page: Page;
  clusterId?: string;
  topicName?: string;
  topicTab?: TopicTab;
};

const pages = new Set<Page>(['dashboard', 'topics', 'messages', 'consumers', 'settings', 'audit']);
const topicTabs = new Set<TopicTab>(['overview', 'messages', 'partitions', 'config']);

function safeDecode(value: string): string {
  try { return decodeURIComponent(value); }
  catch { return value; }
}

export function parseRoutePath(pathname: string): AppRoute {
  const segments = pathname.split('/').filter(Boolean);
  if (segments[0] !== 'clusters' || !segments[1]) return { page: 'dashboard' };
  const clusterId = safeDecode(segments[1]);
  const requestedPage = segments[2] as Page | undefined;
  const page: Page = requestedPage && pages.has(requestedPage) ? requestedPage : 'dashboard';
  if (page !== 'topics' || !segments[3]) return { page, clusterId };
  const requestedTab = segments[4] as TopicTab | undefined;
  return {
    page,
    clusterId,
    topicName: safeDecode(segments[3]),
    topicTab: requestedTab && topicTabs.has(requestedTab) ? requestedTab : 'overview',
  };
}

export function buildRoutePath(route: AppRoute): string {
  if (!route.clusterId) return '/';
  const base = `/clusters/${encodeURIComponent(route.clusterId)}/${route.page}`;
  if (route.page !== 'topics' || !route.topicName) return base;
  return `${base}/${encodeURIComponent(route.topicName)}/${route.topicTab || 'overview'}`;
}
