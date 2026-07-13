import { FormEvent, useState } from 'react';
import { Icon } from '../../components/Icon';

type Credentials = {
  username: string;
  password: string;
};

type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

export async function authenticate(credentials: Credentials, fetcher: Fetcher = fetch): Promise<void> {
  const response = await fetcher('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credentials),
  });
  if (response.ok) return;

  let message = '登录失败，请稍后重试';
  try {
    const body = await response.json() as { error?: { message?: string } };
    message = body.error?.message || message;
  } catch {
    // Preserve the safe fallback when the server did not return JSON.
  }
  throw new Error(message);
}

export function LoginPage({ onSuccess }: { onSuccess?: () => void }) {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      await authenticate({ username, password });
      onSuccess?.();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '登录失败，请稍后重试');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="login-page">
      <section className="login-intro">
        <div className="login-brand"><span className="logo-mark">K</span><b>Kafka Manager</b></div>
        <div>
          <span className="section-code">INTERNAL DEVELOPER CONSOLE</span>
          <h1>把 Kafka 日常管理<br />变成一件简单的事。</h1>
          <p>集群状态、Topic、消息与消费组集中在一个轻量控制台。</p>
        </div>
        <ul>
          <li><Icon name="check" />多集群统一查看</li>
          <li><Icon name="check" />消息 JSON 自动格式化</li>
          <li><Icon name="check" />操作日志按日归档</li>
        </ul>
      </section>
      <form className="login-card" onSubmit={submit}>
        <span className="login-kicker">WELCOME BACK</span>
        <h2>登录控制台</h2>
        <p>使用配置文件中的 Web 账户登录</p>
        <label>
          用户名
          <input value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" required />
        </label>
        <label>
          密码
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" required />
        </label>
        {error && <p className="login-error" role="alert">{error}</p>}
        <button className="login-submit" type="submit" disabled={submitting}>
          {submitting ? '正在验证…' : <>登录 <Icon name="arrow" /></>}
        </button>
        <small>仅供内部 Kafka 日常管理使用</small>
      </form>
    </main>
  );
}
