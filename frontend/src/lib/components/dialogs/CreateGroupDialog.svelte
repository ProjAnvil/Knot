<script lang="ts">
  import { Plus } from 'lucide-svelte'
  import { toast } from 'svelte-sonner'
  import { _ } from 'svelte-i18n'
  import Button from '../ui/button.svelte'
  import Dialog from '../ui/dialog.svelte'
  import Input from '../ui/input.svelte'
  import Label from '../ui/label.svelte'
  import { createGroup } from '$lib/api'

  let {
    onSuccess
  }: {
    onSuccess?: () => void
  } = $props()

  let open = $state(false)
  let name = $state('')
  let loading = $state(false)

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault()

    if (!name.trim()) {
      toast.error($_('group.createError'))
      return
    }

    loading = true

    const result = await createGroup(name)

    if (result.success) {
      toast.success($_('group.createSuccess'))
      open = false
      name = ''
      onSuccess?.()
    } else {
      toast.error(result.error || $_('group.createError'))
    }
    loading = false
  }
</script>

<Dialog bind:open>
  {#snippet trigger()}
    <Button variant="ghost" size="icon" title={$_('sidebar.createGroup')}>
      <Plus class="h-4 w-4" />
    </Button>
  {/snippet}

  <div class="space-y-4">
    <div class="space-y-2">
      <h2 class="text-lg font-semibold">{$_('group.create')}</h2>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div class="space-y-2">
        <Label for="name">{$_('group.name')}</Label>
        <Input
          id="name"
          bind:value={name}
          placeholder={$_('group.placeholder')}
          required
        />
      </div>
      <div class="flex justify-end gap-2">
        <Button type="button" variant="outline" onclick={() => (open = false)}>
          {$_('common.cancel')}
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? $_('common.saving') : $_('common.create')}
        </Button>
      </div>
    </form>
  </div>
</Dialog>
