import { Component, ErrorInfo, ReactNode } from 'react';
import { Icon } from './Icon';

type ErrorBoundaryState = { failed: boolean; message: string };

export function errorBoundaryState(error: unknown): ErrorBoundaryState {
  return {
    failed: true,
    message: error instanceof Error && error.message ? error.message : '页面渲染遇到未知异常',
  };
}

export function reloadApplication(reload: () => void = () => window.location.reload()) {
  reload();
}

export class AppErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
  state: ErrorBoundaryState = { failed: false, message: '' };

  static getDerivedStateFromError(error: unknown): ErrorBoundaryState {
    return errorBoundaryState(error);
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Kafka Manager render error', error, info.componentStack);
  }

  render() {
    if (!this.state.failed) return this.props.children;
    return <main className="fatal-error" role="alert">
      <div className="logo-mark">K</div>
      <span className="section-code">RECOVERY MODE</span>
      <h1>页面遇到异常，但 Kafka 操作没有继续执行</h1>
      <p>当前页面的数据可能不完整。重新加载后会重新读取 Kafka 状态，不会自动重复刚才的写操作。</p>
      <code>{this.state.message}</code>
      <button className="button primary" onClick={() => reloadApplication()}><Icon name="refresh" />重新加载页面</button>
    </main>;
  }
}
