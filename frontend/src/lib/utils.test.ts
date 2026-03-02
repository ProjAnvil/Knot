import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { cn, copyToClipboard } from './utils'

describe('cn - CSS class name utility', () => {
	it('should merge class names correctly', () => {
		expect(cn('foo', 'bar')).toBe('foo bar')
	})

	it('should handle conditional classes', () => {
		expect(cn('foo', true && 'bar', false && 'baz')).toBe('foo bar')
	})

	it('should handle undefined and null values', () => {
		expect(cn('foo', undefined, null, 'bar')).toBe('foo bar')
	})

	it('should handle empty strings', () => {
		expect(cn('foo', '', 'bar')).toBe('foo bar')
	})

	it('should handle arrays of classes', () => {
		expect(cn(['foo', 'bar'], 'baz')).toBe('foo bar baz')
	})

	it('should handle objects with boolean values', () => {
		expect(cn({ foo: true, bar: false, baz: true })).toBe('foo baz')
	})

	it('should merge Tailwind classes correctly - later classes win', () => {
		expect(cn('p-4', 'p-2')).toBe('p-2')
	})

	it('should handle conflicting Tailwind classes', () => {
		expect(cn('text-red-500', 'text-blue-500')).toBe('text-blue-500')
	})

	it('should handle empty input', () => {
		expect(cn()).toBe('')
	})

	it('should handle only falsy values', () => {
		expect(cn(false, null, undefined, 0, '')).toBe('')
	})

	it('should preserve order when no conflicts', () => {
		expect(cn('class-a', 'class-b', 'class-c')).toBe('class-a class-b class-c')
	})

	it('should handle numbers as class names', () => {
		expect(cn('class', 123)).toBe('class 123')
	})
})

describe('copyToClipboard', () => {
	let originalClipboard: any
	let originalIsSecureContext: any

	// Helper to create mock textarea element
	function createMockTextarea(): HTMLTextAreaElement {
		return {
			value: '',
			style: {},
			focus: vi.fn(),
			select: vi.fn(),
			setAttribute: vi.fn()
		} as unknown as HTMLTextAreaElement
	}

	beforeEach(() => {
		// Store original values
		originalClipboard = (navigator as any).clipboard
		originalIsSecureContext = (window as any).isSecureContext

		// Mock createElement for fallback tests
		vi.spyOn(document, 'createElement').mockReturnValue(createMockTextarea())
		vi.spyOn(document.body, 'appendChild').mockImplementation(() => document.createElement('div'))
		vi.spyOn(document.body, 'removeChild').mockImplementation(() => {})
	})

	afterEach(() => {
		// Restore original values
		if (originalClipboard !== undefined) {
			Object.defineProperty(navigator, 'clipboard', {
				writable: true,
				value: originalClipboard
			})
		}
		if (originalIsSecureContext !== undefined) {
			Object.defineProperty(window, 'isSecureContext', {
				writable: true,
				value: originalIsSecureContext
			})
		}

		vi.restoreAllMocks()
	})

	describe('with Clipboard API available (secure context)', () => {
		let mockWriteText: ReturnType<typeof vi.fn>

		beforeEach(() => {
			mockWriteText = vi.fn().mockResolvedValue(undefined)
			Object.defineProperty(navigator, 'clipboard', {
				writable: true,
				value: { writeText: mockWriteText }
			})
			;(window as any).isSecureContext = true
		})

		it('should copy text using Clipboard API', async () => {
			const result = await copyToClipboard('test text')

			expect(result).toBe(true)
			expect(mockWriteText).toHaveBeenCalledWith('test text')
		})

		it('should return true on successful copy', async () => {
			const result = await copyToClipboard('Hello, World!')

			expect(result).toBe(true)
		})

		it('should handle empty string', async () => {
			const result = await copyToClipboard('')

			expect(result).toBe(true)
			expect(mockWriteText).toHaveBeenCalledWith('')
		})

		it('should handle special characters', async () => {
			const specialText = '测试\n\t🔥特殊字符'
			const result = await copyToClipboard(specialText)

			expect(result).toBe(true)
			expect(mockWriteText).toHaveBeenCalledWith(specialText)
		})

		it('should handle very long strings', async () => {
			const longText = 'a'.repeat(10000)
			const result = await copyToClipboard(longText)

			expect(result).toBe(true)
			expect(mockWriteText).toHaveBeenCalledWith(longText)
		})
	})

	describe('with Clipboard API failing (fallback scenario)', () => {
		beforeEach(() => {
			const mockClipboard = {
				writeText: vi.fn().mockRejectedValue(new Error('Clipboard API failed'))
			}
			Object.defineProperty(navigator, 'clipboard', {
				writable: true,
				value: mockClipboard
			})
			;(window as any).isSecureContext = true

			// Mock execCommand for fallback
			vi.spyOn(document, 'execCommand').mockReturnValue(true)
		})

		it('should fall back to execCommand when Clipboard API fails', async () => {
			const createElementSpy = vi.spyOn(document, 'createElement')
			const result = await copyToClipboard('fallback test')

			expect(result).toBe(true)
			expect(createElementSpy).toHaveBeenCalledWith('textarea')
		})

		it('should return true when fallback succeeds', async () => {
			const result = await copyToClipboard('text')

			expect(result).toBe(true)
		})
	})

	describe('without Clipboard API (non-secure context)', () => {
		beforeEach(() => {
			Object.defineProperty(navigator, 'clipboard', {
				writable: true,
				value: undefined
			})
			;(window as any).isSecureContext = false

			// Mock execCommand for fallback
			vi.spyOn(document, 'execCommand').mockReturnValue(true)
		})

		it('should use fallback when Clipboard API is unavailable', async () => {
			const createElementSpy = vi.spyOn(document, 'createElement')
			const result = await copyToClipboard('no clipboard API')

			expect(result).toBe(true)
			expect(createElementSpy).toHaveBeenCalledWith('textarea')
		})
	})

	describe('error scenarios', () => {
		it('should handle fallback failure gracefully', async () => {
			Object.defineProperty(navigator, 'clipboard', {
				writable: true,
				value: undefined
			})
			;(window as any).isSecureContext = false

			vi.spyOn(document, 'execCommand').mockImplementation(() => {
				throw new Error('execCommand failed')
			})

			const result = await copyToClipboard('test')

			expect(result).toBe(false)
		})

		it('should return false when both Clipboard API and fallback fail', async () => {
			const mockClipboard = {
				writeText: vi.fn().mockRejectedValue(new Error('Failed'))
			}
			Object.defineProperty(navigator, 'clipboard', {
				writable: true,
				value: mockClipboard
			})
			;(window as any).isSecureContext = true

			vi.spyOn(document, 'execCommand').mockReturnValue(false)

			const result = await copyToClipboard('should fail')

			expect(result).toBe(false)
		})
	})

	describe('boundary conditions', () => {
		let mockWriteText: ReturnType<typeof vi.fn>

		beforeEach(() => {
			mockWriteText = vi.fn().mockResolvedValue(undefined)
			Object.defineProperty(navigator, 'clipboard', {
				writable: true,
				value: { writeText: mockWriteText }
			})
			;(window as any).isSecureContext = true
		})

		it('should handle whitespace-only string', async () => {
			const result = await copyToClipboard('   ')

			expect(result).toBe(true)
		})

		it('should handle newlines and tabs', async () => {
			const result = await copyToClipboard('line1\nline2\tindented')

			expect(result).toBe(true)
		})

		it('should handle Unicode emoji', async () => {
			const result = await copyToClipboard('🎉👍🚀')

			expect(result).toBe(true)
		})

		it('should handle mixed script characters', async () => {
			const result = await copyToClipboard('Hello你好مرحبا')

			expect(result).toBe(true)
		})
	})
})
