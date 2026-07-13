import { useCallback, useEffect, useState } from 'react';
import { ErrorNotice, Loading } from './components/Common';
import { Icon, IconName } from './components/Icon';
import { LoginPage } from './features/auth/LoginPage';
import { api } from './lib/api';
import { Cluster, DashboardPoint, Topic } from './lib/types';
import { AppRoute, buildRoutePath, Page, parseRoutePath } from './navigation/routes';
import { readSidebarCollapsed, writeSidebarCollapsed } from './navigation/sidebar-state';
import { AuditPage } from './pages/AuditPage';
import { ConsumersPage } from './pages/ConsumersPage';
import { DashboardPage } from './pages/DashboardPage';
import { MessagesPage } from './pages/MessagesPage';
import { SettingsPage } from './pages/SettingsPage';
import { TopicsPage } from './pages/TopicsPage';
import { TopicWorkspace } from './pages/TopicWorkspace';
import { selectExactTopic } from './pages/topic-selection';
import { ThemeMode, useTheme } from './theme/ThemeProvider';
import './styles.css';
import './polish.css';

const nav: { id: Page; label: string; icon: IconName }[] = [
  { id: 'dashboard', label: '态势总览', icon: 'dashboard' },
  { id: 'topics', label: 'Topics', icon: 'topic' },
  { id: 'messages', label: '消息检索', icon: 'message' },
  { id: 'consumers', label: '消费组', icon: 'consumer' },
  { id: 'settings', label: '集群配置', icon: 'settings' },
  { id: 'audit', label: '操作审计', icon: 'audit' },
];

export function App() {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null);
  const [route, setRoute] = useState<AppRoute>(() => parseRoutePath(window.location.pathname));
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [history, setHistory] = useState<Record<string, DashboardPoint[]>>({});
  const [selectedTopic, setSelectedTopic] = useState<{ clusterId: string; item: Topic }>();
  const [topicLoading, setTopicLoading] = useState(Boolean(route.topicName));
  const [topicError, setTopicError] = useState('');
  const [menuOpen, setMenuOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => readSidebarCollapsed(window.localStorage));
  const { mode, setMode } = useTheme();

  const navigate = useCallback((next: AppRoute, replace = false) => {
    const path = buildRoutePath(next);
    window.history[replace ? 'replaceState' : 'pushState']({}, '', path);
    setRoute(next);
    setMenuOpen(false);
  }, []);
  const loadClusters = useCallback(() => api.get<{ items: Cluster[]; history: Record<string, DashboardPoint[]> }>('/api/v1/dashboard').then(({ items, history: points }) => {
    setClusters(items);
    setHistory(points || {});
  }), []);
  const loadTopic = useCallback(async (id: string, name: string) => {
    const { items } = await api.get<{ items: Topic[] }>(`/api/v1/clusters/${id}/topics?search=${encodeURIComponent(name)}&pageSize=200`);
    const found = selectExactTopic(items, name);
    if (!found) throw new Error(`找不到 Topic：${name}`);
    return found;
  }, []);

  useEffect(() => {
    api.get('/api/v1/auth/me').then(() => setAuthenticated(true)).catch(() => setAuthenticated(false));
    const expire = () => setAuthenticated(false);
    const backOrForward = () => setRoute(parseRoutePath(window.location.pathname));
    addEventListener('session-expired', expire);
    addEventListener('popstate', backOrForward);
    return () => { removeEventListener('session-expired', expire); removeEventListener('popstate', backOrForward); };
  }, []);
  useEffect(() => { if (authenticated) loadClusters().catch(() => {}); }, [authenticated, loadClusters]);
  useEffect(() => { writeSidebarCollapsed(window.localStorage, sidebarCollapsed); }, [sidebarCollapsed]);

  const clusterId = route.clusterId && clusters.some((cluster) => cluster.id === route.clusterId) ? route.clusterId : clusters[0]?.id || route.clusterId || '';
  const refreshTopic = useCallback(async () => {
    if (!route.topicName || !clusterId) return;
    const item = await loadTopic(clusterId, route.topicName);
    setSelectedTopic({ clusterId, item });
  }, [clusterId, loadTopic, route.topicName]);
  useEffect(() => {
    if (clusters.length && clusterId && route.clusterId !== clusterId) navigate({ ...route, clusterId }, true);
  }, [clusterId, clusters.length, navigate, route]);
  useEffect(() => {
    const label = route.topicName || nav.find((item) => item.id === route.page)?.label || 'Kafka Manager';
    document.title = `${label} · Kafka Manager`;
  }, [route.page, route.topicName]);

  useEffect(() => {
    if (authenticated !== true || clusters.length === 0) return;
    if (!route.topicName || !clusterId) {
      setSelectedTopic(undefined); setTopicError(''); setTopicLoading(false); return;
    }
    if (selectedTopic?.clusterId === clusterId && selectedTopic.item.Name === route.topicName) return;
    let active = true;
    setTopicLoading(true); setTopicError('');
    loadTopic(clusterId, route.topicName)
      .then((found) => {
        if (!active) return;
        setSelectedTopic({ clusterId, item: found });
      })
      .catch((reason: Error) => { if (active) setTopicError(reason.message); })
      .finally(() => { if (active) setTopicLoading(false); });
    return () => { active = false; };
  }, [authenticated, clusterId, clusters.length, loadTopic, route.topicName, selectedTopic]);

  if (authenticated === null) return <div className="boot"><div className="logo-mark">K</div><span>正在启动 Kafka Manager</span></div>;
  if (!authenticated) return <LoginPage onSuccess={() => setAuthenticated(true)} />;

  const current = clusters.find((item) => item.id === clusterId);
  const topic = route.topicName && selectedTopic?.clusterId === clusterId && selectedTopic.item.Name === route.topicName ? selectedTopic.item : undefined;
  const changePage = (page: Page) => navigate({ page, clusterId });
  const openTopic = (item: Topic, tab: 'overview' | 'messages' = 'overview') => {
    setSelectedTopic({ clusterId, item });
    navigate({ page: 'topics', clusterId, topicName: item.Name, topicTab: tab });
  };

  return <div className="app-shell">
    <aside className={`sidebar ${sidebarCollapsed ? 'collapsed' : ''} ${menuOpen ? 'open' : ''}`}>
      <div className="brand"><div className="logo-mark">K</div><div className="brand-copy"><b>Kafka Manager</b><span>Developer Console</span></div></div>
      <nav id="primary-navigation">{nav.map((item) => <button key={item.id} data-label={item.label} aria-label={sidebarCollapsed ? item.label : undefined} className={route.page === item.id && !route.topicName ? 'active' : ''} onClick={() => changePage(item.id)}><Icon name={item.icon} /><span>{item.label}</span></button>)}</nav>
      <div className="sidebar-utilities">
        <div className="sidebar-foot"><span className={`health-dot ${clusters.some((cluster) => !cluster.online) ? 'warn' : ''}`} /><span className="sidebar-foot-copy">{clusters.filter((cluster) => cluster.online).length}/{clusters.length} 集群在线</span></div>
        <button className="sidebar-toggle" aria-label={sidebarCollapsed ? '展开侧栏' : '收起侧栏'} aria-controls="primary-navigation" aria-expanded={!sidebarCollapsed} title={sidebarCollapsed ? '展开侧栏' : '收起侧栏'} onClick={() => setSidebarCollapsed((value) => !value)}><Icon name={sidebarCollapsed ? 'arrow' : 'back'} /><span className="sidebar-toggle-copy">{sidebarCollapsed ? '展开侧栏' : '收起侧栏'}</span></button>
      </div>
    </aside>
    {menuOpen && <button className="nav-scrim" aria-label="关闭导航" onClick={() => setMenuOpen(false)} />}
    <section className={`workspace ${sidebarCollapsed ? 'sidebar-collapsed' : ''}`}>
      <header className="topbar"><button className="icon-button mobile-menu" aria-label="打开导航" onClick={() => setMenuOpen(true)}><Icon name="menu" /></button><div className="cluster-picker"><span>当前集群</span><select value={clusterId} onChange={(event) => navigate({ page: route.page, clusterId: event.target.value })}>{clusters.map((cluster) => <option key={cluster.id} value={cluster.id}>{cluster.name}</option>)}</select>{current?.readOnly && <span className="readonly-badge">只读</span>}</div><div className="top-actions"><span className={`connection ${current?.online ? 'online' : 'offline'}`}><i />{current?.online ? `在线 · ${current.latencyMs}ms` : '连接中断'}</span><ThemeSwitch mode={mode} setMode={setMode} /><button className="icon-button" title="退出登录" onClick={() => api.post('/api/v1/auth/logout', {}).finally(() => setAuthenticated(false))}><Icon name="logout" /></button></div></header>
      <main className="content">{route.topicName ? topicLoading ? <Loading /> : topicError ? <TopicRouteError message={topicError} back={() => navigate({ page: 'topics', clusterId }, true)} /> : topic ? <TopicWorkspace clusterId={clusterId} topic={topic} tab={route.topicTab || 'overview'} setTab={(topicTab) => navigate({ ...route, topicTab })} onBack={() => navigate({ page: 'topics', clusterId })} onRefresh={refreshTopic} readOnly={Boolean(current?.readOnly)} /> : null : <>
        {route.page === 'dashboard' && <DashboardPage clusters={clusters} history={history} refresh={loadClusters} openCluster={(id, page) => navigate({ page, clusterId: id })} />}
        {route.page === 'topics' && <TopicsPage clusterId={clusterId} onOpen={openTopic} readOnly={Boolean(current?.readOnly)} />}
        {route.page === 'messages' && <MessagesPage clusterId={clusterId} readOnly={Boolean(current?.readOnly)} />}
        {route.page === 'consumers' && <ConsumersPage clusterId={clusterId} readOnly={Boolean(current?.readOnly)} />}
        {route.page === 'settings' && <SettingsPage onSaved={loadClusters} />}
        {route.page === 'audit' && <AuditPage />}
      </>}</main>
    </section>
  </div>;
}

function TopicRouteError({ message, back }: { message: string; back: () => void }) {
  return <div className="route-error"><ErrorNotice message={message} /><button className="button ghost" onClick={back}><Icon name="back" />返回 Topic 列表</button></div>;
}

function ThemeSwitch({ mode, setMode }: { mode: ThemeMode; setMode: (mode: ThemeMode) => void }) {
  const next: ThemeMode = mode === 'system' ? 'light' : mode === 'light' ? 'dark' : 'system';
  return <button className="theme-switch" title={`主题：${mode}`} onClick={() => setMode(next)}><Icon name={mode === 'light' ? 'sun' : mode === 'dark' ? 'moon' : 'system'} /><span>{mode === 'system' ? '自动' : mode === 'light' ? '日间' : '夜间'}</span></button>;
}
