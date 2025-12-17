<script lang="ts">
  import { ChevronDown, ChevronRight, Download, FileDown } from 'lucide-svelte'
  import { toast } from 'svelte-sonner'
  import { _ } from 'svelte-i18n'
  import Button from '../ui/button.svelte'
  import Checkbox from '../ui/checkbox.svelte'
  import Dialog from '../ui/dialog.svelte'
  import type { GroupWithApis } from '$lib/types'

  let {
    groups
  }: {
    groups: GroupWithApis[]
  } = $props()

  const API_BASE = '/api'

  let open = $state(false)
  let selectedGroups = $state<Set<number>>(new Set())
  let selectedApis = $state<Set<number>>(new Set())
  let expandedGroups = $state<Set<number>>(new Set())
  let isExporting = $state(false)

  // Derived state for selected count
  let selectedCount = $derived(selectedApis.size)

  function toggleGroup(groupId: number) {
    if (expandedGroups.has(groupId)) {
      expandedGroups.delete(groupId)
    } else {
      expandedGroups.add(groupId)
    }
    expandedGroups = new Set(expandedGroups)
  }

  function handleGroupCheck(groupId: number, checked: boolean) {
    const group = groups.find((g) => g.id === groupId)
    if (!group) return

    if (checked) {
      selectedGroups.add(groupId)
      // Add all APIs in this group
      group.apis.forEach((api) => selectedApis.add(api.id))
    } else {
      selectedGroups.delete(groupId)
      // Remove all APIs in this group
      group.apis.forEach((api) => selectedApis.delete(api.id))
    }

    selectedGroups = new Set(selectedGroups)
    selectedApis = new Set(selectedApis)
  }

  function handleApiCheck(groupId: number, apiId: number, checked: boolean) {
    const group = groups.find((g) => g.id === groupId)
    if (!group) return

    if (checked) {
      selectedApis.add(apiId)
      // Check if all APIs in group are selected
      const allSelected = group.apis.every((api) => api.id === apiId || selectedApis.has(api.id))
      if (allSelected) {
        selectedGroups.add(groupId)
      }
    } else {
      selectedApis.delete(apiId)
      selectedGroups.delete(groupId)
    }

    selectedGroups = new Set(selectedGroups)
    selectedApis = new Set(selectedApis)
  }

  function isGroupChecked(groupId: number): boolean {
    return selectedGroups.has(groupId)
  }

  function isGroupIndeterminate(groupId: number): boolean {
    const group = groups.find((g) => g.id === groupId)
    if (!group) return false

    const selectedApisInGroup = group.apis.filter((api) => selectedApis.has(api.id)).length

    return selectedApisInGroup > 0 && selectedApisInGroup < group.apis.length
  }

  function handleSelectAll() {
    const allGroups = new Set(groups.map((g) => g.id))
    const allApis = new Set(groups.flatMap((g) => g.apis.map((api) => api.id)))
    selectedGroups = allGroups
    selectedApis = allApis
  }

  function handleClearAll() {
    selectedGroups = new Set()
    selectedApis = new Set()
  }

  async function handleExport() {
    if (selectedApis.size === 0) {
      toast.error($_('export.noSelection'))
      return
    }

    isExporting = true

    try {
      const response = await fetch(`${API_BASE}/export`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          apiIds: Array.from(selectedApis),
        }),
      })

      if (!response.ok) {
        throw new Error('Export failed')
      }

      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `api-docs-${Date.now()}.html`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)

      toast.success($_('export.success'))
      open = false
    } catch (error) {
      console.error('Export error:', error)
      toast.error($_('export.error'))
    } finally {
      isExporting = false
    }
  }
</script>

<Dialog bind:open>
  {#snippet trigger()}
    <Button variant="outline" size="sm" class="gap-2">
      <Download class="h-4 w-4" />
      {$_('export.title')}
    </Button>
  {/snippet}

  <div class="space-y-4">
    <div class="space-y-2">
      <h2 class="text-lg font-semibold">{$_('export.selectItems')}</h2>
    </div>

    <!-- Action buttons -->
    <div class="flex gap-2">
      <Button variant="outline" size="sm" onclick={handleSelectAll}>
        {$_('export.selectAll')}
      </Button>
      <Button variant="outline" size="sm" onclick={handleClearAll}>
        {$_('export.clearAll')}
      </Button>
      <div class="ml-auto text-sm text-muted-foreground">
        {selectedCount} {$_('export.apis')}
      </div>
    </div>

    <!-- Selection tree -->
    <div class="h-[400px] overflow-y-auto border rounded-md p-4">
      <div class="space-y-2">
        {#each groups as group (group.id)}
          <div class="space-y-1">
            <div class="flex items-center gap-2 p-2 hover:bg-muted rounded-md">
              <Button
                variant="ghost"
                size="sm"
                class="h-6 w-6 p-0"
                onclick={() => toggleGroup(group.id)}
              >
                {#if expandedGroups.has(group.id)}
                  <ChevronDown class="h-4 w-4" />
                {:else}
                  <ChevronRight class="h-4 w-4" />
                {/if}
              </Button>
              <Checkbox
                checked={isGroupChecked(group.id)}
                onCheckedChange={(checked) => handleGroupCheck(group.id, checked)}
                class={isGroupIndeterminate(group.id) ? 'opacity-50' : ''}
              />
              <span class="font-medium">{group.name}</span>
              <span class="text-xs text-muted-foreground ml-auto">
                ({group.apis.length} {$_('export.apis')})
              </span>
            </div>

            {#if expandedGroups.has(group.id)}
              <div class="ml-10 space-y-1">
                {#each group.apis as api (api.id)}
                  <div class="flex items-center gap-2 p-2 hover:bg-muted rounded-md">
                    <Checkbox
                      checked={selectedApis.has(api.id)}
                      onCheckedChange={(checked) => handleApiCheck(group.id, api.id, checked)}
                    />
                    <span class="text-sm">{api.name}</span>
                    <span class="text-xs text-muted-foreground ml-auto">
                      {api.method} {api.endpoint}
                    </span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <!-- Export button -->
    <div class="flex justify-end gap-2">
      <Button variant="outline" onclick={() => (open = false)}>
        {$_('common.cancel')}
      </Button>
      <Button
        onclick={handleExport}
        disabled={selectedApis.size === 0 || isExporting}
        class="gap-2"
      >
        <FileDown class="h-4 w-4" />
        {isExporting ? $_('export.exporting') : $_('export.exportHtml')}
      </Button>
    </div>
  </div>
</Dialog>
