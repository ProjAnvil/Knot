<script lang="ts">
  import { _ } from 'svelte-i18n'
  import type { ApiData, ParameterWithChildren } from '$lib/types'
  import { updateApiParametersFromJson, updateApiParametersFromStructure } from '$lib/api'
  import Badge from './ui/badge.svelte'
  import EditableApiName from './doc-viewer/EditableApiName.svelte'
  import EditableEndpoint from './doc-viewer/EditableEndpoint.svelte'
  import EditableJson from './doc-viewer/EditableJson.svelte'
  import EditableNote from './doc-viewer/EditableNote.svelte'
  import EditableParameterTable from './doc-viewer/EditableParameterTable.svelte'

  let {
    apiData,
    onDataChange,
  }: {
    apiData: ApiData
    onDataChange?: () => void
    onStructuralChange?: () => void
  } = $props()

  /**
   * Generate example JSON from parameters with nested support
   * Handles primitives, objects, and arrays with proper nesting
   */
  function generateExampleJson(parameters: ParameterWithChildren[]): Record<string, unknown> | unknown[] {
    const result: Record<string, unknown> = {}

    parameters.forEach((param) => {
      switch (param.type) {
        case 'string':
          result[param.name] = param.name
          break
        case 'number':
          result[param.name] = 0
          break
        case 'boolean':
          result[param.name] = false
          break
        case 'array':
          // If has children, generate array with child values (not wrapped in object)
          if (param.children && param.children.length > 0) {
            // For array, children represent the item schema
            // If array items are primitives (single child), use that value directly
            // If array items are objects (multiple children), build object structure
            if (param.children.length === 1 && ['string', 'number', 'boolean'].includes(param.children[0].type)) {
              // Array of primitives: ["string", "string"] or [0, 0]
              const child = param.children[0]
              const primitiveValue = child.type === 'string' ? child.name : child.type === 'number' ? 0 : false
              result[param.name] = [primitiveValue]
            } else {
              // Array of objects: [{key1: val1, key2: val2}]
              const itemExample = generateExampleJson(param.children)
              result[param.name] = [itemExample]
            }
          } else {
            result[param.name] = []
          }
          break
        case 'object':
          // If has children, recursively build object
          if (param.children && param.children.length > 0) {
            result[param.name] = generateExampleJson(param.children)
          } else {
            result[param.name] = {}
          }
          break
        default:
          result[param.name] = null
      }
    })

    return result
  }

  let requestJson = $state(generateExampleJson(apiData.requestParameters))
  let responseJson = $state(generateExampleJson(apiData.responseParameters))

  // Update JSON examples when apiData changes (when switching between APIs)
  $effect(() => {
    requestJson = generateExampleJson(apiData.requestParameters)
    responseJson = generateExampleJson(apiData.responseParameters)
  })

  async function handleRequestSave(newJson: Record<string, unknown>) {
    const result = await updateApiParametersFromJson({
      apiId: apiData.id,
      paramType: 'request',
      json: newJson,
    })

    if (result.success) {
      requestJson = newJson
      onDataChange?.()
    }

    return result
  }

  async function handleResponseSave(newJson: Record<string, unknown>) {
    const result = await updateApiParametersFromJson({
      apiId: apiData.id,
      paramType: 'response',
      json: newJson,
    })

    if (result.success) {
      responseJson = newJson
      onDataChange?.()
    }

    return result
  }

  function getMethodColorClasses(method: string): string {
    const colors: Record<string, string> = {
      GET: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
      POST: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
      PUT: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
      DELETE: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
      PATCH: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
    }
    return colors[method] || 'bg-gray-100 text-gray-800'
  }

  async function handleRequestParamsSave(params: ParameterWithChildren[]) {
    const result = await updateApiParametersFromStructure({
      apiId: apiData.id,
      paramType: 'request',
      parameters: params,
    })

    if (result.success) {
      onDataChange?.()
    }

    return result
  }

  async function handleResponseParamsSave(params: ParameterWithChildren[]) {
    const result = await updateApiParametersFromStructure({
      apiId: apiData.id,
      paramType: 'response',
      parameters: params,
    })

    if (result.success) {
      onDataChange?.()
    }

    return result
  }
</script>

<div class="p-6 space-y-6">
  <!-- API Header -->
  <div class="space-y-4">
    <div class="flex items-center gap-3">
      <EditableApiName apiId={apiData.id} apiName={apiData.name} onDataChange={onDataChange} />
    </div>

    <div class="flex items-center gap-3">
      <Badge variant="secondary" class="text-sm px-3 py-1">
        {apiData.type}
      </Badge>
      {#if apiData.method}
        <Badge class={getMethodColorClasses(apiData.method)}>
          {apiData.method}
        </Badge>
      {/if}
      <EditableEndpoint apiId={apiData.id} endpoint={apiData.endpoint} onDataChange={onDataChange} />
    </div>
  </div>

  <!-- Request Parameters -->
  <EditableParameterTable
    parameters={apiData.requestParameters}
    title={$_('parameters.request')}
    onSave={handleRequestParamsSave}
  />

  <!-- Request JSON Editor -->
  {#if apiData.requestParameters.length > 0}
    <EditableJson
      title={$_('json.requestExample')}
      json={requestJson}
      onSave={handleRequestSave}
    />
  {/if}

  <!-- Response Parameters -->
  <EditableParameterTable
    parameters={apiData.responseParameters}
    title={$_('parameters.response')}
    onSave={handleResponseParamsSave}
  />

  <!-- Response JSON Editor -->
  {#if apiData.responseParameters.length > 0}
    <EditableJson
      title={$_('json.responseExample')}
      json={responseJson}
      onSave={handleResponseSave}
    />
  {/if}

  <!-- API Note -->
  <EditableNote apiId={apiData.id} initialNote={apiData.note} onDataChange={onDataChange} />
</div>
