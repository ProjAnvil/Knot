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
	duplicateApi,
	updateApiOrders,
	updateGroupOrders,
	updateApiParametersFromJson,
	updateApiParametersFromStructure,
	createApiV2,
	getGroups,
	exportApis,
	importDocument
} from './api'
import type { Group, Api, Parameter, ParameterWithChildren, ApiData, GroupWithApis, ApiRawResponse, ImportPreview } from './types'

// Mock fetch globally
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

			const result = await renameGroup(999, 'New Name')

			expect(result.success).toBe(false)
		})
	})

	describe('deleteGroup', () => {
		it('should delete a group successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
		it('should forward an abort signal when one is provided', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: { id: 1, parameters: [] } })
			} as unknown as Response)

			const controller = new AbortController()
			await getApi(1, controller.signal)

			expect(mockFetch).toHaveBeenCalledWith('/api/apis/1', { signal: controller.signal })
		})

		it('should rethrow so an aborted request is not reported as a failure', async () => {
			const abortError = new DOMException('The operation was aborted.', 'AbortError')
			mockFetch.mockRejectedValueOnce(abortError)

			const controller = new AbortController()
			await expect(getApi(1, controller.signal)).rejects.toThrow('aborted')
		})

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

			const result = await deleteApi(999)

			expect(result.success).toBe(false)
		})
	})

	describe('duplicateApi', () => {
		it('should duplicate an API successfully', async () => {
			const mockDuplicatedApi: Api = {
				id: 2,
				groupId: 1,
				name: 'Login - copy',
				method: 'POST',
				endpoint: '/api/login',
				type: 'REST',
				note: null,
				order: 2,
				createdAt: '2024-01-01T00:00:00Z',
				updatedAt: '2024-01-01T00:00:00Z'
			}

			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockDuplicatedApi })
			} as unknown as Response)

			const result = await duplicateApi(1)

			expect(mockFetch).toHaveBeenCalledWith('/api/apis/1/duplicate', {
				method: 'POST'
			})
			expect(result).toEqual({
				success: true,
				data: mockDuplicatedApi
			})
		})

		it('should handle duplicate with HTTP error', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 404,
				statusText: 'Not Found',
				json: async () => ({})
			} as unknown as Response)

			const result = await duplicateApi(999)

			expect(result).toEqual({
				success: false,
				error: 'HTTP Error: 404 Not Found'
			})
		})

		it('should handle duplicate with network error', async () => {
			mockFetch.mockRejectedValueOnce(new Error('Network error'))

			const result = await duplicateApi(1)

			expect(result).toEqual({
				success: false,
				error: 'Error: Network error'
			})
		})

		it('should handle duplicate with invalid JSON response', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => {
					throw new Error('Invalid JSON')
				}
			} as unknown as Response)

			const result = await duplicateApi(1)

			expect(result).toEqual({
				success: false,
				error: 'Failed to parse response'
			})
		})
	})

	describe('updateApiOrders', () => {
		it('should update API orders successfully', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

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
			} as unknown as Response)

			// Mock update request parameters call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: { count: 2 } })
			} as unknown as Response)

			// Mock update response parameters call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: { count: 1 } })
			} as unknown as Response)

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
			} as unknown as Response)

			// Mock failed request parameters call
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({})
			} as unknown as Response)

			// Mock rollback delete call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as unknown as Response)

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
			} as unknown as Response)

			// Mock successful request parameters call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: { count: 1 } })
			} as unknown as Response)

			// Mock failed response parameters call
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({})
			} as unknown as Response)

			// Mock rollback delete call
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			} as unknown as Response)

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
			} as unknown as Response)

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

describe('API Client - Export/Import', () => {
	beforeEach(() => {
		mockFetch.mockClear()
	})

	afterEach(() => {
		mockFetch.mockReset()
	})

	describe('exportApis', () => {
		it('should post apiIds and format, returning blob and quoted filename', async () => {
			const blob = new Blob(['{"apis":[]}'], { type: 'application/json' })

			mockFetch.mockResolvedValueOnce({
				ok: true,
				headers: {
					get: (name: string) =>
						name === 'Content-Disposition' ? 'attachment; filename="knot-export-1699999999.json"' : null
				},
				blob: async () => blob
			} as unknown as Response)

			const result = await exportApis([1, 2, 3], 'json')

			expect(mockFetch).toHaveBeenCalledWith('/api/export', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ apiIds: [1, 2, 3], format: 'json' })
			})
			expect(result).toEqual({
				blob,
				filename: 'knot-export-1699999999.json'
			})
		})

		it('should parse an unquoted filename from Content-Disposition', async () => {
			const blob = new Blob(['<html></html>'], { type: 'text/html' })

			mockFetch.mockResolvedValueOnce({
				ok: true,
				headers: {
					get: (name: string) =>
						name === 'Content-Disposition' ? 'attachment; filename=api-docs.html' : null
				},
				blob: async () => blob
			} as unknown as Response)

			const result = await exportApis([1], 'html')

			expect(result).toEqual({
				blob,
				filename: 'api-docs.html'
			})
		})

		it('should return an error object on non-OK response', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 500,
				statusText: 'Internal Server Error'
			} as unknown as Response)

			const result = await exportApis([1, 2], 'openapi')

			expect(result).toEqual({
				error: 'HTTP Error: 500 Internal Server Error'
			})
		})

		it('should return an error object on network error', async () => {
			mockFetch.mockRejectedValueOnce(new Error('Network error'))

			const result = await exportApis([1], 'postman')

			expect(result).toEqual({
				error: 'Error: Network error'
			})
		})
	})

	describe('importDocument', () => {
		const mockFile = new File(['{"apis":[]}'], 'export.json', { type: 'application/json' })

		const mockPreview: ImportPreview = {
			format: 'json',
			groupsCreated: 1,
			groupsMerged: 0,
			apisImported: 2,
			parametersImported: 5,
			warnings: []
		}

		it('should post FormData with file and dryRun, returning parsed preview', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockPreview })
			} as unknown as Response)

			const result = await importDocument(mockFile, true)

			expect(mockFetch).toHaveBeenCalledTimes(1)
			const [url, init] = mockFetch.mock.calls[0]
			expect(url).toBe('/api/import')
			expect(init.method).toBe('POST')
			expect(init.body).toBeInstanceOf(FormData)
			const formData = init.body as FormData
			expect(formData.get('file')).toBe(mockFile)
			expect(formData.get('dryRun')).toBe('true')
			expect(result).toEqual({
				success: true,
				data: mockPreview
			})
		})

		it('should send dryRun as false string when not a dry run', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true, data: mockPreview })
			} as unknown as Response)

			await importDocument(mockFile, false)

			const init = mockFetch.mock.calls[0][1]
			const formData = init.body as FormData
			expect(formData.get('dryRun')).toBe('false')
		})

		it('should return the error envelope on import failure', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: false, error: 'Unsupported format' })
			} as unknown as Response)

			const result = await importDocument(mockFile, false)

			expect(result).toEqual({
				success: false,
				error: 'Unsupported format'
			})
		})

		it('should handle HTTP error response', async () => {
			mockFetch.mockResolvedValueOnce({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({})
			} as unknown as Response)

			const result = await importDocument(mockFile, true)

			expect(result).toEqual({
				success: false,
				error: 'HTTP Error: 400 Bad Request'
			})
		})
	})
})
