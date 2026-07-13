import { FormEvent, useState } from 'react';

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
    <main>
      <form onSubmit={submit}>
        <h1>Kafka Manager</h1>
        <p>登录后管理开发与测试 Kafka 集群</p>
        <label>
          用户名
          <input value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" required />
        </label>
        <label>
          密码
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" required />
        </label>
        {error && <p role="alert">{error}</p>}
        <button type="submit" disabled={submitting}>{submitting ? '登录中…' : '登录'}</button>
      </form>
    </main>
  );
}
