// Type definitions matching backend database schema

export interface Group {
	id: number
	name: string
	createdAt: string
	updatedAt: string
}

export interface Api {
	id: number
	groupId: number
	name: string
	method: string
	endpoint: string
	type: string
	note: string | null
	order: number
	createdAt: string
	updatedAt: string
}

export interface Parameter {
	id: number
	apiId: number
	name: string
	type: 'string' | 'number' | 'boolean' | 'array' | 'object'
	required: boolean
	description: string | null
	paramType: 'request' | 'response'
	parentId: number | null
	order: number
}

export interface ParameterWithChildren extends Parameter {
	children: ParameterWithChildren[]
}

export interface GroupWithApis extends Group {
	apis: Api[]
}

// Raw API response from backend (before transformation)
export interface ApiRawResponse extends Omit<Api, 'method'> {
	group?: Group
	groupId: number
	method: string | null
	parameters: Parameter[]
}

export interface ApiData extends Api {
	group?: Group
	requestParameters: ParameterWithChildren[]
	responseParameters: ParameterWithChildren[]
}

export interface ApiResult<T = void> {
	success: boolean
	data?: T
	error?: string
}
