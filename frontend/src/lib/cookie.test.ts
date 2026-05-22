/**
 * Tests for cookie-related utility functions
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { setCookie, getCookie, deleteCookie } from './cookie'

describe('cookie utilities', () => {
	beforeEach(() => {
		// Clear all cookies before each test
		document.cookie.split(';').forEach(c => {
			document.cookie = c.replace(/^ +/, '').replace(/=.*/, '=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/')
		})
	})

	describe('setCookie', () => {
		it('should set cookie with SameSite=Strict', () => {
			setCookie('locale', 'en')
			expect(document.cookie).toContain('locale=en')
		})

		it('should set cookie with the correct value', () => {
			setCookie('locale', 'zh-CN')
			expect(document.cookie).toContain('locale=zh-CN')
		})

		it('should encode special characters', () => {
			setCookie('test', 'hello world')
			expect(getCookie('test')).toBe('hello world')
		})
	})

	describe('getCookie', () => {
		it('should return null when no cookie is set', () => {
			expect(getCookie('locale')).toBeNull()
		})

		it('should return the value when cookie is set', () => {
			document.cookie = 'locale=en; path=/'
			expect(getCookie('locale')).toBe('en')
		})

		it('should handle multiple cookies correctly', () => {
			document.cookie = 'other=value; path=/'
			document.cookie = 'locale=zh-CN; path=/'
			expect(getCookie('locale')).toBe('zh-CN')
		})
	})

	describe('deleteCookie', () => {
		it('should remove a cookie', () => {
			setCookie('locale', 'en')
			expect(getCookie('locale')).toBe('en')
			deleteCookie('locale')
			expect(getCookie('locale')).toBeNull()
		})
	})
})
