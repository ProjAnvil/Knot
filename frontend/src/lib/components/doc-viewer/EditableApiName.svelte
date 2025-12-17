<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { toast } from 'svelte-sonner'
  import { Check, Pencil, X } from 'lucide-svelte'
  import Button from '../ui/button.svelte'
  import Input from '../ui/input.svelte'
  import { updateApi } from '$lib/api'

  let {
    apiId,
    apiName,
    onDataChange,
    className = ''
  }: {
    apiId: number
    apiName: string
    onDataChange?: () => void
    className?: string
  } = $props()

  let isEditing = $state(false)
  let name = $state(apiName)
  let isSaving = $state(false)

  // Sync name with apiName prop when it changes (e.g., when switching APIs)
  $effect(() => {
    name = apiName
  })

  async function handleSave() {
    if (!name.trim()) {
      toast.error($_('api.nameRequired'))
      return
    }

    if (name === apiName) {
      isEditing = false
      return
    }

    isSaving = true
    const result = await updateApi(apiId, { name: name.trim() })
    isSaving = false

    if (result.success) {
      toast.success($_('api.updateSuccess'))
      isEditing = false
      onDataChange?.()
    } else {
      toast.error(result.error || $_('api.updateError'))
      name = apiName // Revert on error
    }
  }

  function handleCancel() {
    name = apiName
    isEditing = false
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleSave()
    } else if (e.key === 'Escape') {
      handleCancel()
    }
  }

  function handleBlur() {
    // Delay to allow button clicks to process first
    setTimeout(() => {
      if (isEditing && !isSaving) {
        handleCancel()
      }
    }, 200)
  }
</script>

{#if isEditing}
  <div class="flex items-center gap-2 flex-1">
    <Input
      bind:value={name}
      onkeydown={handleKeyDown}
      onblur={handleBlur}
      class="text-3xl font-bold h-auto py-1"
      disabled={isSaving}
      autofocus
    />
    <Button
      size="sm"
      variant="ghost"
      class="h-8 w-8 p-0 text-green-600 hover:text-green-700 hover:bg-green-50"
      onclick={handleSave}
      disabled={isSaving}
    >
      <Check class="h-4 w-4" />
    </Button>
    <Button
      size="sm"
      variant="ghost"
      class="h-8 w-8 p-0 text-red-600 hover:text-red-700 hover:bg-red-50"
      onclick={handleCancel}
      disabled={isSaving}
    >
      <X class="h-4 w-4" />
    </Button>
  </div>
{:else}
  <div
    class="flex items-center gap-2 group cursor-pointer flex-1 {className}"
    onclick={() => isEditing = true}
    role="button"
    tabindex="0"
    onkeypress={(e) => e.key === 'Enter' && (isEditing = true)}
  >
    <h1 class="text-3xl font-bold">{apiName}</h1>
    <Pencil class="h-5 w-5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
  </div>
{/if}
