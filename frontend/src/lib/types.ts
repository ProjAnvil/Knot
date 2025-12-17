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
	children?: ParameterWithChildren[]
}

export interface GroupWithApis extends Group {
	apis: Api[]
}

export interface ApiData extends Api {
	group?: Group
	requestParameters: ParameterWithChildren[]
	responseParameters: ParameterWithChildren[]
}

export interface ApiResult<T = any> {
	success: boolean
	data?: T
	error?: string
}
