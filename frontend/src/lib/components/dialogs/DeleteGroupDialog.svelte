<script lang="ts">
  import { toast } from 'svelte-sonner'
  import { _ } from 'svelte-i18n'
  import Button from '../ui/button.svelte'
  import AlertDialog from '../ui/alert-dialog.svelte'
  import { deleteGroup } from '$lib/api'

  let {
    groupId,
    groupName,
    apiCount,
    open = $bindable(false),
    onOpenChange,
    onSuccess
  }: {
    groupId: number
    groupName: string
    apiCount: number
    open?: boolean
    onOpenChange?: (open: boolean) => void
    onSuccess?: () => void
  } = $props()

  let isDeleting = $state(false)

  async function handleDelete() {
    isDeleting = true
    const result = await deleteGroup(groupId)
    isDeleting = false

    if (result.success) {
      toast.success($_('group.deleteSuccess'))
      open = false
      onOpenChange?.(false)
      onSuccess?.()
    } else {
      toast.error(result.error || $_('group.deleteError'))
    }
  }
</script>

<AlertDialog bind:open {onOpenChange}>
  <div class="space-y-4">
    <div>
      <h2 class="text-lg font-semibold">{$_('group.deleteTitle')}</h2>
      <p class="text-sm text-muted-foreground mt-2">
        {$_('group.deleteDescription', { values: { name: groupName, count: apiCount } })}
        <br />
        <strong class="text-destructive">{$_('group.deleteWarning')}</strong>
      </p>
    </div>
    <div class="flex justify-end gap-2">
      <Button variant="outline" disabled={isDeleting} onclick={() => { open = false; onOpenChange?.(false) }}>
        {$_('common.cancel')}
      </Button>
      <Button
        variant="destructive"
        disabled={isDeleting}
        onclick={handleDelete}
      >
        {isDeleting ? $_('common.deleting') : $_('common.delete')}
      </Button>
    </div>
  </div>
</AlertDialog>
