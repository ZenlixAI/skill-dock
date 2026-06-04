import { expect, test, type Page } from '@playwright/test';

const FIXTURE_PACKAGE = '@platform.skill/code-review';
const FIXTURE_VERSION = '0.9.0';

test('submits version governance and lands on the scoped audit view', async ({ page }) => {
  const fixture = await openGovernableFixture(page);
  test.skip(!fixture, `No governable fixture remained for ${FIXTURE_PACKAGE}@${FIXTURE_VERSION}. Reset the sample registry to re-run the mutation path.`);

  if (!fixture) {
    return;
  }
  await expect(page.getByRole('heading', { name: `${FIXTURE_PACKAGE} · ${fixture.version}` })).toBeVisible();

  await page.getByLabel('Action').selectOption(fixture.action);
  await page.getByLabel('Reason').fill(`playwright smoke ${fixture.action} ${Date.now()}`);
  await page.getByLabel('Replacement recommendation').fill('1.2.0');
  await page.getByRole('checkbox', { name: 'I understand this changes installability for downstream consumers.' }).check();
  await page.getByRole('button', { name: 'Submit governance change' }).click();

  await expect(page.getByText('Mutation Result')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Open audit trail' })).toBeVisible();

  await page.getByRole('link', { name: 'Open audit trail' }).click();

  await expect(page).toHaveURL(/\/audit\?/);
  const url = new URL(page.url());
  expect(url.searchParams.get('package')).toBe(FIXTURE_PACKAGE);
  expect(url.searchParams.get('version')).toBe(fixture.version);
  expect(url.searchParams.get('outcome')).toBe('succeeded');

  await expect(page.getByLabel('Package name')).toHaveValue(FIXTURE_PACKAGE);
  await expect(page.getByLabel('Version')).toHaveValue(fixture.version);
  await expect(page.locator('article').filter({ hasText: 'governance.version' }).first()).toBeVisible();
});

async function openGovernableFixture(page: Page) {
  await page.goto(`/skills/${encodeURIComponent(FIXTURE_PACKAGE)}/versions/${FIXTURE_VERSION}`);
  await expect(page.getByRole('heading', { name: `${FIXTURE_PACKAGE} · ${FIXTURE_VERSION}` })).toBeVisible();

  const currentStatus = await readCurrentVersionStatus(page);
  if (currentStatus === 'published') {
    return { version: FIXTURE_VERSION, action: 'deprecate' as const };
  }
  if (currentStatus === 'deprecated') {
    return { version: FIXTURE_VERSION, action: 'yank' as const };
  }

  return null;
}

async function readCurrentVersionStatus(page: Page) {
  const registryBundleCard = page.locator('section').filter({ has: page.getByRole('heading', { name: 'Registry & Bundle' }) }).first();
  await expect
    .poll(
      async () => (await registryBundleCard.textContent()) ?? '',
      {
        message: 'Expected the version detail page to load a concrete version status before governance branching.',
      }
    )
    .toMatch(/published|deprecated|yanked/);

  const stripText = (await registryBundleCard.textContent()) ?? '';
  for (const status of ['published', 'deprecated', 'yanked'] as const) {
    if (stripText.includes(status)) {
      return status;
    }
  }

  throw new Error('Could not determine the loaded version status from the version detail page.');
}
