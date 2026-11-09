import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'

/**
 * End-to-end multi-format export through the real UI.
 *
 * Narrative: create one group and one GET API via the dialogs (the same UI
 * flow as the round-trip spec), then export the selection three times
 * through the export dialog -- as OpenAPI, as a Postman Collection and as
 * HTML -- and verify each download's content: the OpenAPI document is a
 * 3.1.0 doc with the expected path key and operation summary, the Postman
 * collection carries the v2.1 schema URL and a folder item for the group,
 * and the HTML document contains an api-section for the API.
 *
 * Idempotency: the suite must pass twice in a row against the same server
 * and database. The group name carries a Date.now() suffix and every
 * assertion is scoped to that unique name, so leftovers from a previous
 * run never interfere. No waitForTimeout calls -- every wait is an
 * auto-retrying expect/locator action.
 */

test('exporting in OpenAPI, Postman and HTML formats produces valid documents', async ({
  page,
  context,
}, testInfo) => {
  // NOTE: the group name must NOT contain the word "Export" (or "New
  // API"): role-locator name matching is substring-based, and the sidebar
  // renders each group header as a button whose accessible name embeds the
  // group name, so such a name would hijack this spec's (and the round-trip
  // spec's) `getByRole('button', { name: 'Export' })` .last() click.
  const groupName = `E2E Multi-Format ${Date.now()}`
  const apiName = 'List Orders'

  // Pin the locale to English before any navigation; all locators below
  // rely on English labels (the backend reads the same cookie for HTML
  // exports).
  await context.addCookies([{ name: 'locale', value: 'en', url: 'http://localhost:3999' }])

  await page.goto('/')
  await expect(page.getByText('API Documentation')).toBeVisible()

  // --- 1. Create one group + one API via the dialogs ----------------------
  await page.getByTitle('New Group').click()
  await page.locator('#name').fill(groupName)
  await page
    .getByRole('dialog')
    .getByRole('button', { name: 'Create', exact: true })
    .click()
  await expect(page.getByText(groupName, { exact: true })).toBeVisible()

  await page.getByText(groupName, { exact: true }).click()
  // The dialog trigger button is wrapped in a div[role=button], so the role
  // locator matches twice; .last() is the real <button> element.
  await page.getByRole('button', { name: 'New API' }).last().click()

  const createDialog = page.getByRole('dialog')
  await createDialog.locator('#apiName').fill(apiName)
  // GET is the method combobox's default; only the endpoint is needed.
  await createDialog.locator('#endpoint').fill('/orders')
  await createDialog.getByRole('button', { name: 'Create API' }).click()

  const groupBlock = page.locator('[data-group-id]').filter({ hasText: groupName })
  await expect(groupBlock.getByText(apiName, { exact: true })).toBeVisible()

  // exportAs opens the export dialog, selects everything, switches the
  // format radio and returns the captured download saved to a per-run
  // output path. The Export locator is exact-matched because substring
  // matching would also hit group-header buttons whose accessible names
  // embed a group name.
  async function exportAs(format: 'openapi' | 'postman' | 'html', fileName: string): Promise<string> {
    await page.getByRole('button', { name: 'Export', exact: true }).last().click()
    const exportDialog = page.getByRole('dialog')
    await exportDialog.getByRole('button', { name: 'Select All' }).click()
    await exportDialog.locator(`[data-testid="export-format-${format}"]`).click()

    const downloadPromise = page.waitForEvent('download')
    await exportDialog.getByRole('button', { name: 'Export', exact: true }).click()
    const download = await downloadPromise
    const exportPath = testInfo.outputPath(fileName)
    await download.saveAs(exportPath)
    return exportPath
  }

  // --- 2. OpenAPI export ---------------------------------------------------
  const openapiPath = await exportAs('openapi', 'knot-export-openapi.json')
  const openapiDoc = JSON.parse(readFileSync(openapiPath, 'utf8')) as {
    openapi: string
    info: { title: string }
    paths: Record<string, { get?: { summary: string } }>
  }
  expect(openapiDoc.openapi).toBe('3.1.0')
  // Operations are keyed by endpoint path and lower-cased method; the
  // operation summary is the API name.
  expect(openapiDoc.paths['/orders'], 'OpenAPI export must contain the /orders path').toBeDefined()
  expect(openapiDoc.paths['/orders'].get?.summary).toBe(apiName)

  // --- 3. Postman export ---------------------------------------------------
  const postmanPath = await exportAs('postman', 'knot-export-postman.json')
  const postmanDoc = JSON.parse(readFileSync(postmanPath, 'utf8')) as {
    info: { name: string; schema: string }
    item: Array<{ name: string; item?: Array<{ name: string }> }>
  }
  // The v2.1 schema URL identifies the collection format.
  expect(postmanDoc.info.schema).toContain('schema.getpostman.com')
  // Each exported group becomes one folder item carrying its APIs.
  const folder = postmanDoc.item.find((entry) => entry.name === groupName)
  expect(folder, `Postman export must contain a folder for ${groupName}`).toBeDefined()
  expect(folder!.item?.map((request) => request.name)).toContain(apiName)

  // --- 4. HTML export ------------------------------------------------------
  const htmlPath = await exportAs('html', 'knot-export.html')
  const html = readFileSync(htmlPath, 'utf8')
  // The static HTML document renders one api-section per exported API.
  expect(html).toContain('api-section')
  expect(html).toContain(apiName)

  // --- 5. Delete the group so leftovers cannot interfere later ------------
  await groupBlock.locator('button.h-6.w-6').first().click()
  await page.getByRole('button', { name: 'Delete Group' }).click()
  await page
    .getByRole('alertdialog')
    .getByRole('button', { name: 'Delete', exact: true })
    .click()
  await expect(page.getByText(groupName, { exact: true })).toHaveCount(0)
})
