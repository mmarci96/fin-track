import { test, expect } from '@playwright/test';

// These cover the UI shell and client-side flows. They don't depend on backend
// data — the receipts list renders its header even when the API is unreachable.

test('login screen renders', async ({ page }) => {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'fin-track' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Continue' })).toBeVisible();
});

test('sign in lands on the receipts list', async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel('User ID').fill('1');
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.getByRole('heading', { name: 'Receipts' })).toBeVisible();
});

test('bottom nav reaches the scan flow', async ({ page }) => {
  await page.goto('/login');
  await page.getByRole('button', { name: 'Continue' }).click();

  await page.getByRole('link', { name: 'Scan' }).click();
  await expect(page).toHaveURL(/\/scan$/);
  await expect(page.getByRole('button', { name: 'Take photo' })).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'Choose from library' }),
  ).toBeVisible();
});

test('opens a receipt into the editable form', async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel('User ID').fill('1');
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.getByRole('heading', { name: 'Receipts' })).toBeVisible();

  const firstReceipt = page.locator('a[href^="/receipts/"]').first();
  await firstReceipt.waitFor();
  await firstReceipt.click();

  await expect(page).toHaveURL(/\/receipts\/\d+$/);
  await expect(
    page.getByRole('button', { name: 'Save changes' }),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: 'Add item' })).toBeVisible();
});

test('theme can be toggled in settings', async ({ page }) => {
  await page.goto('/login');
  await page.getByRole('button', { name: 'Continue' }).click();

  await page.getByRole('link', { name: 'Settings' }).click();
  const html = page.locator('html');
  await expect(html).not.toHaveClass(/dark/);

  await page.getByRole('button', { name: 'Switch' }).click();
  await expect(html).toHaveClass(/dark/);
});
