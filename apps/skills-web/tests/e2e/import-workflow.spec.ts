import { expect, test } from '@playwright/test';

test('runs prepare, validate, and pack against the live packaging backend', async ({ page }) => {
  const suffix = Date.now().toString();
  const packageName = `@platform.skill/e2e-import-smoke-${suffix}`;
  const version = `0.0.${suffix.slice(-6)}`;

  await page.goto('/');
  await page.getByRole('link', { name: '导入' }).click();
  await expect(page).toHaveURL(/\/workbench\/import$/);
  await expect(page.getByRole('button', { name: '检查并生成发布预览' })).toBeVisible();

  const importSection = page.locator('section').filter({ has: page.getByRole('heading', { name: '导入并发布 skill' }) });
  const sourceTypeSelect = importSection.getByLabel('来源类型');
  const inlineFilesField = page.getByLabel('当前文件内容', { exact: true });
  const displayNameField = page.getByLabel('显示名', { exact: true });
  const descriptionField = page.getByLabel('描述', { exact: true });
  const metadataSourceTypeSelect = importSection.getByLabel('元数据来源类型');

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

  const packHandoffCard = page.locator('section').filter({ has: page.getByRole('heading', { name: '发布交接' }) }).first();

  await page.getByRole('button', { name: '检查并生成发布预览' }).click();
  await expect(packHandoffCard).toContainText('manifest 摘要');
  await expect(packHandoffCard).toContainText(packageName);

  await page.getByRole('button', { name: '发布 skill' }).click();
  await expect(page.getByRole('heading', { name: '发布结果' })).toBeVisible();
  await expect(packHandoffCard).toContainText('catalog publish');

  await expect(packHandoffCard.getByRole('link', { name: '打开版本详情' })).toBeVisible();
});
