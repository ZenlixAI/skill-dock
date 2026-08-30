import { expect, test, type Page } from '@playwright/test';

test('completes the browser session login flow for reader and operator roles', async ({ page }) => {
  await page.goto('/login?next=%2Fcatalog');

  await expect(page.getByRole('heading', { level: 1, name: '进入技能注册表控制台' })).toBeVisible();

  await page.getByLabel('用户名').fill('reader');
  await page.getByLabel('密码').fill('wrong-password');
  await page.getByRole('button', { name: '登录' }).click();
  await expect(page.getByRole('status')).toHaveText('Invalid username or password.');

  await login(page, 'reader', 'reader-secret');

  await expect(page).toHaveURL(/\/catalog$/);
  await expect(page.getByRole('heading', { name: '发现注册表技能' })).toBeVisible();
  await expect(page.getByText('已登录：reader')).toBeVisible();
  await expect(page.getByRole('button', { name: '退出登录' })).toBeVisible();

  await page.goto('/workbench/governance');
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByText('已登录：reader')).toBeVisible();

  await page.getByRole('button', { name: '退出登录' }).click();
  await expect(page).toHaveURL(/\/login\?next=/);
  await expect(page.getByLabel('用户名')).toBeVisible();

  await login(page, 'operator', 'op-secret');

  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByText('已登录：operator')).toBeVisible();
  await expect(page.getByRole('link', { name: '治理', exact: true })).toBeVisible();

  await page.getByRole('link', { name: '治理', exact: true }).click();
  await expect(page).toHaveURL(/\/workbench\/governance$/);
  await expect(page.getByText('把 retire / deprecate / yank 从后端状态机翻译成可操作的治理控制台。')).toBeVisible();
});

async function login(page: Page, username: string, password: string) {
  await page.getByLabel('用户名').fill(username);
  await page.getByLabel('密码').fill(password);
  await page.getByRole('button', { name: '登录' }).click();
}
