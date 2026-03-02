import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs))
}

/**
 * Copy text to clipboard with fallback for non-HTTPS environments
 * @param text - The text to copy
 * @returns true if copy was successful, false otherwise
 */
export async function copyToClipboard(text: string): Promise<boolean> {
	// Try modern clipboard API first
	if (navigator.clipboard && window.isSecureContext) {
		try {
			await navigator.clipboard.writeText(text)
			return true
		} catch (error) {
			console.error('Clipboard API failed, trying fallback:', error)
			return fallbackCopyTextToClipboard(text)
		}
	}
	// Use fallback for non-HTTPS or older browsers
	return fallbackCopyTextToClipboard(text)
}

/**
 * Fallback copy method using deprecated execCommand
 * Used when Clipboard API is not available
 */
function fallbackCopyTextToClipboard(text: string): boolean {
	const textArea = document.createElement('textarea')
	textArea.value = text
	textArea.style.position = 'fixed'
	textArea.style.top = '0'
	textArea.style.left = '0'
	textArea.style.width = '2em'
	textArea.style.height = '2em'
	textArea.style.padding = '0'
	textArea.style.border = 'none'
	textArea.style.outline = 'none'
	textArea.style.boxShadow = 'none'
	textArea.style.background = 'transparent'

	document.body.appendChild(textArea)
	textArea.focus()
	textArea.select()

	try {
		const successful = document.execCommand('copy')
		document.body.removeChild(textArea)
		return successful
	} catch (err) {
		console.error('Fallback copy failed:', err)
		document.body.removeChild(textArea)
		return false
	}
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChild<T> = T extends { child?: any } ? Omit<T, 'child'> : T
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChildren<T> = T extends { children?: any } ? Omit<T, 'children'> : T
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null }
