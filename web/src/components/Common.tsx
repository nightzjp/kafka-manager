import { ReactNode } from 'react';
export function PageHeader({code,title,description,actions}:{code:string;title:string;description:string;actions?:ReactNode}){return <header className="page-header"><div><span className="section-code">{code}</span><h1>{title}</h1><p>{description}</p></div><div className="page-actions">{actions}</div></header>}
export function Empty({title,detail}:{title:string;detail:string}){return <div className="empty"><span>∅</span><h3>{title}</h3><p>{detail}</p></div>}
export function ErrorNotice({message}:{message:string}){return <div className="error-notice"><b>REQUEST FAILED</b><span>{message}</span></div>}
export function Loading(){return <div className="loading"><i/><i/><i/><span>正在读取 Kafka 元数据</span></div>}
export function Dialog({title,children,onClose}:{title:string;children:ReactNode;onClose:()=>void}){return <div className="scrim" onMouseDown={event=>event.target===event.currentTarget&&onClose()}><section className="dialog"><header><span>{title}</span><button onClick={onClose}>×</button></header>{children}</section></div>}
