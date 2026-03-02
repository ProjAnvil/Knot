import { describe, it, expect } from 'vitest'
import {
	performSearch,
	getExpandedGroups,
	navigateNext,
	navigatePrevious,
	isCurrentGroupResult,
	isCurrentApiResult,
	type SearchResult
} from './search'
import type { GroupWithApis } from './types'

// Mock data for testing
const mockGroups: GroupWithApis[] = [
	{
		id: 1,
		name: 'User API',
		createdAt: '2024-01-01T00:00:00Z',
		updatedAt: '2024-01-01T00:00:00Z',
		apis: [
			{
				id: 1,
				groupId: 1,
				name: 'Get User',
				method: 'GET',
				endpoint: '/users/:id',
				type: 'REST',
				note: null,
				order: 0,
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-01T00:00:00Z'
			},
			{
				id: 2,
				groupId: 1,
				name: 'Create User',
				method: 'POST',
				endpoint: '/users',
				type: 'REST',
				note: null,
				order: 1,
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-01T00:00:00Z'
			}
		]
	},
	{
		id: 2,
		name: 'Product API',
		createdAt: '2024-01-01T00:00:00Z',
		updatedAt: '2024-01-01T00:00:00Z',
		apis: [
			{
				id: 3,
				groupId: 2,
				name: 'Get Product',
				method: 'GET',
				endpoint: '/products/:id',
				type: 'REST',
				note: null,
				order: 0,
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-01T00:00:00Z'
			}
		]
	},
	{
		id: 3,
		name: 'Order Service',
		createdAt: '2024-01-01T00:00:00Z',
		updatedAt: '2024-01-01T00:00:00Z',
		apis: []
	}
]

describe('performSearch', () => {
	describe('empty query handling', () => {
		it('should return empty state for empty query', () => {
			const result = performSearch('', mockGroups)

			expect(result.matchedGroupNames.size).toBe(0)
			expect(result.matchedApiIds.size).toBe(0)
			expect(result.results.length).toBe(0)
		})

		it('should return empty state for whitespace-only query', () => {
			const result = performSearch('   ', mockGroups)

			expect(result.matchedGroupNames.size).toBe(0)
			expect(result.matchedApiIds.size).toBe(0)
			expect(result.results.length).toBe(0)
		})
	})

	describe('group name matching', () => {
		it('should match group names case-insensitively', () => {
			const result = performSearch('USER', mockGroups)

			expect(result.matchedGroupNames.has('User API')).toBe(true)
			expect(result.results.some(r => r.type === 'group' && r.id === 1)).toBe(true)
		})

		it('should match partial group names', () => {
			const result = performSearch('prod', mockGroups)

			expect(result.matchedGroupNames.has('Product API')).toBe(true)
			// Also matches "Get Product" API which contains "prod"
			expect(result.results.length).toBe(2)
		})

		it('should match multiple groups', () => {
			const result = performSearch('api', mockGroups)

			expect(result.matchedGroupNames.size).toBe(2)
			expect(result.matchedGroupNames.has('User API')).toBe(true)
			expect(result.matchedGroupNames.has('Product API')).toBe(true)
		})
	})

	describe('API name matching', () => {
		it('should match API names case-insensitively', () => {
			const result = performSearch('GET', mockGroups)

			expect(result.matchedApiIds.has(1)).toBe(true) // Get User
			expect(result.matchedApiIds.has(3)).toBe(true) // Get Product
		})

		it('should match partial API names', () => {
			const result = performSearch('creat', mockGroups)

			expect(result.matchedApiIds.has(2)).toBe(true) // Create User
		})

		it('should match both groups and APIs', () => {
			const result = performSearch('user', mockGroups)

			expect(result.matchedGroupNames.has('User API')).toBe(true)
			expect(result.matchedApiIds.has(1)).toBe(true) // Get User
			expect(result.matchedApiIds.has(2)).toBe(true) // Create User
		})
	})

	describe('no results', () => {
		it('should return empty state when no matches found', () => {
			const result = performSearch('nonexistent', mockGroups)

			expect(result.matchedGroupNames.size).toBe(0)
			expect(result.matchedApiIds.size).toBe(0)
			expect(result.results.length).toBe(0)
		})
	})

	describe('results ordering', () => {
		it('should include correct groupId for API results', () => {
			const result = performSearch('Get User', mockGroups)

			expect(result.results.length).toBe(1)
			expect(result.results[0]).toEqual({
				type: 'api',
				id: 1,
				groupId: 1,
				name: 'Get User'
			})
		})

		it('should include correct groupId for group results', () => {
			const result = performSearch('Order Service', mockGroups)

			expect(result.results.length).toBe(1)
			expect(result.results[0]).toEqual({
				type: 'group',
				id: 3,
				groupId: 3,
				name: 'Order Service'
			})
		})
	})

	describe('edge cases', () => {
		it('should handle groups with no APIs', () => {
			const result = performSearch('Order', mockGroups)

			expect(result.matchedGroupNames.has('Order Service')).toBe(true)
			expect(result.results.length).toBe(1)
		})

		it('should handle empty groups array', () => {
			const result = performSearch('test', [])

			expect(result.matchedGroupNames.size).toBe(0)
			expect(result.matchedApiIds.size).toBe(0)
			expect(result.results.length).toBe(0)
		})

		it('should handle special characters in query', () => {
			const result = performSearch('/users', mockGroups)

			// Special characters should be treated as literal characters
			expect(result.results.length).toBe(0)
		})

		it('should handle Unicode characters', () => {
			const groupsWithUnicode: GroupWithApis[] = [
				{
					id: 1,
					name: '用户接口',
					createdAt: '2024-01-01T00:00:00Z',
					updatedAt: '2024-01-01T00:00:00Z',
					apis: []
				}
			]

			const result = performSearch('用户', groupsWithUnicode)

			expect(result.matchedGroupNames.has('用户接口')).toBe(true)
		})
	})
})

describe('getExpandedGroups', () => {
	it('should return empty set when no results', () => {
		const searchState = {
			matchedGroupNames: new Set<string>(),
			matchedApiIds: new Set<number>(),
			results: []
		}

		const expanded = getExpandedGroups(searchState, mockGroups)

		expect(expanded.size).toBe(0)
	})

	it('should return groups containing matched APIs', () => {
		const searchState = performSearch('Get User', mockGroups)

		const expanded = getExpandedGroups(searchState, mockGroups)

		expect(expanded.has('User API')).toBe(true)
	})

	it('should return matched groups', () => {
		const searchState = performSearch('Product API', mockGroups)

		const expanded = getExpandedGroups(searchState, mockGroups)

		expect(expanded.has('Product API')).toBe(true)
	})

	it('should return multiple groups', () => {
		const searchState = performSearch('Get', mockGroups)

		const expanded = getExpandedGroups(searchState, mockGroups)

		expect(expanded.has('User API')).toBe(true)
		expect(expanded.has('Product API')).toBe(true)
	})
})

describe('navigateNext', () => {
	it('should return -1 when no results', () => {
		expect(navigateNext(0, 0)).toBe(-1)
		expect(navigateNext(-1, 0)).toBe(-1)
	})

	it('should return next index', () => {
		expect(navigateNext(0, 5)).toBe(1)
		expect(navigateNext(2, 5)).toBe(3)
	})

	it('should wrap around to beginning', () => {
		expect(navigateNext(4, 5)).toBe(0)
	})

	it('should handle single result', () => {
		expect(navigateNext(0, 1)).toBe(0)
	})
})

describe('navigatePrevious', () => {
	it('should return -1 when no results', () => {
		expect(navigatePrevious(0, 0)).toBe(-1)
		expect(navigatePrevious(-1, 0)).toBe(-1)
	})

	it('should return previous index', () => {
		expect(navigatePrevious(3, 5)).toBe(2)
		expect(navigatePrevious(2, 5)).toBe(1)
	})

	it('should wrap around to end', () => {
		expect(navigatePrevious(0, 5)).toBe(4)
	})

	it('should handle single result', () => {
		expect(navigatePrevious(0, 1)).toBe(0)
	})
})

describe('isCurrentGroupResult', () => {
	const results: SearchResult[] = [
		{ type: 'group', id: 1, groupId: 1, name: 'Group 1' },
		{ type: 'api', id: 2, groupId: 1, name: 'API 1' },
		{ type: 'group', id: 3, groupId: 3, name: 'Group 3' }
	]

	it('should return true for current group result', () => {
		expect(isCurrentGroupResult(results, 0, 1)).toBe(true)
		expect(isCurrentGroupResult(results, 2, 3)).toBe(true)
	})

	it('should return false for non-current group', () => {
		expect(isCurrentGroupResult(results, 0, 2)).toBe(false)
	})

	it('should return false for API result', () => {
		expect(isCurrentGroupResult(results, 1, 1)).toBe(false)
	})

	it('should return false for invalid index', () => {
		expect(isCurrentGroupResult(results, -1, 1)).toBe(false)
		expect(isCurrentGroupResult(results, 10, 1)).toBe(false)
	})

	it('should return false for empty results', () => {
		expect(isCurrentGroupResult([], 0, 1)).toBe(false)
	})
})

describe('isCurrentApiResult', () => {
	const results: SearchResult[] = [
		{ type: 'group', id: 1, groupId: 1, name: 'Group 1' },
		{ type: 'api', id: 2, groupId: 1, name: 'API 1' },
		{ type: 'api', id: 3, groupId: 1, name: 'API 2' }
	]

	it('should return true for current API result', () => {
		expect(isCurrentApiResult(results, 1, 2)).toBe(true)
		expect(isCurrentApiResult(results, 2, 3)).toBe(true)
	})

	it('should return false for non-current API', () => {
		expect(isCurrentApiResult(results, 1, 3)).toBe(false)
	})

	it('should return false for group result', () => {
		expect(isCurrentApiResult(results, 0, 1)).toBe(false)
	})

	it('should return false for invalid index', () => {
		expect(isCurrentApiResult(results, -1, 2)).toBe(false)
		expect(isCurrentApiResult(results, 10, 2)).toBe(false)
	})

	it('should return false for empty results', () => {
		expect(isCurrentApiResult([], 0, 2)).toBe(false)
	})
})
