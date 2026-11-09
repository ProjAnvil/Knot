// Component tests for the ImportDialog preview-before-confirm flow
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within, cleanup } from '@testing-library/svelte'
import '@testing-library/jest-dom/vitest'
import { addMessages, init } from 'svelte-i18n'
import en from '../../i18n/locales/en.json'
import ImportDialog from './ImportDialog.svelte'
import { importDocument } from '$lib/api'
import type { ImportPreview } from '$lib/types'

// Initialize svelte-i18n with real messages so $_() resolves actual translations.
addMessages('en', en)
init({ fallbackLocale: 'en', initialLocale: 'en' })

// Stub the api module: the logic under test is the dry-run/confirm flow,
// not the multipart transport layer
vi.mock('$lib/api', () => ({
	importDocument: vi.fn()
}))

const importDocumentMock = vi.mocked(importDocument)

const mockPreview: ImportPreview = {
	format: 'postman',
	groupsCreated: 2,
	groupsMerged: 1,
	apisImported: 5,
	parametersImported: 9,
	warnings: ['Group "Users" merged with existing group', 'Skipped folder "Archive" (empty)']
}

function testFile() {
	return new File(['{"info":{"name":"demo"}}'], 'collection.json', { type: 'application/json' })
}

async function openDialog() {
	await fireEvent.click(screen.getByTestId('import-dialog-trigger'))
	return screen.getByRole('dialog')
}

async function selectFile(dialog: HTMLElement, file: File) {
	const input = within(dialog).getByTestId('import-file-input') as HTMLInputElement
	// happy-dom keeps `files` read-only, so define it directly like userEvent does
	Object.defineProperty(input, 'files', { value: [file], configurable: true })
	await fireEvent.change(input)
}

beforeEach(() => {
	importDocumentMock.mockReset()
})

afterEach(() => {
	cleanup()
})

describe('ImportDialog', () => {
	it('dry-runs on file select and renders counts and warnings in the preview', async () => {
		importDocumentMock.mockResolvedValue({ success: true, data: mockPreview })
		render(ImportDialog, { onImported: vi.fn() })
		const dialog = await openDialog()

		const file = testFile()
		await selectFile(dialog, file)

		await waitFor(() => expect(importDocumentMock).toHaveBeenCalledTimes(1))
		expect(importDocumentMock).toHaveBeenCalledWith(file, true)

		const preview = await screen.findByTestId('import-preview')
		expect(preview).toHaveTextContent('postman')
		expect(within(preview).getByTestId('import-count-groups-created')).toHaveTextContent('2')
		expect(within(preview).getByTestId('import-count-groups-merged')).toHaveTextContent('1')
		expect(within(preview).getByTestId('import-count-apis-imported')).toHaveTextContent('5')
		expect(within(preview).getByTestId('import-count-parameters-imported')).toHaveTextContent('9')
		for (const warning of mockPreview.warnings) {
			expect(within(preview).getByText(warning)).toBeInTheDocument()
		}
	})

	it('confirms with dryRun=false, then notifies the parent via onImported', async () => {
		importDocumentMock
			.mockResolvedValueOnce({ success: true, data: mockPreview })
			.mockResolvedValueOnce({ success: true, data: { ...mockPreview, warnings: [] } })
		const onImported = vi.fn()
		render(ImportDialog, { onImported })
		const dialog = await openDialog()

		const file = testFile()
		await selectFile(dialog, file)

		const confirm = await screen.findByTestId('import-confirm')
		expect(confirm).toBeEnabled()
		await fireEvent.click(confirm)

		await waitFor(() => expect(onImported).toHaveBeenCalledTimes(1))
		expect(importDocumentMock).toHaveBeenCalledTimes(2)
		expect(importDocumentMock).toHaveBeenLastCalledWith(file, false)
	})

	it('renders the translated unsupported-format message for a 415-style error', async () => {
		importDocumentMock.mockResolvedValue({
			success: false,
			error: 'HTTP Error: 415 Unsupported Media Type'
		})
		render(ImportDialog, { onImported: vi.fn() })
		const dialog = await openDialog()

		await selectFile(dialog, testFile())

		const error = await screen.findByTestId('import-error')
		expect(error).toHaveTextContent(en.import.unsupportedFormat)
		expect(screen.queryByTestId('import-preview')).not.toBeInTheDocument()
	})

	it('shows a non-415 server error message verbatim', async () => {
		importDocumentMock.mockResolvedValue({ success: false, error: 'Failed to parse document' })
		render(ImportDialog, { onImported: vi.fn() })
		const dialog = await openDialog()

		await selectFile(dialog, testFile())

		const error = await screen.findByTestId('import-error')
		expect(error).toHaveTextContent('Failed to parse document')
	})
})
