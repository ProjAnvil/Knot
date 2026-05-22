<script lang="ts">
  import { Plus, X } from '@lucide/svelte'
  import { toast } from 'svelte-sonner'
  import { _ } from 'svelte-i18n'
  import Button from '../ui/button.svelte'
  import Dialog from '../ui/dialog.svelte'
  import Input from '../ui/input.svelte'
  import Label from '../ui/label.svelte'
  import Select from '../ui/select.svelte'
  import Checkbox from '../ui/checkbox.svelte'
  import Textarea from '../ui/textarea.svelte'
  import { createApiV2, getGroups } from '$lib/api'
  import { onMount } from 'svelte'

  interface Parameter {
    id: string
    name: string
    type: 'string' | 'number' | 'boolean' | 'array' | 'object'
    description: string
    required: boolean
  }

  let {
    groupName,
    onSuccess
  }: {
    groupName: string
    onSuccess?: () => void
  } = $props()

  let open = $state(false)
  let groups = $state<Array<{ id: number; name: string }>>([])
  let selectedGroupId = $state<number | null>(null)
  let apiName = $state('')
  let endpoint = $state('')
  let method = $state<'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'>('GET')
  let type = $state<'HTTP' | 'RPC'>('HTTP')
  let requestParams = $state<Parameter[]>([])
  let responseParams = $state<Parameter[]>([])
  let loading = $state(false)

  // Fetch groups on mount
  onMount(async () => {
    const result = await getGroups()
    if (result.success && result.data) {
      groups = result.data
      // Set initial group based on groupName prop
      const matchingGroup = result.data.find((g) => g.name === groupName)
      if (matchingGroup) {
        selectedGroupId = matchingGroup.id
      }
    }
  })

  function addRequestParam() {
    requestParams = [
      ...requestParams,
      {
        id: Math.random().toString(36).substring(2, 11),
        name: '',
        type: 'string',
        description: '',
        required: false
      }
    ]
  }

  function addResponseParam() {
    responseParams = [
      ...responseParams,
      {
        id: Math.random().toString(36).substring(2, 11),
        name: '',
        type: 'string',
        description: '',
        required: false
      }
    ]
  }

  function removeRequestParam(id: string) {
    requestParams = requestParams.filter((p) => p.id !== id)
  }

  function removeResponseParam(id: string) {
    responseParams = responseParams.filter((p) => p.id !== id)
  }

  function updateRequestParam(id: string, field: keyof Parameter, value: string | boolean) {
    requestParams = requestParams.map((p) => (p.id === id ? { ...p, [field]: value } : p))
  }

  function updateResponseParam(id: string, field: keyof Parameter, value: string | boolean) {
    responseParams = responseParams.map((p) => (p.id === id ? { ...p, [field]: value } : p))
  }

  function resetForm() {
    apiName = ''
    endpoint = ''
    method = 'GET'
    type = 'HTTP'
    requestParams = []
    responseParams = []
  }

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault()

    // Validation
    if (!apiName.trim()) {
      toast.error($_('createApi.nameRequired'))
      return
    }

    if (!selectedGroupId) {
      toast.error($_('createApi.groupRequired'))
      return
    }

    // Validate parameter names
    const emptyReqParams = requestParams.filter((p) => !p.name.trim())
    const emptyResParams = responseParams.filter((p) => !p.name.trim())

    if (emptyReqParams.length > 0 || emptyResParams.length > 0) {
      toast.error($_('createApi.paramNamesRequired'))
      return
    }

    loading = true

    const input = {
      groupId: selectedGroupId,
      name: apiName,
      endpoint,
      type,
      method: type === 'HTTP' ? method : undefined,
      requestParameters: requestParams.map(({ id, ...rest }) => rest),
      responseParameters: responseParams.map(({ id, ...rest }) => rest)
    }

    const result = await createApiV2(input)

    if (result.success && result.data) {
      toast.success(
        `API "${apiName}" created with ${result.data.requestParameterCount} request and ${result.data.responseParameterCount} response parameters`
      )
      open = false
      resetForm()
      onSuccess?.()
    } else {
      toast.error(result.error || $_('createApi.createError'))
    }
    loading = false
  }

  const typeOptions = [
    { value: 'string', label: 'String' },
    { value: 'number', label: 'Number' },
    { value: 'boolean', label: 'Boolean' },
    { value: 'array', label: 'Array' },
    { value: 'object', label: 'Object' }
  ]

  const methodOptions = [
    { value: 'GET', label: 'GET' },
    { value: 'POST', label: 'POST' },
    { value: 'PUT', label: 'PUT' },
    { value: 'DELETE', label: 'DELETE' },
    { value: 'PATCH', label: 'PATCH' }
  ]

  const apiTypeOptions = [
    { value: 'HTTP', label: 'HTTP' },
    { value: 'RPC', label: 'RPC' }
  ]
</script>

<Dialog bind:open class="max-w-5xl max-h-[90vh] overflow-y-auto">
  {#snippet trigger()}
    <Button variant="ghost" size="sm" class="w-full justify-start text-muted-foreground h-8 px-2">
      <Plus class="h-3 w-3 mr-2" /> {$_('createApi.newApi')}
    </Button>
  {/snippet}

  <div class="space-y-4">
    <div class="space-y-2">
      <h2 class="text-lg font-semibold">{$_('createApi.title')}</h2>
    </div>

    <form onsubmit={handleSubmit} class="space-y-6">
      <!-- Basic Info -->
      <div class="grid grid-cols-3 gap-4">
        <div class="space-y-2">
          <Label for="apiName">{$_('createApi.apiNameLabel')}</Label>
          <Input
            id="apiName"
            bind:value={apiName}
            placeholder={$_('createApi.apiNamePlaceholder')}
            required
          />
        </div>
        <div class="space-y-2">
          <Label for="type">{$_('createApi.typeLabel')}</Label>
          <Select bind:value={type} options={apiTypeOptions} />
        </div>
        {#if type === 'HTTP'}
          <div class="space-y-2">
            <Label for="method">{$_('createApi.methodLabel')}</Label>
            <Select bind:value={method} options={methodOptions} />
          </div>
        {/if}
      </div>

      <div class="space-y-2">
        <Label for="endpoint">{$_('createApi.endpointLabel')}</Label>
        <Input
          id="endpoint"
          bind:value={endpoint}
          placeholder={$_('createApi.endpointPlaceholder')}
          required
        />
      </div>

      <!-- Request Parameters -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <Label class="text-base font-semibold">{$_('createApi.requestParams')}</Label>
          <Button type="button" onclick={addRequestParam} size="sm" variant="outline">
            <Plus class="h-4 w-4 mr-1" /> {$_('createApi.addParameter')}
          </Button>
        </div>

        {#if requestParams.length > 0}
          <div class="space-y-2">
            <div class="grid grid-cols-12 gap-2 px-3 text-xs font-medium text-muted-foreground">
              <div class="col-span-3">{$_('createApi.columnName')}</div>
              <div class="col-span-2">{$_('createApi.columnType')}</div>
              <div class="col-span-5">{$_('createApi.columnDescription')}</div>
              <div class="col-span-1 text-center">{$_('createApi.columnRequired')}</div>
              <div class="col-span-1"></div>
            </div>
            {#each requestParams as param (param.id)}
              <div class="grid grid-cols-12 gap-2 items-start p-3 bg-muted/50 rounded-md">
                <div class="col-span-3">
                  <Input
                    value={param.name}
                    oninput={(e: Event & { currentTarget: HTMLInputElement }) => updateRequestParam(param.id, 'name', e.currentTarget.value)}
                    placeholder={$_('createApi.paramNamePlaceholder')}
                    class="h-9"
                  />
                </div>
                <div class="col-span-2">
                  <Select
                    value={param.type}
                    onValueChange={(value) => updateRequestParam(param.id, 'type', value)}
                    options={typeOptions}
                    class="h-9"
                  />
                </div>
                <div class="col-span-5">
                  <Textarea
                    value={param.description}
                    oninput={(e: Event & { currentTarget: HTMLTextAreaElement }) => updateRequestParam(param.id, 'description', e.currentTarget.value)}
                    placeholder={$_('createApi.paramDescPlaceholder')}
                    class="h-9 min-h-9 resize-none"
                    rows={1}
                  />
                </div>
                <div class="col-span-1 flex items-center justify-center pt-1">
                  <Checkbox
                    checked={param.required}
                    onCheckedChange={(checked) => updateRequestParam(param.id, 'required', checked)}
                  />
                </div>
                <div class="col-span-1 flex items-center justify-center">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onclick={() => removeRequestParam(param.id)}
                    class="h-9 w-9 p-0"
                  >
                    <X class="h-4 w-4" />
                  </Button>
                </div>
              </div>
            {/each}
          </div>
        {/if}

        {#if requestParams.length === 0}
          <p class="text-sm text-muted-foreground italic">
            {$_('createApi.noRequestParams')}
          </p>
        {/if}
      </div>

      <!-- Response Parameters -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <Label class="text-base font-semibold">{$_('createApi.responseParams')}</Label>
          <Button type="button" onclick={addResponseParam} size="sm" variant="outline">
            <Plus class="h-4 w-4 mr-1" /> {$_('createApi.addParameter')}
          </Button>
        </div>

        {#if responseParams.length > 0}
          <div class="space-y-2">
            <div class="grid grid-cols-12 gap-2 px-3 text-xs font-medium text-muted-foreground">
              <div class="col-span-3">{$_('createApi.columnName')}</div>
              <div class="col-span-2">{$_('createApi.columnType')}</div>
              <div class="col-span-5">{$_('createApi.columnDescription')}</div>
              <div class="col-span-1 text-center">{$_('createApi.columnRequired')}</div>
              <div class="col-span-1"></div>
            </div>
            {#each responseParams as param (param.id)}
              <div class="grid grid-cols-12 gap-2 items-start p-3 bg-muted/50 rounded-md">
                <div class="col-span-3">
                  <Input
                    value={param.name}
                    oninput={(e: Event & { currentTarget: HTMLInputElement }) => updateResponseParam(param.id, 'name', e.currentTarget.value)}
                    placeholder={$_('createApi.paramNamePlaceholder')}
                    class="h-9"
                  />
                </div>
                <div class="col-span-2">
                  <Select
                    value={param.type}
                    onValueChange={(value) => updateResponseParam(param.id, 'type', value)}
                    options={typeOptions}
                    class="h-9"
                  />
                </div>
                <div class="col-span-5">
                  <Textarea
                    value={param.description}
                    oninput={(e: Event & { currentTarget: HTMLTextAreaElement }) => updateResponseParam(param.id, 'description', e.currentTarget.value)}
                    placeholder={$_('createApi.paramDescPlaceholder')}
                    class="h-9 min-h-9 resize-none"
                    rows={1}
                  />
                </div>
                <div class="col-span-1 flex items-center justify-center pt-1">
                  <Checkbox
                    checked={param.required}
                    onCheckedChange={(checked) => updateResponseParam(param.id, 'required', checked)}
                  />
                </div>
                <div class="col-span-1 flex items-center justify-center">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onclick={() => removeResponseParam(param.id)}
                    class="h-9 w-9 p-0"
                  >
                    <X class="h-4 w-4" />
                  </Button>
                </div>
              </div>
            {/each}
          </div>
        {/if}

        {#if responseParams.length === 0}
          <p class="text-sm text-muted-foreground italic">
            {$_('createApi.noResponseParams')}
          </p>
        {/if}
      </div>

      <div class="flex justify-end gap-2 pt-4 border-t">
        <Button type="button" variant="outline" onclick={() => (open = false)}>
          {$_('common.cancel')}
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? $_('common.creating') : $_('api.create')}
        </Button>
      </div>
    </form>
  </div>
</Dialog>
