<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { toast } from 'svelte-sonner'
  import { Edit2, FileInput, FileOutput, Plus, Save, Trash2 } from 'lucide-svelte'
  import Button from '../ui/button.svelte'
  import Badge from '../ui/badge.svelte'
  import Input from '../ui/input.svelte'
  import Textarea from '../ui/textarea.svelte'
  import Checkbox from '../ui/checkbox.svelte'
  import Select from '../ui/select.svelte'
  import type { ParameterWithChildren } from '$lib/types'

  let {
    parameters,
    title,
    onSave
  }: {
    parameters: ParameterWithChildren[]
    title: string
    onSave: (params: ParameterWithChildren[]) => Promise<{ success: boolean; error?: string }>
  } = $props()

  type FlatParam = ParameterWithChildren & { depth: number; tempId: string }

  let isEditing = $state(false)
  let editParams = $state<FlatParam[]>([])
  let isSaving = $state(false)

  const TYPE_COLORS: Record<string, string> = {
    string: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    number: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    boolean: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
    array: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
    object: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
  }

  // Flatten parameters for editing
  function flattenParameters(params: ParameterWithChildren[], depth = 0): FlatParam[] {
    const result: FlatParam[] = []
    params.forEach((param) => {
      const tempId = `param_${param.id || Math.random().toString(36).substr(2, 9)}`
      result.push({ ...param, depth, tempId })
      if (param.children && param.children.length > 0) {
        result.push(...flattenParameters(param.children, depth + 1))
      }
    })
    return result
  }

  // Rebuild hierarchy from flat list
  function buildHierarchy(flatParams: FlatParam[]): ParameterWithChildren[] {
    const result: ParameterWithChildren[] = []
    const stack: Array<{ param: ParameterWithChildren; depth: number }> = []

    flatParams.forEach((item) => {
      const param: ParameterWithChildren = { ...item }
      delete (param as any).depth
      delete (param as any).tempId
      param.children = []

      // Pop stack until we find the parent
      while (stack.length > 0 && stack[stack.length - 1].depth >= item.depth) {
        stack.pop()
      }

      if (stack.length === 0) {
        result.push(param)
      } else {
        const parent = stack[stack.length - 1].param
        if (!parent.children) parent.children = []
        parent.children.push(param)
      }

      stack.push({ param, depth: item.depth })
    })

    return result
  }

  function handleEdit() {
    editParams = flattenParameters(parameters)
    isEditing = true
  }

  function handleCancel() {
    isEditing = false
    editParams = []
  }

  async function handleSave() {
    // Validate
    const emptyNames = editParams.filter((p) => !p.name.trim())
    if (emptyNames.length > 0) {
      toast.error($_('parameters.updateError'))
      return
    }

    isSaving = true
    const hierarchy = buildHierarchy(editParams)
    const result = await onSave(hierarchy)
    isSaving = false

    if (result.success) {
      toast.success($_('parameters.updateSuccess'))
      isEditing = false
      editParams = []
    } else {
      toast.error(result.error || $_('parameters.updateError'))
    }
  }

  function updateParam(tempId: string, field: keyof ParameterWithChildren, value: any) {
    editParams = editParams.map((p) => (p.tempId === tempId ? { ...p, [field]: value } : p))
  }

  function addParam(afterTempId?: string) {
    const newParam: FlatParam = {
      id: undefined as any,
      name: '',
      type: 'string',
      description: null,
      required: false,
      depth: 0,
      tempId: `new_${Math.random().toString(36).substr(2, 9)}`,
      apiId: parameters[0]?.apiId || 0,
      paramType: parameters[0]?.paramType || 'request',
      order: 0,
      parentId: null,
      createdAt: new Date(),
      updatedAt: new Date(),
    }

    if (afterTempId) {
      const index = editParams.findIndex((p) => p.tempId === afterTempId)
      const afterParam = editParams[index]
      newParam.depth = afterParam.depth
      editParams = [...editParams.slice(0, index + 1), newParam, ...editParams.slice(index + 1)]
    } else {
      editParams = [...editParams, newParam]
    }
  }

  function deleteParam(tempId: string) {
    const index = editParams.findIndex((p) => p.tempId === tempId)
    const param = editParams[index]

    // Remove param and all its children
    const toRemove = [tempId]
    for (let i = index + 1; i < editParams.length; i++) {
      if (editParams[i].depth <= param.depth) break
      toRemove.push(editParams[i].tempId)
    }

    editParams = editParams.filter((p) => !toRemove.includes(p.tempId))
  }

  function indent(tempId: string) {
    editParams = editParams.map((p) => (p.tempId === tempId ? { ...p, depth: Math.min(p.depth + 1, 3) } : p))
  }

  function outdent(tempId: string) {
    editParams = editParams.map((p) => (p.tempId === tempId ? { ...p, depth: Math.max(p.depth - 1, 0) } : p))
  }

  // Render nested parameters recursively
  function renderParameterRows(params: ParameterWithChildren[], depth = 0): any[] {
    const rows: any[] = []
    for (const param of params) {
      rows.push({ ...param, depth })
      if (param.children && param.children.length > 0) {
        rows.push(...renderParameterRows(param.children, depth + 1))
      }
    }
    return rows
  }

  const flatParams = $derived(renderParameterRows(parameters))
</script>

{#if parameters.length === 0 && !isEditing}
  <div class="space-y-2">
    <div class="flex items-center justify-between">
      <h3 class="text-lg font-semibold flex items-center gap-2">
        {#if title.toLowerCase().includes('request') || title.includes('请求')}
          <FileInput class="h-5 w-5 text-blue-600" />
        {:else}
          <FileOutput class="h-5 w-5 text-green-600" />
        {/if}
        {title}
      </h3>
      <Button variant="outline" size="sm" onclick={handleEdit} class="gap-2">
        <Plus class="h-4 w-4" />
        {$_('parameters.addParameters')}
      </Button>
    </div>
    <p class="text-sm text-muted-foreground italic">{$_('parameters.noParameters')}</p>
  </div>
{:else}
  <div class="space-y-3">
    <div class="flex items-center justify-between">
      <h3 class="text-lg font-semibold flex items-center gap-2">
        {#if title.toLowerCase().includes('request') || title.includes('请求')}
          <FileInput class="h-5 w-5 text-blue-600" />
        {:else}
          <FileOutput class="h-5 w-5 text-green-600" />
        {/if}
        {title}
      </h3>
      {#if !isEditing}
        <Button variant="outline" size="sm" onclick={handleEdit} class="gap-2">
          <Edit2 class="h-4 w-4" />
          {$_('common.edit')}
        </Button>
      {:else}
        <div class="flex gap-2">
          <Button variant="outline" size="sm" onclick={handleCancel} disabled={isSaving}>
            {$_('common.cancel')}
          </Button>
          <Button size="sm" onclick={handleSave} disabled={isSaving}>
            <Save class="h-4 w-4 mr-2" />
            {isSaving ? $_('common.saving') : $_('common.save')}
          </Button>
        </div>
      {/if}
    </div>

    {#if isEditing}
      <div class="space-y-2">
        <div class="rounded-md border overflow-x-auto">
          <table class="w-full text-sm">
            <thead class="bg-muted">
              <tr>
                <th class="px-4 py-2 text-left font-semibold w-[200px]">{$_('parameters.name')}</th>
                <th class="px-4 py-2 text-left font-semibold w-[120px]">{$_('parameters.type')}</th>
                <th class="px-4 py-2 text-left font-semibold w-[100px]">{$_('parameters.required')}</th>
                <th class="px-4 py-2 text-left font-semibold">{$_('parameters.description')}</th>
                <th class="px-4 py-2 text-left font-semibold w-[150px]">{$_('parameters.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {#each editParams as param (param.tempId)}
                <tr class="border-t">
                  <td class="px-4 py-2">
                    <Input
                      value={param.name}
                      oninput={(e) => updateParam(param.tempId, 'name', e.currentTarget.value)}
                      placeholder={$_('parameters.name')}
                      class="h-8"
                      style="padding-left: {param.depth * 20 + 8}px"
                    />
                  </td>
                  <td class="px-4 py-2">
                    <Select
                      value={param.type}
                      onValueChange={(v) => updateParam(param.tempId, 'type', v)}
                      options={[
                        { value: 'string', label: $_('types.string') },
                        { value: 'number', label: $_('types.number') },
                        { value: 'boolean', label: $_('types.boolean') },
                        { value: 'array', label: $_('types.array') },
                        { value: 'object', label: $_('types.object') },
                      ]}
                      class="h-8"
                    />
                  </td>
                  <td class="px-4 py-2">
                    <Checkbox
                      checked={param.required}
                      onCheckedChange={(checked) => updateParam(param.tempId, 'required', checked === true)}
                    />
                  </td>
                  <td class="px-4 py-2">
                    <Textarea
                      value={param.description || ''}
                      oninput={(e) => updateParam(param.tempId, 'description', e.currentTarget.value || null)}
                      placeholder={$_('parameters.description')}
                      class="h-8 min-h-8 resize-none"
                      rows={1}
                    />
                  </td>
                  <td class="px-4 py-2">
                    <div class="flex gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onclick={() => outdent(param.tempId)}
                        disabled={param.depth === 0}
                        class="h-8 w-8 p-0"
                        title={$_('parameters.decreaseIndent')}
                      >
                        ←
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onclick={() => indent(param.tempId)}
                        disabled={param.depth >= 3}
                        class="h-8 w-8 p-0"
                        title={$_('parameters.increaseIndent')}
                      >
                        →
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onclick={() => addParam(param.tempId)}
                        class="h-8 w-8 p-0"
                        title={$_('parameters.addAfter')}
                      >
                        <Plus class="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onclick={() => deleteParam(param.tempId)}
                        class="h-8 w-8 p-0 text-destructive"
                        title={$_('parameters.deleteParameter')}
                      >
                        <Trash2 class="h-4 w-4" />
                      </Button>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        <Button variant="outline" size="sm" onclick={() => addParam()} class="gap-2">
          <Plus class="h-4 w-4" />
          {$_('parameters.addParameter')}
        </Button>
      </div>
    {:else}
      <div class="border rounded-lg overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-muted">
            <tr>
              <th class="px-4 py-2 text-left font-semibold w-[200px]">{$_('parameters.name')}</th>
              <th class="px-4 py-2 text-left font-semibold w-[120px]">{$_('parameters.type')}</th>
              <th class="px-4 py-2 text-left font-semibold w-[100px]">{$_('parameters.required')}</th>
              <th class="px-4 py-2 text-left font-semibold">{$_('parameters.description')}</th>
            </tr>
          </thead>
          <tbody>
            {#each flatParams as param (param.id)}
              <tr class="border-t hover:bg-muted/50">
                <td class="px-4 py-2 font-mono text-sm">
                  <span style="padding-left: {param.depth * 20}px" class="inline-flex items-center gap-2">
                    {#if param.depth > 0}
                      <span class="text-muted-foreground">└─ </span>
                    {/if}
                    {param.name}
                  </span>
                </td>
                <td class="px-4 py-2">
                  <Badge variant="secondary" class={TYPE_COLORS[param.type] || 'bg-gray-100 text-gray-800'}>
                    {param.type}
                  </Badge>
                </td>
                <td class="px-4 py-2">
                  {#if param.required}
                    <span class="text-xs font-semibold text-destructive">
                      {$_('common.required')}
                    </span>
                  {:else}
                    <span class="text-xs text-muted-foreground">{$_('common.optional')}</span>
                  {/if}
                </td>
                <td class="px-4 py-2 text-sm text-muted-foreground">
                  {#if param.description}
                    {param.description}
                  {:else}
                    <span class="italic">{$_('parameters.noParameters')}</span>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
{/if}
