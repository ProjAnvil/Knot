import type { GroupWithApis } from './types'

export interface SearchResult {
	type: 'group' | 'api'
	id: number
	groupId: number
	name: string
}

export interface SearchState {
	matchedGroupNames: Set<string>
	matchedApiIds: Set<number>
	results: SearchResult[]
}

/**
 * Perform fuzzy search on groups and APIs
 * @param query - The search query string
 * @param groups - The groups to search in
 * @returns Search state containing matched groups, APIs, and results list
 */
export function performSearch(query: string, groups: GroupWithApis[]): SearchState {
	const trimmedQuery = query.trim().toLowerCase()

	if (!trimmedQuery) {
		return {
			matchedGroupNames: new Set(),
			matchedApiIds: new Set(),
			results: []
		}
	}

	const matchedGroupNames = new Set<string>()
	const matchedApiIds = new Set<number>()
	const results: SearchResult[] = []

	for (const group of groups) {
		// Check if group name matches
		if (group.name.toLowerCase().includes(trimmedQuery)) {
			matchedGroupNames.add(group.name)
			results.push({ type: 'group', id: group.id, groupId: group.id, name: group.name })
		}

		// Check if any API name matches
		for (const api of group.apis) {
			if (api.name.toLowerCase().includes(trimmedQuery)) {
				matchedApiIds.add(api.id)
				results.push({ type: 'api', id: api.id, groupId: group.id, name: api.name })
			}
		}
	}

	return {
		matchedGroupNames,
		matchedApiIds,
		results
	}
}

/**
 * Get the groups that should be expanded based on search results
 * @param searchState - The search state
 * @param groups - All groups
 * @returns Set of group names that should be expanded
 */
export function getExpandedGroups(searchState: SearchState, groups: GroupWithApis[]): Set<string> {
	if (searchState.results.length === 0) {
		return new Set()
	}

	const expandedGroups = new Set<string>()

	for (const result of searchState.results) {
		const group = groups.find(g => g.id === result.groupId)
		if (group) {
			expandedGroups.add(group.name)
		}
	}

	return expandedGroups
}

/**
 * Navigate to the next search result
 * @param currentIndex - Current result index
 * @param totalResults - Total number of results
 * @returns New result index
 */
export function navigateNext(currentIndex: number, totalResults: number): number {
	if (totalResults === 0) return -1
	return currentIndex >= totalResults - 1 ? 0 : currentIndex + 1
}

/**
 * Navigate to the previous search result
 * @param currentIndex - Current result index
 * @param totalResults - Total number of results
 * @returns New result index
 */
export function navigatePrevious(currentIndex: number, totalResults: number): number {
	if (totalResults === 0) return -1
	return currentIndex <= 0 ? totalResults - 1 : currentIndex - 1
}

/**
 * Check if a group is the current focused result
 * @param results - Search results
 * @param currentIndex - Current result index
 * @param groupId - Group ID to check
 * @returns True if this group is the current focus
 */
export function isCurrentGroupResult(
	results: SearchResult[],
	currentIndex: number,
	groupId: number
): boolean {
	if (currentIndex < 0 || currentIndex >= results.length) return false
	const result = results[currentIndex]
	return result.type === 'group' && result.id === groupId
}

/**
 * Check if an API is the current focused result
 * @param results - Search results
 * @param currentIndex - Current result index
 * @param apiId - API ID to check
 * @returns True if this API is the current focus
 */
export function isCurrentApiResult(
	results: SearchResult[],
	currentIndex: number,
	apiId: number
): boolean {
	if (currentIndex < 0 || currentIndex >= results.length) return false
	const result = results[currentIndex]
	return result.type === 'api' && result.id === apiId
}
