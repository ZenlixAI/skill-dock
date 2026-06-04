import { expect, test } from '@playwright/test';

const FIXTURE_PACKAGE = '@platform.skill/pdf-skill';
const FIXTURE_VERSION = '1.2.0';

test('browses the live catalog and drills into a version', async ({ page }) => {
  await page.goto('/');

  await expect(page.getByText('Real API mode')).toBeVisible();

  await page.getByRole('link', { name: 'Catalog', exact: true }).click();
  await expect(page).toHaveURL(/\/catalog$/);

  const skillCard = page.locator('article').filter({ hasText: FIXTURE_PACKAGE }).first();
  await expect(skillCard).toBeVisible();
  await skillCard.getByRole('link', { name: 'Open skill' }).click();

  await expect(page).toHaveURL(new RegExp(`/skills/${encodeURIComponent(FIXTURE_PACKAGE)}$`));
  await expect(page.getByRole('heading', { name: 'PDF Skill' })).toBeVisible();

  const versionCard = page.locator('article').filter({ hasText: FIXTURE_VERSION, has: page.getByRole('link', { name: 'Open version' }) }).first();
  await expect(versionCard).toBeVisible();
  await versionCard.getByRole('link', { name: 'Open version' }).click();

  await expect(page).toHaveURL(new RegExp(`/versions/${FIXTURE_VERSION}$`));

  await expect(page.getByRole('heading', { name: `${FIXTURE_PACKAGE} · ${FIXTURE_VERSION}` })).toBeVisible();
  const registryBundleCard = page.locator('section').filter({ has: page.getByRole('heading', { name: 'Registry & Bundle' }) }).first();
  await expect(registryBundleCard).toContainText('published');
  await expect(registryBundleCard).toContainText('resolvable');
  await expect(registryBundleCard).toContainText('commands, scripts');
  await expect(registryBundleCard).toContainText(/sha256:/);
});
