// API service functions for communicating with backend

import type { Api, ApiData, ApiRawResponse, ApiResult, ExportFormat, Group, GroupWithApis, ImportPreview, Parameter, ParameterWithChildren } from './types'

const API_BASE = '/api'

async function handleResponse<T>(response: Response): Promise<ApiResult<T>> {
	if (!response.ok) {
		return {
			success: false,
			error: `HTTP Error: ${response.status} ${response.statusText}`,
		}
	}

	try {
		const data = await response.json()
		return data
	} catch (error) {
		return {
			success: false,
			error: 'Failed to parse response',
		}
	}
}

// Groups API
export async function getGroupsWithApis(): Promise<ApiResult<GroupWithApis[]>> {
	try {
		const response = await fetch(`${API_BASE}/groups/with-apis`)
		return await handleResponse<GroupWithApis[]>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function createGroup(name: string): Promise<ApiResult<Group>> {
	try {
		const response = await fetch(`${API_BASE}/groups`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name }),
		})
		return await handleResponse<Group>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function renameGroup(id: number, name: string): Promise<ApiResult<Group>> {
	try {
		const response = await fetch(`${API_BASE}/groups/${id}`, {
			method: 'PATCH',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name }),
		})
		return await handleResponse<Group>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function deleteGroup(id: number): Promise<ApiResult<void>> {
	try {
		const response = await fetch(`${API_BASE}/groups/${id}`, {
			method: 'DELETE',
		})
		return await handleResponse<void>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

// Build hierarchical structure for nested parameters
function buildParameterTree(params: Parameter[]): ParameterWithChildren[] {
	const map = new Map<number, ParameterWithChildren>()
	const roots: ParameterWithChildren[] = []

	// First pass: create all nodes with empty children
	params.forEach((param) => {
		map.set(param.id, { ...param, children: [] })
	})

	// Second pass: build tree structure
	params.forEach((param) => {
		const node = map.get(param.id)
		if (node) {
			if (param.parentId === null) {
				roots.push(node)
			} else {
				const parent = map.get(param.parentId)
				if (parent) {
					parent.children.push(node)
				}
			}
		}
	})

	return roots
}

// APIs API
export async function getApi(id: number, signal?: AbortSignal): Promise<ApiResult<ApiData>> {
	try {
		// Keep the single-argument form when no signal is passed so callers and
		// tests that assert on the fetch call are unaffected.
		const response = signal
			? await fetch(`${API_BASE}/apis/${id}`, { signal })
			: await fetch(`${API_BASE}/apis/${id}`)
		const result = await handleResponse<ApiRawResponse>(response)

		if (result.success && result.data) {
			// Transform parameters array into requestParameters and responseParameters
			const parameters = result.data.parameters || []
			const requestParameters = parameters.filter((p) => p.paramType === 'request')
			const responseParameters = parameters.filter((p) => p.paramType === 'response')

			const apiData: ApiData = {
				...result.data,
				method: result.data.method || '',
				requestParameters: buildParameterTree(requestParameters),
				responseParameters: buildParameterTree(responseParameters),
			}

			return { success: true, data: apiData }
		}

		return { success: false, error: result.error }
	} catch (error) {
		// Let callers distinguish an intentional cancellation from a real failure
		// so a superseded request never surfaces as an error.
		if (signal?.aborted || (error instanceof DOMException && error.name === 'AbortError')) {
			throw error
		}
		return { success: false, error: String(error) }
	}
}

export async function createApi(data: {
	name: string
	groupId: number
	method: string
	endpoint: string
	type: string
}): Promise<ApiResult<Api>> {
	try {
		const response = await fetch(`${API_BASE}/apis`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data),
		})
		return await handleResponse<Api>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function updateApi(id: number, data: Partial<Api>): Promise<ApiResult<Api>> {
	try {
		const response = await fetch(`${API_BASE}/apis/${id}`, {
			method: 'PATCH',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data),
		})
		return await handleResponse<Api>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function duplicateApi(id: number): Promise<ApiResult<Api>> {
	try {
		const response = await fetch(`${API_BASE}/apis/${id}/duplicate`, {
			method: 'POST',
		})
		return await handleResponse<Api>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function deleteApi(id: number): Promise<ApiResult<void>> {
	try {
		const response = await fetch(`${API_BASE}/apis/${id}`, {
			method: 'DELETE',
		})
		return await handleResponse<void>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function updateApiOrders(orders: { id: number; order: number }[]): Promise<ApiResult<void>> {
	try {
		const response = await fetch(`${API_BASE}/apis/orders`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ apiOrders: orders }),
		})
		return await handleResponse<void>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function updateGroupOrders(orders: { id: number; order: number }[]): Promise<ApiResult<void>> {
	try {
		const response = await fetch(`${API_BASE}/groups/orders`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ groupOrders: orders }),
		})
		return await handleResponse<void>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

// Parameters API
export async function updateApiParametersFromJson(data: {
	apiId: number
	paramType: 'request' | 'response'
	json: Record<string, unknown>
}): Promise<ApiResult<void>> {
	try {
		const response = await fetch(`${API_BASE}/apis/${data.apiId}/parameters/from-json`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				paramType: data.paramType,
				json: data.json,
			}),
		})
		return await handleResponse<void>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

export async function updateApiParametersFromStructure(data: {
	apiId: number
	paramType: 'request' | 'response'
	parameters: ParameterWithChildren[]
}): Promise<ApiResult<{ count: number }>> {
	try {
		const response = await fetch(`${API_BASE}/apis/${data.apiId}/parameters`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				paramType: data.paramType,
				parameters: data.parameters,
			}),
		})
		return await handleResponse<{ count: number }>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

// Create API with parameters (V2)
export async function createApiV2(data: {
	groupId: number
	name: string
	endpoint: string
	method?: string
	type: string
	requestParameters?: Array<{
		name: string
		type: 'string' | 'number' | 'boolean' | 'array' | 'object'
		description?: string
		required: boolean
	}>
	responseParameters?: Array<{
		name: string
		type: 'string' | 'number' | 'boolean' | 'array' | 'object'
		description?: string
		required: boolean
	}>
}): Promise<
	ApiResult<{
		id: number
		name: string
		endpoint: string
		method: string | null
		type: string
		requestParameterCount: number
		responseParameterCount: number
	}>
> {
	try {
		// First create the API
		const createResponse = await fetch(`${API_BASE}/apis`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				groupId: data.groupId,
				name: data.name,
				method: data.method,
				endpoint: data.endpoint,
				type: data.type,
			}),
		})

		const createResult = await handleResponse<Api>(createResponse)

		if (!createResult.success || !createResult.data) {
			return { success: false, error: createResult.error || 'Failed to create API' }
		}

		const apiId = createResult.data.id

		// Then add request parameters if any
		if (data.requestParameters && data.requestParameters.length > 0) {
			const reqParamsResponse = await fetch(`${API_BASE}/apis/${apiId}/parameters`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					paramType: 'request',
					parameters: data.requestParameters,
				}),
			})

			const reqParamsResult = await handleResponse<{ count: number }>(reqParamsResponse)
			if (!reqParamsResult.success) {
				// Rollback by deleting the API
				await deleteApi(apiId)
				return { success: false, error: 'Failed to create request parameters' }
			}
		}

		// Then add response parameters if any
		if (data.responseParameters && data.responseParameters.length > 0) {
			const resParamsResponse = await fetch(`${API_BASE}/apis/${apiId}/parameters`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					paramType: 'response',
					parameters: data.responseParameters,
				}),
			})

			const resParamsResult = await handleResponse<{ count: number }>(resParamsResponse)
			if (!resParamsResult.success) {
				// Rollback by deleting the API
				await deleteApi(apiId)
				return { success: false, error: 'Failed to create response parameters' }
			}
		}

		return {
			success: true,
			data: {
				id: apiId,
				name: data.name,
				endpoint: data.endpoint,
				method: data.method || null,
				type: data.type,
				requestParameterCount: data.requestParameters?.length || 0,
				responseParameterCount: data.responseParameters?.length || 0,
			},
		}
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

// Get groups (simple list without APIs)
export async function getGroups(): Promise<ApiResult<Array<{ id: number; name: string }>>> {
	try {
		const response = await fetch(`${API_BASE}/groups`)
		return await handleResponse<Array<{ id: number; name: string }>>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}

// Export/Import API
export async function exportApis(apiIds: number[], format: ExportFormat): Promise<{ blob: Blob; filename: string } | { error: string }> {
	try {
		const response = await fetch(`${API_BASE}/export`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ apiIds, format }),
		})

		if (!response.ok) {
			return { error: `HTTP Error: ${response.status} ${response.statusText}` }
		}

		// Parse the filename from Content-Disposition, tolerating both
		// quoted ("name.ext") and unquoted (name.ext) forms.
		const disposition = response.headers.get('Content-Disposition') || ''
		const match = disposition.match(/filename="?([^";]+)"?/)
		const filename = match ? match[1] : 'knot-export'

		const blob = await response.blob()
		return { blob, filename }
	} catch (error) {
		return { error: String(error) }
	}
}

export async function importDocument(file: File, dryRun: boolean): Promise<ApiResult<ImportPreview>> {
	try {
		const formData = new FormData()
		formData.append('file', file)
		formData.append('dryRun', String(dryRun))

		// No Content-Type header: the browser must set the multipart boundary itself.
		const response = await fetch(`${API_BASE}/import`, {
			method: 'POST',
			body: formData,
		})
		return await handleResponse<ImportPreview>(response)
	} catch (error) {
		return { success: false, error: String(error) }
	}
}
