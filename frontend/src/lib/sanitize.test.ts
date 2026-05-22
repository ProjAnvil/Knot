import { describe, it, expect } from 'vitest'
import { sanitizeHtml } from './sanitize'

describe('sanitizeHtml', () => {
	describe('basic functionality', () => {
		it('should return empty string for empty input', () => {
			expect(sanitizeHtml('')).toBe('')
		})

		it('should return empty string for null/undefined-like input', () => {
			expect(sanitizeHtml(null as unknown as string)).toBe('')
			expect(sanitizeHtml(undefined as unknown as string)).toBe('')
		})

		it('should preserve plain text', () => {
			expect(sanitizeHtml('Hello World')).toBe('Hello World')
		})

		it('should preserve allowed HTML tags', () => {
			expect(sanitizeHtml('<p>Hello</p>')).toBe('<p>Hello</p>')
			expect(sanitizeHtml('<strong>Bold</strong>')).toBe('<strong>Bold</strong>')
			expect(sanitizeHtml('<em>Italic</em>')).toBe('<em>Italic</em>')
			expect(sanitizeHtml('<h1>Title</h1>')).toBe('<h1>Title</h1>')
		})

		it('should preserve nested allowed tags', () => {
			const input = '<ul><li><strong>Item 1</strong></li><li>Item 2</li></ul>'
			expect(sanitizeHtml(input)).toBe(input)
		})

		it('should preserve code blocks', () => {
			const input = '<pre><code class="language-js">const x = 1;</code></pre>'
			expect(sanitizeHtml(input)).toBe(input)
		})
	})

	describe('XSS prevention', () => {
		it('should remove script tags', () => {
			const input = '<p>Hello</p><script>alert("xss")</script><p>World</p>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('<script')
			expect(result).not.toContain('</script')
			expect(result).toContain('<p>Hello</p>')
			expect(result).toContain('<p>World</p>')
		})

		it('should remove onclick handlers', () => {
			const input = '<div onclick="alert(\'xss\')">Click me</div>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('onclick')
			expect(result).toContain('Click me')
		})

		it('should remove onerror handlers from images', () => {
			const input = '<img src="x" onerror="alert(\'xss\')" />'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('onerror')
		})

		it('should remove javascript: URLs from links', () => {
			const input = '<a href="javascript:alert(\'xss\')">Click</a>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('javascript:')
		})

		it('should remove data: URLs from images', () => {
			const input = '<img src="data:text/html,<script>alert(1)</script>" />'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('data:')
		})

		it('should remove vbscript: URLs', () => {
			const input = '<a href="vbscript:alert(1)">Click</a>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('vbscript:')
		})

		it('should remove iframe tags', () => {
			const input = '<iframe src="https://evil.com"></iframe>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('<iframe')
		})

		it('should remove style tags', () => {
			const input = '<style>body{background:url("javascript:alert(1)")}</style>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('<style')
		})

		it('should remove form tags', () => {
			const input = '<form action="https://evil.com"><input type="text" /></form>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('<form')
		})

		it('should remove HTML comments', () => {
			const input = '<p>Hello</p><!-- comment --><p>World</p>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('<!--')
			expect(result).toContain('<p>Hello</p>')
			expect(result).toContain('<p>World</p>')
		})

		it('should handle nested malicious content', () => {
			const input = '<div><p><script>alert("xss")</script></p></div>'
			const result = sanitizeHtml(input)
			expect(result).not.toContain('<script')
		})
	})

	describe('safe link handling', () => {
		it('should allow regular http links', () => {
			const input = '<a href="https://example.com">Link</a>'
			const result = sanitizeHtml(input)
			expect(result).toContain('href="https://example.com"')
		})

		it('should add target="_blank" and rel="noopener noreferrer" to links', () => {
			const input = '<a href="https://example.com">Link</a>'
			const result = sanitizeHtml(input)
			expect(result).toContain('target="_blank"')
			expect(result).toContain('rel="noopener noreferrer"')
		})
	})

	describe('attribute filtering', () => {
		it('should remove disallowed attributes', () => {
			const input = '<div class="ok" style="color:red" data-custom="x">Text</div>'
			const result = sanitizeHtml(input)
			expect(result).toContain('class="ok"')
			expect(result).not.toContain('style=')
			expect(result).not.toContain('data-custom')
		})

		it('should allow id on heading elements', () => {
			const input = '<h2 id="my-heading">Title</h2>'
			const result = sanitizeHtml(input)
			expect(result).toContain('id="my-heading"')
		})

		it('should allow alt and src on img', () => {
			const input = '<img src="https://example.com/img.png" alt="An image" />'
			const result = sanitizeHtml(input)
			expect(result).toContain('src="https://example.com/img.png"')
			expect(result).toContain('alt="An image"')
		})
	})

	describe('markdown common elements', () => {
		it('should preserve table structure', () => {
			const input = '<table><thead><tr><th>Header</th></tr></thead><tbody><tr><td>Cell</td></tr></tbody></table>'
			expect(sanitizeHtml(input)).toBe(input)
		})

		it('should preserve blockquotes', () => {
			const input = '<blockquote><p>Quote text</p></blockquote>'
			expect(sanitizeHtml(input)).toBe(input)
		})

		it('should preserve horizontal rules', () => {
			const input = '<hr>'
			expect(sanitizeHtml(input)).toContain('<hr>')
		})

		it('should preserve details/summary elements', () => {
			const input = '<details><summary>Click to expand</summary><p>Content</p></details>'
			expect(sanitizeHtml(input)).toBe(input)
		})
	})
})
