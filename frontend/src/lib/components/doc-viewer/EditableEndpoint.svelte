<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { toast } from 'svelte-sonner'
  import { Check, Pencil, X } from 'lucide-svelte'
  import Button from '../ui/button.svelte'
  import Input from '../ui/input.svelte'
  import { updateApi } from '$lib/api'

  let {
    apiId,
    endpoint,
    onDataChange,
    className = ''
  }: {
    apiId: number
    endpoint: string
    onDataChange?: () => void
    className?: string
  } = $props()

  let isEditing = $state(false)
  let value = $state(endpoint)
  let isSaving = $state(false)

  // Sync value with endpoint prop when it changes (e.g., when switching APIs)
  $effect(() => {
    value = endpoint
  })

  async function handleSave() {
    if (!value.trim()) {
      toast.error($_('api.endpointRequired') || 'Endpoint is required')
      return
    }

    if (value === endpoint) {
      isEditing = false
      return
    }

    isSaving = true
    const result = await updateApi(apiId, { endpoint: value.trim() })
    isSaving = false

    if (result.success) {
      toast.success($_('api.updateSuccess'))
      isEditing = false
      onDataChange?.()
    } else {
      toast.error(result.error || $_('api.updateError'))
      value = endpoint // Revert on error
    }
  }

  function handleCancel() {
    value = endpoint
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
  <div class="inline-flex items-center gap-2 max-w-2xl">
    <Input
      bind:value
      onkeydown={handleKeyDown}
      onblur={handleBlur}
      class="font-mono text-lg h-auto py-2 min-w-[400px]"
      disabled={isSaving}
      autofocus
    />
    <Button
      size="sm"
      variant="ghost"
      class="h-8 w-8 p-0 flex-shrink-0 text-green-600 hover:text-green-700 hover:bg-green-50"
      onclick={handleSave}
      disabled={isSaving}
    >
      <Check class="h-4 w-4" />
    </Button>
    <Button
      size="sm"
      variant="ghost"
      class="h-8 w-8 p-0 flex-shrink-0 text-red-600 hover:text-red-700 hover:bg-red-50"
      onclick={handleCancel}
      disabled={isSaving}
    >
      <X class="h-4 w-4" />
    </Button>
  </div>
{:else}
  <div
    class="inline-flex items-center gap-2 group cursor-pointer {className}"
    onclick={() => isEditing = true}
    role="button"
    tabindex="0"
    onkeypress={(e) => e.key === 'Enter' && (isEditing = true)}
  >
    <code class="text-lg font-mono bg-muted px-4 py-2 rounded-md border">
      {endpoint}
    </code>
    <Pencil class="h-5 w-5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
  </div>
{/if}
