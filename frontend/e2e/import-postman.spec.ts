import { fileURLToPath } from 'node:url'
import { expect, test, type Page } from '@playwright/test'

/**
 * End-to-end Postman Collection v2.1 import through the real UI.
 *
 * Narrative: feed e2e/fixtures/postman-collection.json to the import dialog,
 * verify the dry-run preview counts (folders become groups: "Users",
 * "Users/Admin" via nested-folder path flattening, and a group named after
 * the collection for the lone top-level requests), confirm, then verify the
 * sidebar shows the groups and the imported APIs carry the mapping
 * semantics: the raw JSON body of POST /users became request parameters and
 * the "x-knot-type": "rpc" item imported as an RPC-typed API.
 *
 * Idempotency: the suite must pass twice in a row against the same server
 * and database. The groups the fixture creates carry fixed names, so the
 * test deletes every imported group at the end; a dry-run against a clean
 * database therefore always reports the same counts. No waitForTimeout
 * calls -- every wait is an auto-retrying expect/locator action.
 */

// Group names the fixture's folder structure maps to.
const USERS_GROUP = 'Users'
const ADMIN_GROUP = 'Users/Admin'
const COLLECTION_GROUP = 'E2E Postman Collection'

const fixturePath = fileURLToPath(new URL('./fixtures/postman-collection.json', import.meta.url))

// deleteGroupByName removes one sidebar group through its dropdown menu,
// using the same kebab -> "Delete Group" -> confirm flow as the round-trip
// spec. Group names containing each other ("Users" vs "Users/Admin") are
// disambiguated by matching the header's exact text.
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

test('importing a Postman collection maps folders, bodies and rpc markers', async ({
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

  // Folders map to groups: "Users", nested "Users/Admin", plus one group
  // named after info.name for the two lone top-level requests.
  await expect(preview.getByTestId('import-count-groups-created')).toContainText(
    /Groups created:\s*3/
  )
  await expect(preview.getByTestId('import-count-groups-merged')).toContainText(
    /Groups merged:\s*0/
  )
  // List Users, Create User, Delete Admin, Health Check, Submit Job.
  await expect(preview.getByTestId('import-count-apis-imported')).toContainText(
    /APIs imported:\s*5/
  )
  // Only Create User contributes parameters: its raw JSON body {"name",
  // "age"} becomes two request parameter nodes.
  await expect(preview.getByTestId('import-count-parameters-imported')).toContainText(
    /Parameters imported:\s*2/
  )

  // --- 2. Confirm and verify the sidebar ----------------------------------
  await page.getByTestId('import-confirm').click()

  await expect(page.getByText(USERS_GROUP, { exact: true })).toBeVisible()
  await expect(page.getByText(ADMIN_GROUP, { exact: true })).toBeVisible()
  await expect(page.getByText(COLLECTION_GROUP, { exact: true })).toBeVisible()

  // --- 3. The POST /users body became request parameters ------------------
  await page.getByText(USERS_GROUP, { exact: true }).click()
  const usersBlock = page
    .locator('[data-group-id]')
    .filter({ has: page.getByText(USERS_GROUP, { exact: true }) })
  await usersBlock.locator('[data-api-id]').filter({ hasText: 'Create User' }).click()

  await expect(page.getByText('{{baseUrl}}/users')).toBeVisible()
  await expect(page.getByText('POST', { exact: true })).toBeVisible()
  // The raw body {"name": "...", "age": 0} derived these parameter rows.
  await expect(page.getByRole('cell', { name: 'name', exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'age', exact: true })).toBeVisible()
  // The v2.1 {content} description object imported as the API note.
  await expect(page.getByText('Creates a user from a raw JSON body.')).toBeVisible()

  // --- 4. The x-knot-type:"rpc" item imported as an RPC-typed API ---------
  await page.getByText(COLLECTION_GROUP, { exact: true }).click()
  const collectionBlock = page.locator('[data-group-id]').filter({
    has: page.getByText(COLLECTION_GROUP, { exact: true }),
  })
  await collectionBlock.locator('[data-api-id]').filter({ hasText: 'Submit Job' }).click()

  await expect(page.getByText('{{baseUrl}}/jobs/submit')).toBeVisible()
  // The detail header renders the API type as a badge; RPC items carry no
  // HTTP method badge.
  await expect(page.getByText('RPC', { exact: true })).toBeVisible()

  // --- 5. Clean up so the next run starts with the same counts ------------
  await deleteGroupByName(page, USERS_GROUP)
  await deleteGroupByName(page, ADMIN_GROUP)
  await deleteGroupByName(page, COLLECTION_GROUP)
})
