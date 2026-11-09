import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'

/**
 * End-to-end export/import round-trip through the real UI.
 *
 * Narrative: create a group and an API (with a note and nested request
 * parameters) via the dialogs and detail-view editors, export everything as
 * a Knot JSON v1 file through the export dialog, delete the group, then
 * import the file back through the import dialog and verify the group, API,
 * note and parameter tree are restored.
 *
 * Idempotency: the suite must pass twice in a row against the same server
 * and database. Group names carry a Date.now() suffix and every assertion
 * is scoped to that unique name, so leftovers from a previous run never
 * interfere. No waitForTimeout calls -- every wait is an auto-retrying
 * expect/locator action.
 */
test('export/import round-trip restores a group with nested parameters', async ({
  page,
  context,
}, testInfo) => {
  const groupName = `E2E Group ${Date.now()}`
  const apiName = 'Create User'
  const noteText = 'Round-trip note: restore me verbatim.'

  // Pin the locale to English before any navigation. The app defaults to
  // Chinese unless the `locale` cookie is set (backend reads the same cookie
  // for HTML exports), and all locators below rely on English labels.
  await context.addCookies([{ name: 'locale', value: 'en', url: 'http://localhost:3999' }])

  await page.goto('/')
  await expect(page.getByText('API Documentation')).toBeVisible()

  // --- 1. Create the group via the sidebar's "New Group" dialog ---------
  await page.getByTitle('New Group').click()
  await page.locator('#name').fill(groupName)
  await page
    .getByRole('dialog')
    .getByRole('button', { name: 'Create', exact: true })
    .click()
  await expect(page.getByText(groupName, { exact: true })).toBeVisible()

  // --- 2. Expand the group and create the API via CreateApiDialog -------
  await page.getByText(groupName, { exact: true }).click()
  // The dialog trigger button is wrapped in a div[role=button], so the role
  // locator matches twice; .last() is the real <button> element.
  await page.getByRole('button', { name: 'New API' }).last().click()

  const createDialog = page.getByRole('dialog')
  await createDialog.locator('#apiName').fill(apiName)
  // The method <select> is the only combobox offering a POST option.
  await createDialog
    .getByRole('combobox')
    .filter({ has: page.getByRole('option', { name: 'POST' }) })
    .selectOption('POST')
  await createDialog.locator('#endpoint').fill('/users')
  await createDialog.getByRole('button', { name: 'Create API' }).click()

  // The new API shows up inside the group's expanded block in the sidebar.
  const groupBlock = page.locator('[data-group-id]').filter({ hasText: groupName })
  await expect(groupBlock.getByText(apiName, { exact: true })).toBeVisible()

  // --- 3. Open the API detail view ---------------------------------------
  await groupBlock.locator('[data-api-id]').filter({ hasText: apiName }).click()
  await expect(page.getByText('/users')).toBeVisible()

  // --- 4. Add nested request parameters via the parameters editor -------
  // The empty request-parameters section offers an "Add Parameters" button
  // (the response section has one too, so scope via the section heading).
  await page
    .getByRole('heading', { name: 'Request Parameters' })
    .locator('..')
    .getByRole('button', { name: 'Add Parameters' })
    .click()

  const addParamButton = page.getByRole('button', { name: 'Add Parameter', exact: true })
  await addParamButton.click()
  await addParamButton.click()
  await addParamButton.click()

  const paramRows = page.locator('tbody tr')
  await paramRows.nth(0).getByPlaceholder('Name').fill('user')
  await paramRows.nth(0).locator('select').selectOption('object')
  await paramRows.nth(1).getByPlaceholder('Name').fill('name')
  await paramRows.nth(1).locator('select').selectOption('string')
  await paramRows.nth(2).getByPlaceholder('Name').fill('email')
  await paramRows.nth(2).locator('select').selectOption('string')
  // Indent rows 2 and 3 one level so they become children of "user".
  await paramRows.nth(1).getByTitle('Increase indent').click()
  await paramRows.nth(2).getByTitle('Increase indent').click()
  await page.getByRole('button', { name: 'Save', exact: true }).click()

  // The read-only table renders the tree, nested rows prefixed with "└─".
  await expect(page.getByRole('cell', { name: 'user', exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: /email/ })).toBeVisible()

  // --- 5. Add a note via the note editor ---------------------------------
  await page
    .getByRole('heading', { name: 'Note' })
    .locator('..')
    .getByRole('button', { name: 'Edit' })
    .click()
  await page.getByPlaceholder('Add notes here... Supports Markdown syntax.').fill(noteText)
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByText(noteText)).toBeVisible()

  // --- 6. Export via the export dialog ------------------------------------
  // Same trigger-wrapper ambiguity as "New API": .last() is the real button.
  await page.getByRole('button', { name: 'Export' }).last().click()

  const exportDialog = page.getByRole('dialog')
  await exportDialog.getByRole('button', { name: 'Select All' }).click()
  await exportDialog.locator('[data-testid="export-format-json"]').click()

  const downloadPromise = page.waitForEvent('download')
  await exportDialog.getByRole('button', { name: 'Export', exact: true }).click()
  const download = await downloadPromise
  const exportPath = testInfo.outputPath('knot-export.json')
  await download.saveAs(exportPath)

  // The downloaded file must be a Knot JSON v1 document carrying our group,
  // API endpoint, note and the nested parameter tree.
  const doc = JSON.parse(readFileSync(exportPath, 'utf8')) as {
    format: string
    version: number
    groups: Array<{
      name: string
      apis: Array<{
        name: string
        endpoint: string
        method: string
        note: string | null
        parameters: {
          request: Array<{
            name: string
            children: Array<{ name: string }>
          }>
        }
      }>
    }>
  }
  expect(doc.format).toBe('knot')
  expect(doc.version).toBe(1)
  // Find by name so leftovers from a previous run (same server+DB) cannot
  // break the assertion; on a clean database our group is groups[0].
  const exportedGroup = doc.groups.find((g) => g.name === groupName)
  expect(exportedGroup, `export must contain group ${groupName}`).toBeDefined()
  if (doc.groups.length === 1) {
    expect(doc.groups[0].name).toBe(groupName)
    expect(doc.groups[0].apis[0].endpoint).toBe('/users')
  }
  const exportedApi = exportedGroup!.apis[0]
  expect(exportedApi.name).toBe(apiName)
  expect(exportedApi.endpoint).toBe('/users')
  expect(exportedApi.method).toBe('POST')
  expect(exportedApi.note).toContain(noteText)
  const userNode = exportedApi.parameters.request.find((p) => p.name === 'user')
  expect(userNode, 'exported parameter tree must contain the "user" node').toBeDefined()
  expect(userNode!.children.map((c) => c.name)).toEqual(
    expect.arrayContaining(['name', 'email'])
  )

  // --- 7. Delete the group via the sidebar dropdown ----------------------
  // The group header row is the first element of the group block; its
  // kebab (MoreVertical) button is the first small icon button.
  await groupBlock.locator('button.h-6.w-6').first().click()
  await page.getByRole('button', { name: 'Delete Group' }).click()
  await page
    .getByRole('alertdialog')
    .getByRole('button', { name: 'Delete', exact: true })
    .click()
  await expect(page.getByText(groupName, { exact: true })).toHaveCount(0)

  // Drop any client-side state left in the detail view so post-import
  // assertions can only pass against freshly restored data.
  await page.goto('/')

  // --- 8. Import the exported file through the import dialog -------------
  await page.getByTestId('import-dialog-trigger').click()
  await page.getByTestId('import-file-input').setInputFiles(exportPath)

  const preview = page.getByTestId('import-preview')
  await expect(preview).toBeVisible()
  // Our group was deleted above, so the dry-run preview must create exactly
  // one group (pre-existing leftovers get merged instead).
  await expect(preview.getByTestId('import-count-groups-created')).toContainText(
    /Groups created:\s*1/
  )

  await page.getByTestId('import-confirm').click()
  await expect(page.getByText(groupName, { exact: true })).toBeVisible()

  // --- 9. Verify the restored content -------------------------------------
  await page.getByText(groupName, { exact: true }).click()
  const restoredGroup = page.locator('[data-group-id]').filter({ hasText: groupName })
  await restoredGroup.locator('[data-api-id]').filter({ hasText: apiName }).click()

  await expect(page.getByText('/users')).toBeVisible()
  await expect(page.getByText('POST', { exact: true })).toBeVisible()
  await expect(page.getByText(noteText)).toBeVisible()
  await expect(page.getByRole('cell', { name: 'user', exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: /email/ })).toBeVisible()
})
