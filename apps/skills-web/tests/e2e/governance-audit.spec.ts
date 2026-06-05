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

  await page.getByLabel('动作').selectOption(fixture.action);
  await page.getByLabel('原因').fill(`playwright smoke ${fixture.action} ${Date.now()}`);
  await page.getByLabel('替代版本建议').fill('1.2.0');
  await page.getByRole('checkbox', { name: '我理解这会改变下游消费者的可安装性。' }).check();
  await page.getByRole('button', { name: '提交治理变更' }).click();

  await expect(page.getByRole('heading', { name: '操作结果' })).toBeVisible();
  await expect(page.getByRole('link', { name: '打开审计线索' })).toBeVisible();

  await page.getByRole('link', { name: '打开审计线索' }).click();

  await expect(page).toHaveURL(/\/workbench\/audit\?/);
  const url = new URL(page.url());
  expect(url.searchParams.get('package')).toBe(FIXTURE_PACKAGE);
  expect(url.searchParams.get('version')).toBe(fixture.version);
  expect(url.searchParams.get('outcome')).toBe('succeeded');

  await expect(page.getByLabel('包名')).toHaveValue(FIXTURE_PACKAGE);
  await expect(page.getByLabel('版本')).toHaveValue(fixture.version);
  await expect(page.locator('article').filter({ hasText: 'governance.version' }).first()).toBeVisible();
});

async function openGovernableFixture(page: Page) {
  await page.goto('/catalog');
  await page.getByLabel('搜索').fill('code-review');
  const skillCard = page.locator('article').filter({ hasText: FIXTURE_PACKAGE }).first();
  await expect(skillCard).toBeVisible();
  await skillCard.getByRole('link', { name: '打开' }).click();
  await expect(page.getByRole('heading', { name: /code-review/ })).toBeVisible();

  const versionRow = page.locator('article').filter({ hasText: FIXTURE_VERSION }).first();
  await expect(versionRow).toBeVisible();
  await versionRow.getByRole('link').click();
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
  const registryBundleCard = page.locator('section').filter({ has: page.getByRole('heading', { name: '仓信息与 Bundle' }) }).first();
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
