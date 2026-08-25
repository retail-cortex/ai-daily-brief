import { test, expect } from '@playwright/test';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

test('has title', async ({ page }) => {
  // Load the compiled index.html directly from build output
  const indexPath = path.resolve(__dirname, '../dist/index.html');
  await page.goto(`file://${indexPath}`);

  // Expect a title "to contain" a substring.
  await expect(page).toHaveTitle(/AI & Cloud Intelligence Agent/);
});
