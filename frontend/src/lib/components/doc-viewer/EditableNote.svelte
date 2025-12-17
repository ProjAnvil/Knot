<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { toast } from 'svelte-sonner'
  import { Pencil, Save, StickyNote, X, Code, Eye } from 'lucide-svelte'
  import Button from '../ui/button.svelte'
  import Textarea from '../ui/textarea.svelte'
  import { updateApi } from '$lib/api'
  import { marked } from 'marked'
  import { gfmHeadingId } from 'marked-gfm-heading-id'

  let {
    apiId,
    initialNote,
    onDataChange
  }: {
    apiId: number
    initialNote: string | null
    onDataChange?: () => void
  } = $props()

  let isEditing = $state(false)
  let note = $state(initialNote || '')
  let isSaving = $state(false)
  let activeTab = $state<'edit' | 'preview'>('edit')

  // Update note when switching to a different API
  $effect(() => {
    note = initialNote || ''
    isEditing = false
    activeTab = 'edit'
  })

  // Configure marked for GitHub Flavored Markdown
  marked.use(gfmHeadingId())
  marked.setOptions({
    gfm: true,
    breaks: true,
  })

  const renderedMarkdown = $derived(note ? marked.parse(note) as string : '')

  async function handleSave() {
    isSaving = true
    try {
      const result = await updateApi(apiId, { note })
      if (result.success) {
        toast.success($_('api.noteSaved'))
        isEditing = false
        onDataChange?.()
      } else {
        toast.error(result.error || $_('api.noteSaveFailed'))
      }
    } finally {
      isSaving = false
    }
  }

  function handleCancel() {
    note = initialNote || ''
    isEditing = false
    activeTab = 'edit'
  }
</script>

<div class="space-y-2">
  <div class="flex items-center justify-between">
    <h3 class="text-lg font-semibold flex items-center gap-2">
      <StickyNote class="h-5 w-5 text-yellow-600" />
      {$_('api.note')}
    </h3>
    {#if !isEditing}
      <Button variant="ghost" size="sm" onclick={() => isEditing = true}>
        <Pencil class="h-4 w-4 mr-2" />
        {$_('common.edit')}
      </Button>
    {/if}
  </div>

  {#if isEditing}
    <div class="space-y-2">
      <!-- Tabs for Edit/Preview -->
      <div class="w-full">
        <div class="flex border-b">
          <button
            class="px-4 py-2 text-sm font-medium flex items-center gap-2 border-b-2 transition-colors {activeTab === 'edit' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
            onclick={() => activeTab = 'edit'}
          >
            <Code class="h-4 w-4" />
            {$_('api.edit')}
          </button>
          <button
            class="px-4 py-2 text-sm font-medium flex items-center gap-2 border-b-2 transition-colors {activeTab === 'preview' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
            onclick={() => activeTab = 'preview'}
          >
            <Eye class="h-4 w-4" />
            {$_('api.preview')}
          </button>
        </div>

        <div class="mt-2">
          {#if activeTab === 'edit'}
            <Textarea
              bind:value={note}
              placeholder={$_('api.notePlaceholder')}
              class="min-h-[200px] font-mono text-sm"
            />
          {:else}
            <div class="min-h-[200px] border rounded-md p-4 prose prose-sm max-w-none dark:prose-invert">
              {#if note}
                {@html renderedMarkdown}
              {:else}
                <p class="text-muted-foreground italic">{$_('api.noNote')}</p>
              {/if}
            </div>
          {/if}
        </div>
      </div>

      <div class="flex gap-2">
        <Button onclick={handleSave} disabled={isSaving} size="sm">
          <Save class="h-4 w-4 mr-2" />
          {isSaving ? $_('common.saving') : $_('common.save')}
        </Button>
        <Button variant="outline" onclick={handleCancel} disabled={isSaving} size="sm">
          <X class="h-4 w-4 mr-2" />
          {$_('common.cancel')}
        </Button>
      </div>
    </div>
  {:else}
    <div class="prose prose-sm max-w-none dark:prose-invert">
      {#if note}
        {@html renderedMarkdown}
      {:else}
        <p class="text-muted-foreground italic">{$_('api.noNote')}</p>
      {/if}
    </div>
  {/if}
</div>
