import { describe, expect, it, vi } from 'vitest';
import { authenticate } from './LoginPage';

describe('authenticate', () => {
  it('posts credentials to the login endpoint', async () => {
    const fetcher = vi.fn(async () => new Response(null, { status: 204 }));

    await authenticate({ username: 'admin', password: 'secret' }, fetcher);

    expect(fetcher).toHaveBeenCalledWith('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: 'admin', password: 'secret' }),
    });
  });

  it('returns a readable error when credentials are rejected', async () => {
    const fetcher = vi.fn(async () => new Response(JSON.stringify({ error: { message: '用户名或密码错误' } }), {
      status: 401,
      headers: { 'Content-Type': 'application/json' },
    }));

    await expect(authenticate({ username: 'admin', password: 'wrong' }, fetcher)).rejects.toThrow('用户名或密码错误');
  });
});
