/**
 * Tests to verify i18n locale files are complete and consistent
 */
import { describe, it, expect } from 'vitest'
import en from './locales/en.json'
import zhCN from './locales/zh-CN.json'

function getAllKeys(obj: Record<string, unknown>, prefix = ''): string[] {
	const keys: string[] = []
	for (const key of Object.keys(obj)) {
		const fullKey = prefix ? `${prefix}.${key}` : key
		if (typeof obj[key] === 'object' && obj[key] !== null && !Array.isArray(obj[key])) {
			keys.push(...getAllKeys(obj[key] as Record<string, unknown>, fullKey))
		} else {
			keys.push(fullKey)
		}
	}
	return keys
}

describe('i18n locale files', () => {
	const enKeys = getAllKeys(en)
	const zhKeys = getAllKeys(zhCN)

	it('should have matching keys between en and zh-CN', () => {
		const missingInZh = enKeys.filter(key => !zhKeys.includes(key))
		const missingInEn = zhKeys.filter(key => !enKeys.includes(key))

		expect(missingInZh).toEqual([])
		expect(missingInEn).toEqual([])
	})

	it('should not have empty values in en locale', () => {
		const emptyKeys = enKeys.filter(key => {
			const value = getNestedValue(en, key)
			return typeof value === 'string' && value.trim() === ''
		})
		expect(emptyKeys).toEqual([])
	})

	it('should not have empty values in zh-CN locale', () => {
		const emptyKeys = zhKeys.filter(key => {
			const value = getNestedValue(zhCN, key)
			return typeof value === 'string' && value.trim() === ''
		})
		expect(emptyKeys).toEqual([])
	})

	it('should have required common keys', () => {
		const requiredKeys = [
			'common.cancel',
			'common.save',
			'common.edit',
			'common.delete',
			'common.loading',
			'common.create',
			'common.creating',
		]
		for (const key of requiredKeys) {
			expect(enKeys).toContain(key)
			expect(zhKeys).toContain(key)
		}
	})

	it('should have required createApi dialog keys', () => {
		const requiredKeys = [
			'createApi.title',
			'createApi.apiNameLabel',
			'createApi.apiNamePlaceholder',
			'createApi.endpointLabel',
			'createApi.endpointPlaceholder',
			'createApi.typeLabel',
			'createApi.methodLabel',
			'createApi.requestParams',
			'createApi.responseParams',
			'createApi.addParameter',
			'createApi.noRequestParams',
			'createApi.noResponseParams',
			'createApi.paramNamePlaceholder',
			'createApi.paramDescPlaceholder',
			'createApi.nameRequired',
			'createApi.groupRequired',
			'createApi.paramNamesRequired',
			'createApi.newApi',
		]
		for (const key of requiredKeys) {
			expect(enKeys).toContain(key)
			expect(zhKeys).toContain(key)
		}
	})

	it('should have required json keys including validJson', () => {
		expect(enKeys).toContain('json.valid')
		expect(zhKeys).toContain('json.valid')
	})

	it('should have selectApiPrompt key', () => {
		expect(enKeys).toContain('common.selectApiPrompt')
		expect(zhKeys).toContain('common.selectApiPrompt')
	})
})

function getNestedValue(obj: Record<string, unknown>, path: string): unknown {
	return path.split('.').reduce((curr: unknown, key: string) => {
		if (curr && typeof curr === 'object') {
			return (curr as Record<string, unknown>)[key]
		}
		return undefined
	}, obj)
}
