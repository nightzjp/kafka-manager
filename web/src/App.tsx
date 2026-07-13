import { useEffect, useState } from 'react';
import { LoginPage } from './features/auth/LoginPage';
import { api } from './lib/api';
import { Cluster } from './lib/types';
import { DashboardPage } from './pages/DashboardPage';
import { TopicsPage } from './pages/TopicsPage';
import { MessagesPage } from './pages/MessagesPage';
import { ConsumersPage } from './pages/ConsumersPage';
import { SettingsPage } from './pages/SettingsPage';
import { AuditPage } from './pages/AuditPage';
import './styles.css';

type Page='dashboard'|'topics'|'messages'|'consumers'|'settings'|'audit';
const nav:[Page,string,string][]=[['dashboard','态势','◫'],['topics','Topics','≋'],['messages','消息检索','⌁'],['consumers','消费组','◉'],['settings','集群配置','⌘'],['audit','操作记录','↗']];

export function App(){
 const [authenticated,setAuthenticated]=useState<boolean|null>(null);const[page,setPage]=useState<Page>('dashboard');const[clusters,setClusters]=useState<Cluster[]>([]);const[clusterId,setClusterId]=useState('');
 const loadClusters=()=>api.get<{items:Cluster[]}>('/api/v1/clusters').then(({items})=>{setClusters(items);setClusterId(current=>current||items[0]?.id||'')});
 useEffect(()=>{api.get('/api/v1/auth/me').then(()=>setAuthenticated(true)).catch(()=>setAuthenticated(false));const expire=()=>setAuthenticated(false);window.addEventListener('session-expired',expire);return()=>window.removeEventListener('session-expired',expire)},[]);
 useEffect(()=>{if(authenticated)loadClusters().catch(()=>{})},[authenticated]);
 if(authenticated===null)return <div className="boot"><span>K</span><small>INITIALIZING CONTROL PLANE</small></div>;
 if(!authenticated)return <LoginPage onSuccess={()=>setAuthenticated(true)}/>;
 const current=clusters.find(cluster=>cluster.id===clusterId);
 return <div className="app-shell">
  <aside className="rail"><div className="brand"><b>K</b><span>KAFKA<br/>MANAGER</span></div><nav>{nav.map(([id,label,icon])=><button key={id} className={page===id?'active':''} onClick={()=>setPage(id)}><i>{icon}</i><span>{label}</span></button>)}</nav><div className="rail-foot"><span className="pulse"/>CONTROL PLANE<br/><small>LOCAL / SECURE</small></div></aside>
  <section className="workspace"><header className="topbar"><div><span className="eyebrow">CURRENT TARGET</span><select value={clusterId} onChange={event=>setClusterId(event.target.value)}>{clusters.map(cluster=><option key={cluster.id} value={cluster.id}>{cluster.name} · {cluster.id}</option>)}</select></div><div className={`connection ${current?.online?'online':'offline'}`}><span/>{current?.online?'CONNECTED':'OFFLINE'}<small>{current?.latencyMs||0} ms</small></div><button className="logout" onClick={()=>api.post('/api/v1/auth/logout',{}).finally(()=>setAuthenticated(false))}>退出</button></header>
  <main className="content">{page==='dashboard'&&<DashboardPage clusters={clusters} refresh={loadClusters} openCluster={(id,target)=>{setClusterId(id);setPage(target)}}/>}{page==='topics'&&<TopicsPage clusterId={clusterId}/>} {page==='messages'&&<MessagesPage clusterId={clusterId}/>} {page==='consumers'&&<ConsumersPage clusterId={clusterId}/>} {page==='settings'&&<SettingsPage onSaved={loadClusters}/>} {page==='audit'&&<AuditPage/>}</main></section>
 </div>
}
