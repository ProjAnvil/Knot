import { fileURLToPath } from 'node:url'
import { expect, test, type Page } from '@playwright/test'

/**
 * End-to-end OpenAPI 3.0 import through the real UI.
 *
 * Narrative: feed e2e/fixtures/openapi-petstore-min.json to the import
 * dialog, confirm, then verify the first tag ("pets") became the sidebar
 * group, GET /pets and POST /pets imported as HTTP APIs, the POST /pets
 * request parameters were resolved from the components.schemas.Pet $ref
 * (including the "status" description), and the operation carrying
 * "x-knot-type": "rpc" imported as an RPC-typed API.
 *
 * Idempotency: the suite must pass twice in a row against the same server
 * and database. The fixture's single group carries a fixed name, so the
 * test deletes it at the end; a dry-run against a clean database therefore
 * always reports the same counts. No waitForTimeout calls -- every wait is
 * an auto-retrying expect/locator action.
 */

// The group every operation's first tag maps to.
const PETS_GROUP = 'pets'

const fixturePath = fileURLToPath(new URL('./fixtures/openapi-petstore-min.json', import.meta.url))

// deleteGroupByName removes one sidebar group through its dropdown menu,
// using the same kebab -> "Delete Group" -> confirm flow as the round-trip
// spec.
async function deleteGroupByName(page: Page, groupName: string) {
  const groupBlock = page
    .locator('[data-group-id]')
    .filter({ has: page.getByText(groupName, { exact: true }) })
  await groupBlock.locator('button.h-6.w-6').first().click()
  await page.getByRole('button', { name: 'Delete Group' }).click()
  await page
    .getByRole('alertdialog')
    .getByRole('button', { name: 'Delete', exact: true })
    .click()
  await expect(page.getByText(groupName, { exact: true })).toHaveCount(0)
}

test('importing an OpenAPI document maps tags, $refs and rpc markers', async ({
  page,
  context,
}) => {
  // Pin the locale to English before any navigation; all locators below
  // rely on English labels.
  await context.addCookies([{ name: 'locale', value: 'en', url: 'http://localhost:3999' }])

  await page.goto('/')
  await expect(page.getByText('API Documentation')).toBeVisible()

  // --- 1. Dry-run the fixture through the import dialog -------------------
  await page.getByTestId('import-dialog-trigger').click()
  await page.getByTestId('import-file-input').setInputFiles(fixturePath)

  const preview = page.getByTestId('import-preview')
  await expect(preview).toBeVisible()

  // Every operation is tagged "pets", so one group is created.
  await expect(preview.getByTestId('import-count-groups-created')).toContainText(
    /Groups created:\s*1/
  )
  await expect(preview.getByTestId('import-count-groups-merged')).toContainText(
    /Groups merged:\s*0/
  )
  // List pets (GET /pets), Create pet (POST /pets), Fetch pets batch (rpc).
  await expect(preview.getByTestId('import-count-apis-imported')).toContainText(
    /APIs imported:\s*3/
  )
  // Only POST /pets contributes parameters: the $ref'd Pet schema resolves
  // to name, status and tag nodes.
  await expect(preview.getByTestId('import-count-parameters-imported')).toContainText(
    /Parameters imported:\s*3/
  )

  // --- 2. Confirm and verify the tag-created group ------------------------
  await page.getByTestId('import-confirm').click()

  await expect(page.getByText(PETS_GROUP, { exact: true })).toBeVisible()
  await page.getByText(PETS_GROUP, { exact: true }).click()
  const petsBlock = page
    .locator('[data-group-id]')
    .filter({ has: page.getByText(PETS_GROUP, { exact: true }) })

  // GET /pets and POST /pets both imported under the tag group.
  await expect(
    petsBlock.locator('[data-api-id]').filter({ hasText: 'List pets' })
  ).toBeVisible()
  await expect(
    petsBlock.locator('[data-api-id]').filter({ hasText: 'Create pet' })
  ).toBeVisible()

  // --- 3. POST /pets parameters resolved from the $ref'd Pet schema -------
  await petsBlock.locator('[data-api-id]').filter({ hasText: 'Create pet' }).click()

  await expect(page.getByText('/pets', { exact: true })).toBeVisible()
  await expect(page.getByText('POST', { exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'name', exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'status', exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'tag', exact: true })).toBeVisible()
  // The $ref resolution carried the property description through.
  await expect(page.getByText('pet status in the store')).toBeVisible()

  // --- 4. The x-knot-type:"rpc" operation imported as RPC-typed -----------
  await petsBlock.locator('[data-api-id]').filter({ hasText: 'Fetch pets batch' }).click()

  await expect(page.getByText('/pets/search')).toBeVisible()
  // The detail header renders the API type as a badge; RPC items carry no
  // HTTP method badge.
  await expect(page.getByText('RPC', { exact: true })).toBeVisible()

  // --- 5. Clean up so the next run starts with the same counts ------------
  await deleteGroupByName(page, PETS_GROUP)
})
