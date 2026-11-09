<script lang="ts">
  import { FileUp, TriangleAlert, Upload } from '@lucide/svelte'
  import { toast } from 'svelte-sonner'
  import { _ } from 'svelte-i18n'
  import Button from '../ui/button.svelte'
  import Dialog from '../ui/dialog.svelte'
  import { importDocument } from '$lib/api'
  import type { ImportPreview } from '$lib/types'

  let {
    onImported
  }: {
    onImported: () => void
  } = $props()

  let open = $state(false)
  let file = $state<File | null>(null)
  let preview = $state<ImportPreview | null>(null)
  let errorMessage = $state('')
  let isAnalyzing = $state(false)
  let isImporting = $state(false)

  // The backend rejects unknown formats with HTTP 415; either the status
  // code or an "unsupported" wording in the message identifies that case
  function isUnsupportedFormat(message: string): boolean {
    return /415|unsupported/i.test(message)
  }

  // Unsupported-format errors get a translated message; everything else is
  // surfaced verbatim so users see the actual server response
  function displayError(result: { error?: string }): string {
    const message = result.error || $_('import.error')
    return isUnsupportedFormat(result.error || '') ? $_('import.unsupportedFormat') : message
  }

  function reset() {
    file = null
    preview = null
    errorMessage = ''
    isAnalyzing = false
    isImporting = false
  }

  function handleOpenChange(value: boolean) {
    open = value
    if (!value) reset()
  }

  // Picking a file immediately dry-runs the import so the user can review
  // what would happen before committing to it
  async function handleFileSelect(event: Event) {
    const input = event.currentTarget as HTMLInputElement
    const selected = input.files?.[0] ?? null

    // Allow picking the same file again on a retry
    input.value = ''

    preview = null
    errorMessage = ''

    if (!selected) return

    file = selected
    isAnalyzing = true
    const result = await importDocument(selected, true)
    isAnalyzing = false

    if (!result.success || !result.data) {
      file = null
      errorMessage = displayError(result)
      return
    }

    preview = result.data
  }

  async function handleConfirm() {
    if (!file) return

    isImporting = true
    const result = await importDocument(file, false)
    isImporting = false

    if (!result.success) {
      errorMessage = displayError(result)
      toast.error(errorMessage)
      return
    }

    toast.success($_('import.success'))
    onImported()
    handleOpenChange(false)
  }
</script>

<Dialog bind:open onOpenChange={handleOpenChange}>
  {#snippet trigger()}
    <Button variant="outline" size="sm" class="gap-2" data-testid="import-dialog-trigger">
      <Upload class="h-4 w-4" />
      {$_('import.title')}
    </Button>
  {/snippet}

  <div class="space-y-4">
    <div class="space-y-2">
      <h2 class="text-lg font-semibold">{$_('import.selectFile')}</h2>
    </div>

    <!-- File input: selecting a file triggers the dry-run preview -->
    <input
      type="file"
      accept=".json,application/json"
      aria-label={$_('import.selectFile')}
      data-testid="import-file-input"
      onchange={handleFileSelect}
      class="text-sm"
    />

    {#if isAnalyzing}
      <p class="text-sm text-muted-foreground">{$_('common.loading')}</p>
    {/if}

    {#if errorMessage}
      <div
        data-testid="import-error"
        class="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
      >
        {errorMessage}
      </div>
    {/if}

    {#if preview}
      <div data-testid="import-preview" class="space-y-3 rounded-md border p-4">
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium">{$_('import.detectedFormat')}</span>
          <span class="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium uppercase text-primary">
            {preview.format}
          </span>
        </div>

        <div class="grid grid-cols-2 gap-2 text-sm">
          <div data-testid="import-count-groups-created">
            <span class="text-muted-foreground">{$_('import.groupsCreated')}: </span>
            <span class="font-medium">{preview.groupsCreated}</span>
          </div>
          <div data-testid="import-count-groups-merged">
            <span class="text-muted-foreground">{$_('import.groupsMerged')}: </span>
            <span class="font-medium">{preview.groupsMerged}</span>
          </div>
          <div data-testid="import-count-apis-imported">
            <span class="text-muted-foreground">{$_('import.apisImported')}: </span>
            <span class="font-medium">{preview.apisImported}</span>
          </div>
          <div data-testid="import-count-parameters-imported">
            <span class="text-muted-foreground">{$_('import.parametersImported')}: </span>
            <span class="font-medium">{preview.parametersImported}</span>
          </div>
        </div>

        {#if preview.warnings.length > 0}
          <div class="space-y-1">
            <div class="flex items-center gap-1 text-sm font-medium">
              <TriangleAlert class="h-4 w-4 text-yellow-600" />
              {$_('import.warnings')}
            </div>
            <ul class="list-disc pl-5 text-xs text-muted-foreground">
              {#each preview.warnings as warning (warning)}
                <li>{warning}</li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
    {/if}

    <div class="flex justify-end gap-2">
      <Button variant="outline" onclick={() => handleOpenChange(false)}>
        {$_('import.cancel')}
      </Button>
      <Button
        data-testid="import-confirm"
        onclick={handleConfirm}
        disabled={!preview || isImporting}
        class="gap-2"
      >
        <FileUp class="h-4 w-4" />
        {isImporting ? $_('import.importing') : $_('import.confirm')}
      </Button>
    </div>
  </div>
</Dialog>
