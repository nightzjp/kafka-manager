import { expect, Page, test } from '@playwright/test';

const cluster = {
  id: 'protected', name: '受保护集群', online: true, latencyMs: 12, brokers: 3, topics: 1,
  partitions: 3, consumerGroups: 1, underReplicated: 0, totalLag: 12, readOnly: true,
};
const topic = {
  Name: 'orders.v1', Internal: false, PartitionCount: 3, ReplicationFactor: 3, UnderReplicated: 0,
  Partitions: [{ ID: 0, Leader: 1, Replicas: [1, 2, 3], ISR: [1, 2, 3], OfflineReplicas: [] }],
};

async function mockApplication(page: Page, authenticated = true) {
  let signedIn = authenticated;
  await page.addInitScript(() => {
    if (!sessionStorage.getItem('kafka-manager.e2e-initialized')) {
      localStorage.clear();
      sessionStorage.setItem('kafka-manager.e2e-initialized', 'true');
    }
  });
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    const json = (body: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    if (path === '/api/v1/auth/me') return signedIn ? json({ username: 'admin' }) : json({ error: { message: '未登录' } }, 401);
    if (path === '/api/v1/auth/login') { signedIn = true; return json({ status: 'ok' }); }
    if (path === '/api/v1/dashboard') return json({ items: [cluster], history: { protected: [] } });
    if (path === '/api/v1/clusters/protected/topics') return json({ items: [topic], total: 1, page: 1, pageSize: 200 });
    if (path === '/api/v1/clusters/protected/messages') return json({
      items: [{ topic: 'orders.v1', partition: 0, offset: 42, timestamp: 1783929198754, key: 'device-1', value: '{"sn":"541288EPC11C0013D5967C34","used":51583}', headers: [] }],
      scanned: 1, matched: 1, skippedInvalidJson: 0, resultLimited: false, scanLimited: false,
    });
    if (path === '/api/v1/clusters/protected/consumer-groups') return json({ items: [{
      Name: 'orders-worker', State: 'Stable', Protocol: 'consumer', MemberCount: 2, TotalLag: 12,
      Partitions: [{ topic: 'orders.v1', partition: 0, currentOffset: 88, endOffset: 100, lag: 12 }],
    }] });
    if (path.endsWith('/configs')) return json({ items: [] });
    return json({ items: [] });
  });
}

test('logs in and keeps the desktop sidebar preference', async ({ page }) => {
  await mockApplication(page, false);
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '登录控制台' })).toBeVisible();
  await page.getByLabel('密码').fill('secret');
  await page.getByRole('button', { name: /登录/ }).click();
  await expect(page.getByText('Kafka 态势总览')).toBeVisible();
  await page.getByRole('button', { name: '收起侧栏' }).click();
  await expect(page.locator('.sidebar')).toHaveClass(/collapsed/);
  await page.reload();
  await expect(page.locator('.sidebar')).toHaveClass(/collapsed/);
  await page.getByRole('button', { name: '展开侧栏' }).click();
  await expect(page.locator('.sidebar')).not.toHaveClass(/collapsed/);
  await page.getByTitle('主题：system').click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
});

test('opens Topic messages, formats JSON, and explains read-only protection', async ({ page }) => {
  await mockApplication(page);
  await page.goto('/clusters/protected/topics');
  await expect(page.getByText('只读', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '只读集群' })).toBeDisabled();
  await page.getByRole('button', { name: 'orders.v1', exact: true }).click();
  await expect(page.getByText('当前集群为只读模式')).toBeVisible();
  await page.getByRole('tab', { name: '消息' }).click();
  await expect(page.locator('.json-string').getByText('"541288EPC11C0013D5967C34"', { exact: true })).toBeVisible();
  await expect(page.getByRole('tabpanel').getByRole('button', { name: '只读集群' })).toBeDisabled();
});

test('opens Consumer diagnostics and disables Offset reset in read-only mode', async ({ page }) => {
  await mockApplication(page);
  await page.goto('/clusters/protected/consumers');
  await page.getByRole('button', { name: /orders-worker/ }).first().click();
  await expect(page.getByRole('heading', { name: 'orders-worker' })).toBeVisible();
  await expect(page.getByRole('button', { name: '只读集群' })).toBeDisabled();
});
