import { useEffect, useState } from 'react';
import { ConfirmDialog, ErrorNotice, Loading, PageHeader } from '../components/Common';
import { useFeedback } from '../components/Feedback';
import { api } from '../lib/api';
import { AppConfig, ConfigBackup, KafkaClusterConfig } from '../lib/types';

const blank = (): KafkaClusterConfig => ({ id: '', name: '', brokers: [''], readOnly: false, security: { protocol: 'PLAINTEXT' } });

export function SettingsPage({ onSaved }: { onSaved: () => void }) {
  const [cfg, setCfg] = useState<AppConfig>();
  const [backups, setBackups] = useState<ConfigBackup[]>([]);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [restoreTarget, setRestoreTarget] = useState<ConfigBackup>();
  const [restoring, setRestoring] = useState(false);
  const feedback = useFeedback();
  const load = () => {
    api.get<AppConfig>('/api/v1/config').then((value) => { setCfg(value); setError(''); }).catch((reason: Error) => setError(reason.message));
    api.get<{ items: ConfigBackup[] }>('/api/v1/config/backups').then((response) => setBackups(response.items)).catch(() => {});
  };
  useEffect(load, []);
  if (!cfg) return <><PageHeader code="05 / CONFIG" title="集群配置" description="配置写入 YAML，并在连接验证成功后热加载。" />{error ? <ErrorNotice message={error} /> : <Loading />}</>;

  const update = (index: number, patch: Partial<KafkaClusterConfig>) => setCfg({ ...cfg, clusters: cfg.clusters.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item) });
  async function save() {
    if (!cfg) return;
    setSaving(true); setError('');
    const clusterCount = cfg.clusters.length;
    try {
      await api.put('/api/v1/config', cfg);
      feedback.success('配置已保存并热加载', `${clusterCount} 个 Kafka 集群连接验证通过`);
      onSaved(); load();
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : '保存失败';
      setError(message); feedback.error('配置保存失败', message);
    } finally { setSaving(false); }
  }
  async function restore() {
    if (!restoreTarget) return;
    setRestoring(true); setError('');
    try {
      await api.post(`/api/v1/config/backups/${restoreTarget.id}`, {});
      feedback.success('配置已回滚', restoreTarget.id);
      setRestoreTarget(undefined); load(); onSaved();
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : '回滚失败';
      setError(message); feedback.error('配置回滚失败', message);
    } finally { setRestoring(false); }
  }

  return <>
    <PageHeader code="05 / CONFIG" title="集群配置" description="保存前会连接所有 Broker；失败时保留当前有效配置和客户端。" actions={<button className="button primary" disabled={saving} onClick={save}>{saving ? '验证连接中…' : '保存并热加载'}</button>} />
    {error && <ErrorNotice message={error} />}
    <div className="settings-grid">{cfg.clusters.map((cluster, index) => <article className="config-card" key={`${cluster.id}-${index}`}>
      <header><span>CLUSTER {String(index + 1).padStart(2, '0')}</span><button onClick={() => setCfg({ ...cfg, clusters: cfg.clusters.filter((_, itemIndex) => itemIndex !== index) })}>移除</button></header>
      <label>显示名称<input value={cluster.name} onChange={(event) => update(index, { name: event.target.value })} /></label>
      <label>唯一 ID<input value={cluster.id} onChange={(event) => update(index, { id: event.target.value })} /></label>
      <label>Broker 地址（每行一个）<textarea value={cluster.brokers.join('\n')} onChange={(event) => update(index, { brokers: event.target.value.split('\n').filter(Boolean) })} /></label>
      <label className="check-field config-readonly"><input type="checkbox" checked={Boolean(cluster.readOnly)} onChange={(event) => update(index, { readOnly: event.target.checked })} />只读模式（禁止所有 Kafka 写操作）</label>
      <label>连接协议<select value={cluster.security.protocol || 'PLAINTEXT'} onChange={(event) => update(index, { security: { ...cluster.security, protocol: event.target.value } })}><option>PLAINTEXT</option><option>SSL</option><option>SASL_PLAINTEXT</option><option>SASL_SSL</option></select></label>
      {cluster.security.protocol.startsWith('SASL') && <><label>认证机制<select value={cluster.security.mechanism || 'SCRAM-SHA-256'} onChange={(event) => update(index, { security: { ...cluster.security, mechanism: event.target.value } })}><option>PLAIN</option><option>SCRAM-SHA-256</option><option>SCRAM-SHA-512</option></select></label><label>用户名<input value={cluster.security.username || ''} onChange={(event) => update(index, { security: { ...cluster.security, username: event.target.value } })} /></label><label>密码<input type="password" placeholder="留空保持原密码" onChange={(event) => update(index, { security: { ...cluster.security, password: event.target.value } })} /></label></>}
    </article>)}<button className="add-cluster" onClick={() => setCfg({ ...cfg, clusters: [...cfg.clusters, blank()] })}><b>＋</b><span>新增 Kafka 集群</span><small>支持无认证、SASL 与 TLS</small></button></div>
    <section className="retention"><h3>本地数据保留策略</h3><label>审计保留天数<input type="number" min="1" value={cfg.audit.retentionDays} onChange={(event) => setCfg({ ...cfg, audit: { ...cfg.audit, retentionDays: Number(event.target.value) } })} /></label><label>审计单文件上限 (MB)<input type="number" min="1" value={cfg.audit.maxFileSizeMB} onChange={(event) => setCfg({ ...cfg, audit: { ...cfg.audit, maxFileSizeMB: Number(event.target.value) } })} /></label><label>配置备份保留天数<input type="number" min="1" value={cfg.audit.configBackupRetentionDays} onChange={(event) => setCfg({ ...cfg, audit: { ...cfg.audit, configBackupRetentionDays: Number(event.target.value) } })} /></label></section>
    {backups.length > 0 && <section className="backups"><h3>配置备份</h3>{backups.slice(0, 10).map((backup) => <div key={backup.id}><code>{backup.id}</code><span>{new Date(backup.createdAt).toLocaleString()} · {(backup.size / 1024).toFixed(1)} KB</span><button className="button ghost" onClick={() => { setError(''); setRestoreTarget(backup); }}>回滚</button></div>)}</section>}
    {restoreTarget && <ConfirmDialog title="回滚配置" description={<><b>将当前配置替换为备份 {restoreTarget.id}。</b><p>系统会重新验证 Kafka 连接并热加载；当前配置会先自动备份。</p></>} confirmLabel="确认回滚配置" pending={restoring} error={error} onClose={() => { setRestoreTarget(undefined); setError(''); }} onConfirm={restore} />}
  </>;
}
