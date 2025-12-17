<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { toast } from 'svelte-sonner'
  import { Trash2 } from 'lucide-svelte'
  import Button from '../ui/button.svelte'
  import AlertDialog from '../ui/alert-dialog.svelte'
  import { deleteApi } from '$lib/api'

  let {
    apiId,
    apiName,
    onDelete
  }: {
    apiId: number
    apiName: string
    onDelete?: () => void
  } = $props()

  let open = $state(false)
  let isDeleting = $state(false)

  async function handleDelete() {
    isDeleting = true
    const result = await deleteApi(apiId)
    isDeleting = false

    if (result.success) {
      toast.success($_('api.deleteSuccess'))
      open = false
      onDelete?.()
    } else {
      toast.error(result.error || $_('api.deleteError'))
    }
  }
</script>

<Button
  variant="ghost"
  size="sm"
  class="gap-2 text-destructive hover:text-destructive hover:bg-destructive/10"
  onclick={(e: MouseEvent) => {
    e.stopPropagation()
    open = true
  }}
>
  <Trash2 class="h-4 w-4" />
  {$_('common.delete')}
</Button>

<AlertDialog bind:open>
  <div class="space-y-4">
    <div class="space-y-2">
      <h2 class="text-lg font-semibold">{$_('api.deleteTitle')}</h2>
      <p class="text-sm text-muted-foreground">
        {$_('api.deleteDescription', { values: { name: apiName } })}
        <br />
        <strong class="text-destructive">{$_('api.deleteWarning')}</strong>
      </p>
    </div>
    <div class="flex gap-2 justify-end">
      <Button variant="outline" onclick={() => open = false} disabled={isDeleting}>
        {$_('common.cancel')}
      </Button>
      <Button
        variant="destructive"
        onclick={handleDelete}
        disabled={isDeleting}
      >
        {isDeleting ? $_('common.deleting') : $_('common.delete')}
      </Button>
    </div>
  </div>
</AlertDialog>
