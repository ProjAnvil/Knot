// Unit tests for API client functions
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
	getGroupsWithApis,
	createGroup,
	renameGroup,
	deleteGroup,
	getApi,
	createApi,
	updateApi,
	deleteApi,
	updateApiOrders,
	updateGroupOrders,
	updateApiParametersFromJson,
	updateApiParametersFromStructure,
	createApiV2,
	getGroups
} from './api'
import type { Group, Api, Parameter, ParameterWithChildren, ApiData, GroupWithApis, ApiRawResponse } from './types'

// Mock fetch globally
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('API Client - Groups', () => {
	beforeEach(() => {
		mockFetch.mockClear()
	})

	afterEach(() => {
		mockFetch.mockReset()
	})

	describe('getGroupsWithApis', () => {
		it('should fetch groups with APIs successfully', async () => {
			const mockGroups: GroupWithApis[] = [
				{
					id: 1,
					name: 'User Management',
					createdAt: '2024-01-01T00:00:00Z',
					updatedAt: '2024-01-01T00:00:00Z',
					apis: [
						{
							id: 1,
							groupId: 1,
							name: 'Login',
							method: 'POST',
							endpoint: '/api/login',
							type: 'REST',
							note: null,
							order: 1,
							createdAt: '2024-01-01T00:00:00Z',
							updatedAt: '2024-01-01T00:00:00Z'
						}
					]
				}
			]

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockGroups })
			} as Response)

			const result = await getGroupsWithApis()

			expect(mockFetch).toHaveBeenCalledWith('/api/groups/with-apis')
			expect(result).toEqual({
				success: true,
				data: mockGroups
			})
		})

		it('should handle HTTP error response', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 500,
				statusText: 'Internal Server Error',
				json: async () => ({})
			} as Response)

			const result = await getGroupsWithApis()

			expect(result).toEqual({
				success: false,
				error: 'HTTP Error: 500 Internal Server Error'
			})
		})

		it('should handle network error', async () => {
			mockFetch.mockRejectedValueOnce(new Error('Network error'))

			const result = await getGroupsWithApis()

			expect(result).toEqual({
				success: false,
				error: 'Error: Network error'
			})
		})

		it('should handle invalid JSON response', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => {
					throw new Error('Invalid JSON')
				}
			} as Response)

			const result = await getGroupsWithApis()

			expect(result).toEqual({
				success: false,
				error: 'Failed to parse response'
			})
		})
	})

	describe('createGroup', () => {
		it('should create a new group successfully', async () => {
			const mockGroup: Group = {
				id: 1,
				name: 'New Group',
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-01T00:00:00Z'
			}

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockGroup })
			} as Response)

			const result = await createGroup('New Group')

			expect(mockFetch).toHaveBeenCalledWith('/api/groups', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name: 'New Group' })
			})
			expect(result).toEqual({
				success: true,
				data: mockGroup
			})
		})

		it('should handle group creation error', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({})
			} as Response)

			const result = await createGroup('')

			expect(result.success).toBe(false)
			expect(result.error).toContain('HTTP Error')
		})
	})

	describe('renameGroup', () => {
		it('should rename a group successfully', async () => {
			const mockGroup: Group = {
				id: 1,
				name: 'Updated Group Name',
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-02T00:00:00Z'
			}

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockGroup })
			} as Response)

			const result = await renameGroup(1, 'Updated Group Name')

			expect(mockFetch).toHaveBeenCalledWith('/api/groups/1', {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name: 'Updated Group Name' })
			})
			expect(result).toEqual({
				success: true,
				data: mockGroup
			})
		})

		it('should handle group rename error', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 404,
				statusText: 'Not Found',
				json: async () => ({})
			} as Response)

			const result = await renameGroup(999, 'New Name')

			expect(result.success).toBe(false)
		})
	})

	describe('deleteGroup', () => {
		it('should delete a group successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as Response)

			const result = await deleteGroup(1)

			expect(mockFetch).toHaveBeenCalledWith('/api/groups/1', {
				method: 'DELETE'
			})
			expect(result).toEqual({ success: true })
		})

		it('should handle group deletion error', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 404,
				statusText: 'Not Found',
				json: async () => ({})
			} as Response)

			const result = await deleteGroup(999)

			expect(result.success).toBe(false)
		})
	})

	describe('getGroups', () => {
		it('should fetch simple groups list successfully', async () => {
			const mockGroups = [
				{ id: 1, name: 'Group 1' },
				{ id: 2, name: 'Group 2' }
			]

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockGroups })
			} as Response)

			const result = await getGroups()

			expect(mockFetch).toHaveBeenCalledWith('/api/groups')
			expect(result).toEqual({
				success: true,
				data: mockGroups
			})
		})
	})

	describe('updateGroupOrders', () => {
		it('should update group orders successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as Response)

			const orders = [
				{ id: 1, order: 0 },
				{ id: 2, order: 1 }
			]

			const result = await updateGroupOrders(orders)

			expect(mockFetch).toHaveBeenCalledWith('/api/groups/orders', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ groupOrders: orders })
			})
			expect(result).toEqual({ success: true })
		})

		it('should handle order update error', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({})
			} as Response)

			const result = await updateGroupOrders([{ id: 1, order: 0 }])

			expect(result.success).toBe(false)
		})
	})
})

describe('API Client - APIs', () => {
	beforeEach(() => {
		mockFetch.mockClear()
	})

	afterEach(() => {
		mockFetch.mockReset()
	})

	describe('getApi', () => {
		it('should fetch API with hierarchical parameters', async () => {
			const mockRawResponse: ApiRawResponse = {
				id: 1,
				groupId: 1,
				name: 'Login API',
				method: 'POST',
				endpoint: '/api/login',
				type: 'REST',
				note: null,
				order: 1,
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-01T00:00:00Z',
				parameters: [
					{
						id: 1,
						apiId: 1,
						name: 'username',
						type: 'string',
						required: true,
						description: 'User name',
						paramType: 'request',
						parentId: null,
						order: 1
					},
					{
						id: 2,
						apiId: 1,
						name: 'password',
						type: 'string',
						required: true,
						description: 'User password',
						paramType: 'request',
						parentId: null,
						order: 2
					},
					{
						id: 3,
						apiId: 1,
						name: 'token',
						type: 'string',
						required: true,
						description: 'Auth token',
						paramType: 'response',
						parentId: null,
						order: 1
					}
				]
			}

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockRawResponse })
			} as Response)

			const result = await getApi(1)

			expect(mockFetch).toHaveBeenCalledWith('/api/apis/1')
			expect(result.success).toBe(true)
			if (result.data) {
				expect(result.data.requestParameters).toHaveLength(2)
				expect(result.data.responseParameters).toHaveLength(1)
				expect(result.data.requestParameters[0].name).toBe('username')
			}
		})

		it('should handle API not found', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 404,
				statusText: 'Not Found',
				json: async () => ({})
			} as Response)

			const result = await getApi(999)

			expect(result.success).toBe(false)
		})

		it('should build nested parameter tree correctly', async () => {
			const mockRawResponse: ApiRawResponse = {
				id: 1,
				groupId: 1,
				name: 'Complex API',
				method: 'POST',
				endpoint: '/api/complex',
				type: 'REST',
				note: null,
				order: 1,
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-01T00:00:00Z',
				parameters: [
					{
						id: 1,
						apiId: 1,
						name: 'user',
						type: 'object',
						required: true,
						description: null,
						paramType: 'request',
						parentId: null,
						order: 1
					},
					{
						id: 2,
						apiId: 1,
						name: 'name',
						type: 'string',
						required: true,
						description: null,
						paramType: 'request',
						parentId: 1,
						order: 1
					},
					{
						id: 3,
						apiId: 1,
						name: 'email',
						type: 'string',
						required: true,
						description: null,
						paramType: 'request',
						parentId: 1,
						order: 2
					}
				]
			}

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockRawResponse })
			} as Response)

			const result = await getApi(1)

			expect(result.success).toBe(true)
			if (result.data) {
				expect(result.data.requestParameters).toHaveLength(1)
				expect(result.data.requestParameters[0].name).toBe('user')
				expect(result.data.requestParameters[0].children).toHaveLength(2)
				expect(result.data.requestParameters[0].children[0].name).toBe('name')
				expect(result.data.requestParameters[0].children[1].name).toBe('email')
			}
		})
	})

	describe('createApi', () => {
		it('should create a new API successfully', async () => {
			const mockApi: Api = {
				id: 1,
				groupId: 1,
				name: 'New API',
				method: 'GET',
				endpoint: '/api/new',
				type: 'REST',
				note: null,
				order: 1,
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-01T00:00:00Z'
			}

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockApi })
			} as Response)

			const result = await createApi({
				name: 'New API',
				groupId: 1,
				method: 'GET',
				endpoint: '/api/new',
				type: 'REST'
			})

			expect(mockFetch).toHaveBeenCalledWith('/api/apis', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: 'New API',
					groupId: 1,
					method: 'GET',
					endpoint: '/api/new',
					type: 'REST'
				})
			})
			expect(result).toEqual({
				success: true,
				data: mockApi
			})
		})
	})

	describe('updateApi', () => {
		it('should update an API successfully', async () => {
			const mockApi: Api = {
				id: 1,
				groupId: 1,
				name: 'Updated API',
				method: 'POST',
				endpoint: '/api/updated',
				type: 'REST',
				note: 'Updated note',
				order: 1,
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-02T00:00:00Z'
			}

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockApi })
			} as Response)

			const result = await updateApi(1, {
				name: 'Updated API',
				endpoint: '/api/updated',
				note: 'Updated note'
			})

			expect(mockFetch).toHaveBeenCalledWith('/api/apis/1', {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: 'Updated API',
					endpoint: '/api/updated',
					note: 'Updated note'
				})
			})
			expect(result).toEqual({
				success: true,
				data: mockApi
			})
		})
	})

	describe('deleteApi', () => {
		it('should delete an API successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as Response)

			const result = await deleteApi(1)

			expect(mockFetch).toHaveBeenCalledWith('/api/apis/1', {
				method: 'DELETE'
			})
			expect(result).toEqual({ success: true })
		})

		it('should handle API deletion error', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 404,
				statusText: 'Not Found',
				json: async () => ({})
			} as Response)

			const result = await deleteApi(999)

			expect(result.success).toBe(false)
		})
	})

	describe('updateApiOrders', () => {
		it('should update API orders successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as Response)

			const orders = [
				{ id: 1, order: 0 },
				{ id: 2, order: 1 }
			]

			const result = await updateApiOrders(orders)

			expect(mockFetch).toHaveBeenCalledWith('/api/apis/orders', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ apiOrders: orders })
			})
			expect(result).toEqual({ success: true })
		})
	})

	describe('updateApiParametersFromJson', () => {
		it('should update parameters from JSON successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as Response)

			const jsonData = { username: 'string', age: 'number' }

			const result = await updateApiParametersFromJson({
				apiId: 1,
				paramType: 'request',
				json: jsonData
			})

			expect(mockFetch).toHaveBeenCalledWith('/api/apis/1/parameters/from-json', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					paramType: 'request',
					json: jsonData
				})
			})
			expect(result).toEqual({ success: true })
		})
	})

	describe('updateApiParametersFromStructure', () => {
		it('should update parameters from structure successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: { count: 2 } })
			} as Response)

			const parameters: ParameterWithChildren[] = [
				{
					id: 1,
					apiId: 1,
					name: 'user',
					type: 'object',
					required: true,
					description: null,
					paramType: 'request',
					parentId: null,
					order: 1,
					children: []
				}
			]

			const result = await updateApiParametersFromStructure({
				apiId: 1,
				paramType: 'request',
				parameters
			})

			expect(mockFetch).toHaveBeenCalledWith('/api/apis/1/parameters', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					paramType: 'request',
					parameters
				})
			})
			expect(result).toEqual({
				success: true,
				data: { count: 2 }
			})
		})
	})

	describe('createApiV2', () => {
		it('should create API with parameters successfully', async () => {
			// Mock create API call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({
					success: true,
					data: {
						id: 1,
						groupId: 1,
						name: 'New API',
						method: 'POST',
						endpoint: '/api/new',
						type: 'REST',
						note: null,
						order: 1,
						createdAt: '2024-01-01T00:00:00Z',
						updatedAt: '2024-01-01T00:00:00Z'
					}
				})
			} as Response)

			// Mock update request parameters call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: { count: 2 } })
			} as Response)

			// Mock update response parameters call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: { count: 1 } })
			} as Response)

			const result = await createApiV2({
				groupId: 1,
				name: 'New API',
				endpoint: '/api/new',
				method: 'POST',
				type: 'REST',
				requestParameters: [
					{ name: 'username', type: 'string', required: true },
					{ name: 'password', type: 'string', required: true }
				],
				responseParameters: [
					{ name: 'token', type: 'string', required: true }
				]
			})

			expect(result.success).toBe(true)
			if (result.data) {
				expect(result.data.id).toBe(1)
				expect(result.data.requestParameterCount).toBe(2)
				expect(result.data.responseParameterCount).toBe(1)
			}
		})

		it('should rollback on request parameter creation failure', async () => {
			// Mock create API call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({
					success: true,
					data: {
						id: 1,
						groupId: 1,
						name: 'New API',
						method: 'POST',
						endpoint: '/api/new',
						type: 'REST',
						note: null,
						order: 1,
						createdAt: '2024-01-01T00:00:00Z',
						updatedAt: '2024-01-01T00:00:00Z'
					}
				})
			} as Response)

			// Mock failed request parameters call
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({})
			} as Response)

			// Mock rollback delete call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as Response)

			const result = await createApiV2({
				groupId: 1,
				name: 'New API',
				endpoint: '/api/new',
				method: 'POST',
				type: 'REST',
				requestParameters: [
					{ name: 'username', type: 'string', required: true }
				]
			})

			expect(result.success).toBe(false)
			expect(result.error).toBe('Failed to create request parameters')
			// Verify delete was called for rollback
			expect(mockFetch).toHaveBeenCalledTimes(3)
		})

		it('should rollback on response parameter creation failure', async () => {
			// Mock create API call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({
					success: true,
					data: {
						id: 1,
						groupId: 1,
						name: 'New API',
						method: 'POST',
						endpoint: '/api/new',
						type: 'REST',
						note: null,
						order: 1,
						createdAt: '2024-01-01T00:00:00Z',
						updatedAt: '2024-01-01T00:00:00Z'
					}
				})
			} as Response)

			// Mock successful request parameters call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: { count: 1 } })
			} as Response)

			// Mock failed response parameters call
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({})
			} as Response)

			// Mock rollback delete call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as Response)

			const result = await createApiV2({
				groupId: 1,
				name: 'New API',
				endpoint: '/api/new',
				method: 'POST',
				type: 'REST',
				requestParameters: [
					{ name: 'username', type: 'string', required: true }
				],
				responseParameters: [
					{ name: 'token', type: 'string', required: true }
				]
			})

			expect(result.success).toBe(false)
			expect(result.error).toBe('Failed to create response parameters')
		})

		it('should handle API creation failure', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({})
			} as Response)

			const result = await createApiV2({
				groupId: 1,
				name: 'New API',
				endpoint: '/api/new',
				method: 'POST',
				type: 'REST'
			})

			expect(result.success).toBe(false)
		})
	})
})
