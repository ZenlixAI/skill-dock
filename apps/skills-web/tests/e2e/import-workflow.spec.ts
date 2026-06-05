import { expect, test } from '@playwright/test';

test('runs prepare, validate, and pack against the live packaging backend', async ({ page }) => {
  const suffix = Date.now().toString();
  const packageName = `@platform.skill/e2e-import-smoke-${suffix}`;
  const version = `0.0.${suffix.slice(-6)}`;

  await page.goto('/');
  await page.getByRole('link', { name: '导入' }).click();
  await expect(page).toHaveURL(/\/workbench\/import$/);
  await expect(page.getByRole('button', { name: 'Prepare' })).toBeVisible();

  const sourceTypeSelect = page.locator('select').nth(0);
  const inlineFilesField = page.locator('textarea').nth(0);
  const displayNameField = page.locator('input').nth(2);
  const descriptionField = page.locator('textarea').nth(1);
  const metadataSourceTypeSelect = page.locator('select').nth(1);

  await sourceTypeSelect.selectOption('platform_authoring');
  await inlineFilesField.fill(
    JSON.stringify(
      {
        'commands/run.txt': 'echo smoke',
        'scripts/check.sh': 'printf smoke',
      },
      null,
      2
    )
  );
  await page.getByLabel('packageName').fill(packageName);
  await page.getByLabel('version').fill(version);
  await displayNameField.fill('E2E Import Smoke');
  await descriptionField.fill('Playwright smoke test for the real packaging workflow.');
  await metadataSourceTypeSelect.selectOption('platform_created');

  const packHandoffCard = page.locator('section').filter({ has: page.getByRole('heading', { name: 'Pack 交接' }) }).first();

  await page.getByRole('button', { name: 'Prepare' }).click();
  await expect(packHandoffCard).toContainText('工作目录');
  await expect(packHandoffCard).toContainText(packageName);

  await page.getByRole('button', { name: 'Validate' }).click();
  await expect(page.getByText('validate ok')).toBeVisible();
  await expect(packHandoffCard).toContainText('manifest 摘要');

  await page.getByRole('button', { name: 'Pack' }).click();
  await expect(packHandoffCard).toContainText('bundle 路径');
  await expect(packHandoffCard).toContainText('pack 已可进入发布交接');
  await expect(packHandoffCard.getByRole('link', { name: '继续进入 release record' })).toBeVisible();
});
