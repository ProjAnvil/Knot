<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { toast } from 'svelte-sonner'
  import { AlertCircle, Check, Code2, Edit2, Save, X } from 'lucide-svelte'
  import Button from '../ui/button.svelte'
  import Textarea from '../ui/textarea.svelte'

  let {
    title,
    json,
    onSave
  }: {
    title: string
    json: Record<string, unknown>
    onSave: (newJson: Record<string, unknown>) => Promise<{ success: boolean; error?: string }>
  } = $props()

  let isEditing = $state(false)
  let editValue = $state('')
  let validationError = $state<string | null>(null)
  let isSaving = $state(false)

  function handleEdit() {
    editValue = JSON.stringify(json, null, 2)
    validationError = null
    isEditing = true
  }

  function handleCancel() {
    editValue = ''
    validationError = null
    isEditing = false
  }

  function validateJson(jsonString: string): { valid: boolean; parsed?: Record<string, unknown>; error?: string } {
    try {
      const parsed = JSON.parse(jsonString)
      if (typeof parsed !== 'object' || parsed === null) {
        return { valid: false, error: 'JSON must be an object' }
      }
      return { valid: true, parsed }
    } catch (error) {
      return {
        valid: false,
        error: error instanceof Error ? error.message : 'Invalid JSON format',
      }
    }
  }

  async function handleSave() {
    const validation = validateJson(editValue)

    if (!validation.valid || !validation.parsed) {
      validationError = validation.error || $_('json.invalid')
      toast.error(`${$_('json.invalid')}: ${validation.error}`)
      return
    }

    isSaving = true
    const result = await onSave(validation.parsed)
    isSaving = false

    if (result.success) {
      toast.success($_('json.updateSuccess'))
      isEditing = false
      validationError = null
    } else {
      toast.error(result.error || $_('json.updateError'))
      validationError = result.error || $_('json.updateError')
    }
  }

  // Real-time validation while typing
  function handleChange(value: string) {
    editValue = value
    const validation = validateJson(value)
    validationError = validation.valid ? null : validation.error || 'Invalid JSON'
  }

  // Render JSON with syntax highlighting
  function renderJsonWithHighlight(jsonString: string) {
    const tokens: Array<{ type: string; value: string }> = []

    // Simple tokenizer
    const regex = /"[^"]*"(?=\s*:)|"[^"]*"|true|false|null|-?\d+\.?\d*|[{}[\],]/g
    let lastIndex = 0
    let match: RegExpExecArray | null

    while ((match = regex.exec(jsonString)) !== null) {
      // Add whitespace before token
      if (match.index > lastIndex) {
        tokens.push({ type: 'whitespace', value: jsonString.substring(lastIndex, match.index) })
      }

      const token = match[0]

      // Determine token type and color
      if (token.startsWith('"') && jsonString[regex.lastIndex] === ':') {
        // Object key
        tokens.push({ type: 'key', value: token })
      } else if (token.startsWith('"')) {
        // String value
        tokens.push({ type: 'string', value: token })
      } else if (token === 'true' || token === 'false') {
        // Boolean
        tokens.push({ type: 'boolean', value: token })
      } else if (token === 'null') {
        // Null
        tokens.push({ type: 'null', value: token })
      } else if (/^-?\d+\.?\d*$/.test(token)) {
        // Number
        tokens.push({ type: 'number', value: token })
      } else {
        // Punctuation
        tokens.push({ type: 'punctuation', value: token })
      }

      lastIndex = regex.lastIndex
    }

    // Add remaining text
    if (lastIndex < jsonString.length) {
      tokens.push({ type: 'whitespace', value: jsonString.substring(lastIndex) })
    }

    return tokens
  }

  const highlightedTokens = $derived(renderJsonWithHighlight(JSON.stringify(json, null, 2)))
</script>

<div class="space-y-2">
  <div class="flex items-center justify-between">
    <h3 class="text-lg font-semibold flex items-center gap-2">
      <Code2 class="h-5 w-5 text-purple-600" />
      {title}
    </h3>
    {#if !isEditing}
      <Button variant="outline" size="sm" onclick={handleEdit} class="gap-2">
        <Edit2 class="h-4 w-4" />
        {$_('common.edit')}
      </Button>
    {:else}
      <div class="flex gap-2">
        <Button variant="outline" size="sm" onclick={handleCancel} class="gap-2" disabled={isSaving}>
          <X class="h-4 w-4" />
          {$_('common.cancel')}
        </Button>
        <Button
          variant="default"
          size="sm"
          onclick={handleSave}
          class="gap-2"
          disabled={!!validationError || isSaving}
        >
          <Save class="h-4 w-4" />
          {isSaving ? $_('common.saving') : $_('common.save')}
        </Button>
      </div>
    {/if}
  </div>

  {#if isEditing}
    <div class="space-y-2">
      <Textarea
        bind:value={editValue}
        oninput={(e) => handleChange(e.currentTarget.value)}
        class="font-mono text-sm min-h-[300px] resize-y"
        placeholder="Enter JSON..."
      />
      {#if validationError}
        <div class="flex items-start gap-2 text-sm text-destructive">
          <AlertCircle class="h-4 w-4 mt-0.5 shrink-0" />
          <span>{validationError}</span>
        </div>
      {:else}
        <div class="flex items-center gap-2 text-sm text-green-600">
          <Check class="h-4 w-4" />
          <span>Valid JSON</span>
        </div>
      {/if}
    </div>
  {:else}
    <div class="relative rounded-md border bg-muted/50">
      <pre class="overflow-x-auto p-4"><code class="relative rounded font-mono text-sm">{#each highlightedTokens as token}{#if token.type === 'key'}<span class="text-sky-600 dark:text-sky-400">{token.value}</span>{:else if token.type === 'string'}<span class="text-emerald-600 dark:text-emerald-400">{token.value}</span>{:else if token.type === 'boolean'}<span class="text-violet-600 dark:text-violet-400">{token.value}</span>{:else if token.type === 'null'}<span class="text-slate-500 dark:text-slate-400">{token.value}</span>{:else if token.type === 'number'}<span class="text-amber-600 dark:text-amber-400">{token.value}</span>{:else if token.type === 'punctuation'}<span class="text-slate-600 dark:text-slate-400">{token.value}</span>{:else}{token.value}{/if}{/each}</code></pre>
    </div>
  {/if}
</div>
