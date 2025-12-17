<script lang="ts">
  import { Loader2 } from 'lucide-svelte'
  import { toast } from 'svelte-sonner'
  import { _ } from 'svelte-i18n'
  import Button from '../ui/button.svelte'
  import Dialog from '../ui/dialog.svelte'
  import Input from '../ui/input.svelte'
  import Label from '../ui/label.svelte'
  import { renameGroup } from '$lib/api'

  let {
    groupId,
    currentName,
    open = $bindable(false),
    onOpenChange,
    onSuccess
  }: {
    groupId: number
    currentName: string
    open?: boolean
    onOpenChange?: (open: boolean) => void
    onSuccess?: () => void
  } = $props()

  let newName = $state(currentName)
  let isLoading = $state(false)

  $effect(() => {
    newName = currentName
  })

  async function handleRename() {
    if (!newName.trim()) {
      toast.error($_('group.placeholder'))
      return
    }

    if (newName === currentName) {
      open = false
      onOpenChange?.(false)
      return
    }

    isLoading = true
    try {
      const result = await renameGroup(groupId, newName)
      if (result.success) {
        toast.success($_('group.renameSuccess'))
        open = false
        onOpenChange?.(false)
        onSuccess?.()
      } else {
        toast.error(result.error || $_('group.renameError'))
      }
    } catch (error) {
      console.error('Failed to rename group:', error)
      toast.error($_('group.renameError'))
    } finally {
      isLoading = false
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !isLoading) {
      handleRename()
    }
  }
</script>

<Dialog bind:open {onOpenChange}>
  <div class="space-y-4">
    <div>
      <h2 class="text-lg font-semibold">{$_('group.renameGroup')}</h2>
      <p class="text-sm text-muted-foreground mt-1">{$_('group.renameDescription')}</p>
    </div>
    <div class="space-y-2">
      <Label for="name">{$_('group.name')}</Label>
      <Input
        id="name"
        bind:value={newName}
        onkeydown={handleKeyDown}
        disabled={isLoading}
      />
    </div>
    <div class="flex justify-end gap-2">
      <Button variant="outline" onclick={() => { open = false; onOpenChange?.(false) }} disabled={isLoading}>
        {$_('common.cancel')}
      </Button>
      <Button onclick={handleRename} disabled={isLoading}>
        {#if isLoading}
          <Loader2 class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        {$_('common.save')}
      </Button>
    </div>
  </div>
</Dialog>
