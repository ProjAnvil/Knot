// Component tests for the ExportDialog format picker
import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from 'vitest'
import { render, screen, fireEvent, waitFor, within, cleanup } from '@testing-library/svelte'
import '@testing-library/jest-dom/vitest'
import { addMessages, init } from 'svelte-i18n'
import en from '../../i18n/locales/en.json'
import ExportDialog from './ExportDialog.svelte'
import type { GroupWithApis } from '$lib/types'

// Initialize svelte-i18n with real messages so $_() resolves actual translations.
// svelte-i18n v4 has no `messages` init option, so load the dictionary up front.
addMessages('en', en)
init({ fallbackLocale: 'en', initialLocale: 'en' })

const mockGroups: GroupWithApis[] = [
	{
		id: 1,
		name: 'Users',
		createdAt: '2026-01-01T00:00:00Z',
		updatedAt: '2026-01-01T00:00:00Z',
		apis: [
			{
				id: 1,
				groupId: 1,
				name: 'List users',
				method: 'GET',
				endpoint: '/users',
				type: 'rest',
				note: null,
				order: 0,
				createdAt: '2026-01-01T00:00:00Z',
				updatedAt: '2026-01-01T00:00:00Z'
			},
			{
				id: 2,
				groupId: 1,
				name: 'Create user',
				method: 'POST',
				endpoint: '/users',
				type: 'rest',
				note: null,
				order: 1,
				createdAt: '2026-01-01T00:00:00Z',
				updatedAt: '2026-01-01T00:00:00Z'
			}
		]
	}
]

const mockFetch = vi.fn()

// Minimal Response-like object covering what exportApis() uses
function mockExportResponse(filename: string) {
	return {
		ok: true,
		headers: new Headers({ 'Content-Disposition': `attachment; filename="${filename}"` }),
		blob: async () => new Blob(['exported'], { type: 'application/octet-stream' })
	}
}

const FORMATS = ['html', 'json', 'openapi', 'postman'] as const

let clickSpy: MockInstance<HTMLAnchorElement['click']>
let clickedAnchor: HTMLAnchorElement | undefined
let createObjectURLMock: ReturnType<typeof vi.fn>
let revokeObjectURLMock: ReturnType<typeof vi.fn>
const originalCreateObjectURL = URL.createObjectURL
const originalRevokeObjectURL = URL.revokeObjectURL

async function openDialog() {
	// The dialog trigger is the only element with the export title before opening
	await fireEvent.click(screen.getByText('Export'))
	return screen.getByRole('dialog')
}

beforeEach(() => {
	mockFetch.mockReset()
	mockFetch.mockResolvedValue(mockExportResponse('knot-export-1.json'))
	vi.stubGlobal('fetch', mockFetch)

	// Stub browser download APIs: happy-dom does not implement object URLs,
	// and anchor clicks would try to navigate the test environment.
	clickedAnchor = undefined
	clickSpy = vi
		.spyOn(HTMLAnchorElement.prototype, 'click')
		.mockImplementation(function (this: HTMLAnchorElement) {
			clickedAnchor = this
		})
	createObjectURLMock = vi.fn(() => 'blob:mock-url')
	revokeObjectURLMock = vi.fn()
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	;(URL as any).createObjectURL = createObjectURLMock
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	;(URL as any).revokeObjectURL = revokeObjectURLMock
})

afterEach(() => {
	cleanup()
	clickSpy.mockRestore()
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	;(URL as any).createObjectURL = originalCreateObjectURL
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	;(URL as any).revokeObjectURL = originalRevokeObjectURL
	vi.unstubAllGlobals()
})

describe('ExportDialog format picker', () => {
	it('renders four format radio cards with labels and descriptions', async () => {
		render(ExportDialog, { groups: mockGroups })
		const dialog = await openDialog()

		const expectedLabels: Record<string, string> = {
			html: 'HTML',
			json: 'JSON',
			openapi: 'OpenAPI',
			postman: 'Postman'
		}
		for (const fmt of FORMATS) {
			const card = screen.getByTestId(`export-format-${fmt}`)
			expect(card).toBeInTheDocument()
			expect(card).toHaveTextContent(expectedLabels[fmt])
		}

		// Each card carries a one-line description (locale files store
		// "html.desc" as a sibling key of "html", so use bracket access)
		const htmlDesc = en.export.format['html.desc']
		expect(screen.getByTestId('export-format-html')).toHaveTextContent(htmlDesc)
		expect(within(dialog).getByText(en.export.formatLabel)).toBeInTheDocument()
	})

	it('defaults to the html format', async () => {
		render(ExportDialog, { groups: mockGroups })
		await openDialog()

		const htmlRadio = within(screen.getByTestId('export-format-html')).getByRole(
			'radio'
		) as HTMLInputElement
		const jsonRadio = within(screen.getByTestId('export-format-json')).getByRole(
			'radio'
		) as HTMLInputElement

		expect(htmlRadio.checked).toBe(true)
		expect(jsonRadio.checked).toBe(false)
	})

	it('exports in the selected format and downloads the server-provided filename', async () => {
		render(ExportDialog, { groups: mockGroups })
		const dialog = await openDialog()

		// Select every API, then pick the json format
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Select All' }))
		await fireEvent.click(within(screen.getByTestId('export-format-json')).getByRole('radio'))

		// Submit with the generic confirm button
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Export' }))

		await waitFor(() => expect(clickSpy).toHaveBeenCalledTimes(1))

		// Request body carries the selected format
		expect(mockFetch).toHaveBeenCalledTimes(1)
		const [url, options] = mockFetch.mock.calls[0]
		expect(url).toBe('/api/export')
		expect(options.body).toContain('"format":"json"')
		const body = JSON.parse(options.body)
		expect(body.apiIds).toEqual([1, 2])

		// Download triggered with the filename returned by the server
		expect(createObjectURLMock).toHaveBeenCalledTimes(1)
		expect(clickedAnchor?.download).toBe('knot-export-1.json')
	})
})
