import { Cluster, DashboardPoint } from '../lib/types';
import { ErrorNotice, Loading, PageHeader, Status } from '../components/Common';
import { Icon } from '../components/Icon';
import { DashboardLoadState } from '../navigation/connection-state';

type Props = {
  clusters: Cluster[];
  history: Record<string, DashboardPoint[]>;
  loadState: DashboardLoadState;
  refresh: () => void;
  openCluster: (id: string, page: 'topics' | 'consumers') => void;
};

export function DashboardPage({ clusters, history, loadState, refresh, openCluster }: Props) {
  const header = <PageHeader code="OVERVIEW" title="Kafka 态势总览" description="后台持续采样 Kafka，页面读取内存快照，不会随浏览器数量增加 Kafka 压力。" actions={<button className="button ghost" onClick={refresh}><Icon name="refresh" />刷新显示</button>} />;
  if (loadState !== 'ready' && clusters.length === 0) {
    return <>{header}{loadState === 'error' ? <ErrorNotice message="暂时无法读取后台监控快照，请检查 Kafka Manager 服务状态后重试。" /> : <Loading />}</>;
  }
  const totals = clusters.reduce((current, cluster) => ({
    brokers: current.brokers + cluster.brokers,
    topics: current.topics + cluster.topics,
    partitions: current.partitions + cluster.partitions,
    groups: current.groups + cluster.consumerGroups,
    lag: current.lag + cluster.totalLag,
    isr: current.isr + cluster.underReplicated,
  }), { brokers: 0, topics: 0, partitions: 0, groups: 0, lag: 0, isr: 0 });
  const offline = clusters.filter((cluster) => clusterStatus(cluster) === 'offline').length;
  const loading = clusters.filter((cluster) => clusterStatus(cluster) === 'loading').length;
  const lagUnavailable = clusters.filter((cluster) => clusterStatus(cluster) === 'online' && cluster.lagAvailable === false).length;

  return <>
    {header}
    <section className="alert-grid">
      <Alert label="离线集群" value={offline} tone={offline ? 'bad' : loading ? 'warn' : 'good'} detail={offline ? '需要立即检查连接' : loading ? `${loading} 个集群正在首次采样` : '所有集群连接正常'} />
      <Alert label="ISR 异常" value={totals.isr} tone={totals.isr ? 'bad' : 'good'} detail={totals.isr ? '存在副本不同步' : '副本同步正常'} />
      <Alert label="消费积压" value={totals.lag} tone={lagUnavailable || totals.lag ? 'warn' : 'good'} detail={lagUnavailable ? `${lagUnavailable} 个集群的 Lag 暂不可用` : totals.lag ? '检查高 Lag 消费组' : '当前没有积压'} />
    </section>
    <section className="summary-grid dashboard-summary">
      <Metric label="Brokers" value={totals.brokers} />
      <Metric label="Topics" value={totals.topics} />
      <Metric label="Partitions" value={totals.partitions} />
      <Metric label="Consumer Groups" value={totals.groups} />
    </section>
    <div className="section-heading"><div><h2>集群</h2><p>连接、规模和消费趋势</p></div><span>{clusters.length} 个已配置集群</span></div>
    <div className="cluster-grid">{clusters.map((cluster) => <ClusterCard key={cluster.id} cluster={cluster} history={history[cluster.id] || []} openCluster={openCluster} />)}</div>
  </>;
}

function ClusterCard({ cluster, history, openCluster }: { cluster: Cluster; history: DashboardPoint[]; openCluster: Props['openCluster'] }) {
  const state = clusterStatus(cluster);
  return <article className={`cluster-card ${state === 'offline' ? 'is-offline' : state === 'loading' ? 'is-loading' : ''}`}>
    <header><div><span className="cluster-avatar">{cluster.name.slice(0, 1).toUpperCase()}</span><div><h2>{cluster.name}</h2><code>{cluster.id}</code></div></div><Status tone={state === 'online' ? 'good' : state === 'offline' ? 'bad' : 'neutral'}>{state === 'online' ? `在线 · ${cluster.latencyMs}ms` : state === 'offline' ? '离线' : '首次采样中'}</Status></header>
    {state === 'online' ? <>
      <div className="cluster-stats"><Metric label="Topics" value={cluster.topics} /><Metric label="Groups" value={cluster.consumerGroups} /><Metric label="Total Lag" value={cluster.lagAvailable === false ? '不可用' : cluster.totalLag} warning={cluster.lagAvailable === false || cluster.totalLag > 0} /><Metric label="ISR 异常" value={cluster.underReplicated} warning={cluster.underReplicated > 0} /></div>
      <Sparkline points={history} />
      <footer><button onClick={() => openCluster(cluster.id, 'topics')}>查看 Topics <Icon name="arrow" /></button><button onClick={() => openCluster(cluster.id, 'consumers')}>消费组 <Icon name="arrow" /></button></footer>
    </> : state === 'loading' ? <div className="offline-message pending"><Icon name="refresh" size={24} /><div><b>正在获取 Kafka 状态</b><p>后台采样器正在建立连接并生成首份快照。</p><span>页面刷新不会重复触发 Kafka 查询。</span></div></div> : <div className="offline-message"><Icon name="warning" size={24} /><div><b>无法连接 Kafka</b><p>{cluster.error}</p><span>后台将自动退避并重试，请检查 Broker 地址、网络和认证配置。</span></div></div>}
  </article>;
}

function clusterStatus(cluster: Cluster): 'loading' | 'online' | 'offline' {
  return cluster.status || (cluster.online ? 'online' : 'offline');
}

function Alert({ label, value, tone, detail }: { label: string; value: number; tone: string; detail: string }) {
  return <article className={`alert-card ${tone}`}><div><span>{label}</span><strong>{value.toLocaleString()}</strong></div><p>{detail}</p></article>;
}

function Metric({ label, value, warning = false }: { label: string; value: number | string; warning?: boolean }) {
  return <div className={`metric ${warning ? 'warn' : ''}`}><span>{label}</span><strong>{value.toLocaleString()}</strong></div>;
}

function Sparkline({ points }: { points: DashboardPoint[] }) {
  const values = points.map((point) => point.totalLag);
  const maximum = Math.max(1, ...values);
  const width = 320;
  const height = 58;
  const line = values.map((value, index) => `${values.length < 2 ? 0 : index * width / (values.length - 1)},${height - value / maximum * (height - 10) - 5}`).join(' ');
  return <div className="trend"><div><span>Lag 趋势</span><b>{values.at(-1)?.toLocaleString() || 0}</b></div><svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" aria-label="Lag 趋势"><polyline points={line} /></svg></div>;
}
