import { expect, test } from '@playwright/test';

test('runs prepare, validate, and pack against the live packaging backend', async ({ page }) => {
  const suffix = Date.now().toString();
  const packageName = `@platform.skill/e2e-import-smoke-${suffix}`;
  const version = `0.0.${suffix.slice(-6)}`;

  await page.goto('/import');
  await expect(page.getByRole('heading', { name: '把 prepare → validate → pack 做成显式工作流，而不是一键黑盒。' })).toBeVisible();

  await page.getByLabel('Source type').selectOption('platform_authoring');
  await page.getByLabel('Inline files JSON').fill(
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
  await page.getByLabel('displayName').fill('E2E Import Smoke');
  await page.getByLabel('description').fill('Playwright smoke test for the real packaging workflow.');
  await page.getByLabel('metadata sourceType').selectOption('platform_created');

  const packHandoffCard = page.locator('section').filter({ has: page.getByRole('heading', { name: 'Pack handoff' }) }).first();

  await page.getByRole('button', { name: 'Prepare' }).click();
  await expect(packHandoffCard).toContainText('workspace path');
  await expect(packHandoffCard).toContainText(packageName);

  await page.getByRole('button', { name: 'Validate' }).click();
  await expect(page.getByText('validate ok')).toBeVisible();
  await expect(packHandoffCard).toContainText('manifest digest');

  await page.getByRole('button', { name: 'Pack' }).click();
  await expect(packHandoffCard).toContainText('bundle path');
  await expect(packHandoffCard).toContainText('pack ready for release handoff');
  await expect(packHandoffCard.getByRole('link', { name: 'Continue to release record' })).toBeVisible();
});
