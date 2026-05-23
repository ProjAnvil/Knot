/**
 * HTML sanitizer to prevent XSS attacks when rendering user-provided HTML/Markdown
 * Uses a whitelist-based approach to allow only safe HTML elements and attributes
 */

const ALLOWED_TAGS = new Set([
	'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
	'p', 'br', 'hr',
	'ul', 'ol', 'li',
	'blockquote', 'pre', 'code',
	'strong', 'em', 'b', 'i', 'u', 's', 'del', 'ins',
	'a', 'img',
	'table', 'thead', 'tbody', 'tr', 'th', 'td',
	'div', 'span',
	'dl', 'dt', 'dd',
	'sup', 'sub',
	'details', 'summary',
	'kbd', 'var', 'abbr', 'mark',
])

const ALLOWED_ATTRIBUTES: Record<string, Set<string>> = {
	'a': new Set(['href', 'title', 'target', 'rel']),
	'img': new Set(['src', 'alt', 'title', 'width', 'height']),
	'th': new Set(['align']),
	'td': new Set(['align']),
	'code': new Set(['class']),
	'pre': new Set(['class']),
	'span': new Set(['class']),
	'div': new Set(['class']),
	'h1': new Set(['id']),
	'h2': new Set(['id']),
	'h3': new Set(['id']),
	'h4': new Set(['id']),
	'h5': new Set(['id']),
	'h6': new Set(['id']),
}

const DANGEROUS_PROTOCOLS = /^\s*(javascript|vbscript|data):/i

/**
 * Sanitize HTML string to prevent XSS attacks
 * Removes all non-whitelisted tags, attributes, and dangerous URLs
 */
export function sanitizeHtml(html: string): string {
	if (!html) return ''

	const parser = new DOMParser()
	const doc = parser.parseFromString(html, 'text/html')
	const body = doc.body

	sanitizeNode(body)

	return body.innerHTML
}

function sanitizeNode(node: Node): void {
	const childNodes = Array.from(node.childNodes)

	for (const child of childNodes) {
		if (child.nodeType === Node.TEXT_NODE) {
			// Text nodes are safe
			continue
		}

		if (child.nodeType === Node.COMMENT_NODE) {
			// Remove comments
			node.removeChild(child)
			continue
		}

		if (child.nodeType !== Node.ELEMENT_NODE) {
			node.removeChild(child)
			continue
		}

		const element = child as Element
		const tagName = element.tagName.toLowerCase()

		if (!ALLOWED_TAGS.has(tagName)) {
			// Replace disallowed element with its text content
			const textNode = document.createTextNode(element.textContent || '')
			node.replaceChild(textNode, child)
			continue
		}

		// Remove disallowed attributes
		const allowedAttrs = ALLOWED_ATTRIBUTES[tagName] || new Set()
		const attributes = Array.from(element.attributes)

		for (const attr of attributes) {
			if (!allowedAttrs.has(attr.name)) {
				element.removeAttribute(attr.name)
			}
		}

		// Check for dangerous URLs in href and src attributes
		const href = element.getAttribute('href')
		if (href && DANGEROUS_PROTOCOLS.test(href)) {
			element.removeAttribute('href')
		}

		const src = element.getAttribute('src')
		if (src && DANGEROUS_PROTOCOLS.test(src)) {
			element.removeAttribute('src')
		}

		// Force links to open in new tab safely
		if (tagName === 'a') {
			element.setAttribute('target', '_blank')
			element.setAttribute('rel', 'noopener noreferrer')
		}

		// Recursively sanitize children
		sanitizeNode(element)
	}
}
