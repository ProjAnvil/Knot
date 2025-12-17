<script lang="ts">
  import { ChevronDown, ChevronRight, GripVertical, MoreVertical, Pencil, Trash2 } from 'lucide-svelte'
  import { dndzone } from 'svelte-dnd-action'
  import { toast } from 'svelte-sonner'
  import { _ } from 'svelte-i18n'
  import type { GroupWithApis } from '$lib/types'
  import { updateApiOrders, updateGroupOrders } from '$lib/api'
  import { cn } from '$lib/utils'
  import CreateGroupDialog from './dialogs/CreateGroupDialog.svelte'
  import DeleteGroupDialog from './dialogs/DeleteGroupDialog.svelte'
  import RenameGroupDialog from './dialogs/RenameGroupDialog.svelte'
  import CreateApiDialog from './dialogs/CreateApiDialog.svelte'
  import ExportDialog from './dialogs/ExportDialog.svelte'
  import LanguageSwitcher from './LanguageSwitcher.svelte'
  import DropdownMenu from './ui/dropdown-menu.svelte'

  let {
    groups = [],
    selectedApiId,
    selectedGroupId,
    onApiSelect,
    onDataChange
  }: {
    groups: GroupWithApis[]
    selectedApiId?: number
    selectedGroupId?: number
    onApiSelect?: (apiId: number) => void
    onDataChange?: () => void
  } = $props()

  // Only keep expanded groups in memory (no sessionStorage)
  let expandedGroups = $state<Set<string>>(new Set())
  let renameDialogOpen = $state(false)
  let deleteDialogOpen = $state(false)
  let selectedGroupForAction = $state<{ id: number; name: string; apiCount: number } | null>(null)
  let localGroups = $state<GroupWithApis[]>([...groups])
  let groupDragDisabled = $state(true)
  let apiDragDisabled = $state<Record<number, boolean>>({})
  let hasAutoExpanded = $state(false)

  // Update localGroups when groups change
  $effect(() => {
    localGroups = [...groups]
  })

  // Auto-expand group only once when URL loads with selectedGroupId
  $effect(() => {
    if (selectedGroupId !== undefined && groups.length > 0 && !hasAutoExpanded) {
      const group = groups.find(g => g.id === selectedGroupId)
      if (group) {
        const newSet = new Set(expandedGroups)
        newSet.add(group.name)
        expandedGroups = newSet
        hasAutoExpanded = true
      }
    }
  })

  function toggleGroup(groupName: string) {
    const newSet = new Set(expandedGroups)
    if (newSet.has(groupName)) {
      newSet.delete(groupName)
    } else {
      newSet.add(groupName)
    }
    expandedGroups = newSet
  }

  function selectApi(apiId: number) {
    onApiSelect?.(apiId)
  }

  function handleDndConsider(groupId: number) {
    return (e: CustomEvent<any>) => {
      const group = localGroups.find(g => g.id === groupId)
      if (!group) return

      const newGroups = localGroups.map(g =>
        g.id === groupId ? { ...g, apis: e.detail.items } : g
      )
      localGroups = newGroups
    }
  }

  function handleDndFinalize(groupId: number) {
    return async (e: CustomEvent<any>) => {
      const group = localGroups.find(g => g.id === groupId)
      if (!group) return

      const newApis = e.detail.items
      const newGroups = localGroups.map(g =>
        g.id === groupId ? { ...g, apis: newApis } : g
      )
      localGroups = newGroups

      // Update order values
      const apiOrders = newApis.map((api: any, index: number) => ({
        id: api.id,
        order: index
      }))

      const result = await updateApiOrders(apiOrders)
      if (!result.success) {
        toast.error(result.error || 'Failed to update API order')
        localGroups = [...groups]
      } else {
        onDataChange?.()
      }
    }
  }

  function openRenameDialog(group: GroupWithApis) {
    selectedGroupForAction = { id: group.id, name: group.name, apiCount: group.apis.length }
    renameDialogOpen = true
  }

  function openDeleteDialog(group: GroupWithApis) {
    selectedGroupForAction = { id: group.id, name: group.name, apiCount: group.apis.length }
    deleteDialogOpen = true
  }

  function handleGroupDndConsider(e: CustomEvent<any>) {
    localGroups = e.detail.items
  }

  async function handleGroupDndFinalize(e: CustomEvent<any>) {
    const newGroups = e.detail.items
    localGroups = newGroups

    // Update order values
    const groupOrders = newGroups.map((group: any, index: number) => ({
      id: group.id,
      order: index
    }))

    const result = await updateGroupOrders(groupOrders)
    if (!result.success) {
      toast.error(result.error || 'Failed to update group order')
      localGroups = [...groups]
    } else {
      onDataChange?.()
    }
  }
</script>

<div class="flex flex-col h-full">
  <div class="p-4 border-b">
    <div class="flex justify-between items-center mb-2">
      <span class="font-bold">{$_('sidebar.title')}</span>
      <div class="flex gap-1">
        <LanguageSwitcher />
        <CreateGroupDialog onSuccess={onDataChange} />
      </div>
    </div>
    <div class="mt-2">
      <ExportDialog {groups} />
    </div>
  </div>

  <div class="flex-1 overflow-y-auto p-2">
    {#if localGroups.length === 0}
      <div class="text-center text-muted-foreground text-sm p-4">
        {$_('sidebar.noGroups')}
      </div>
    {:else}
      <div
        use:dndzone={{ items: localGroups, flipDurationMs: 200, type: 'group', dragDisabled: groupDragDisabled }}
        onconsider={handleGroupDndConsider}
        onfinalize={handleGroupDndFinalize}
      >
        {#each localGroups as group (group.id)}
          <div class="mb-2">
            <div
              class={cn(
                'flex items-center p-2 cursor-pointer hover:bg-muted rounded-md select-none gap-2 w-full',
                expandedGroups.has(group.name) && 'bg-muted'
              )}
              onclick={(e) => {
                if (!e.defaultPrevented) {
                  toggleGroup(group.name)
                }
              }}
              role="button"
              tabindex="0"
              onkeydown={(e) => {
                if (e.key === 'Enter') {
                  toggleGroup(group.name)
                }
              }}
            >
              <div 
                class="cursor-grab active:cursor-grabbing p-1 hover:bg-accent rounded shrink-0"
                onmouseenter={() => groupDragDisabled = false}
                onmouseleave={() => groupDragDisabled = true}
                onmousedown={(e) => e.stopPropagation()}
                onclick={(e) => e.stopPropagation()}
                onkeydown={(e) => e.stopPropagation()}
                role="button"
                tabindex="-1"
                aria-label="Drag handle"
              >
                <GripVertical class="h-4 w-4 text-muted-foreground" />
              </div>
              <div class="flex-1 flex items-center gap-2 min-w-0">
                {#if expandedGroups.has(group.name)}
                  <ChevronDown class="h-4 w-4 shrink-0" />
                {:else}
                  <ChevronRight class="h-4 w-4 shrink-0" />
                {/if}
                <span class="font-medium flex-1 min-w-0 truncate">{group.name}</span>
                <span class="text-xs text-muted-foreground shrink-0">{group.apis.length}</span>
              </div>

              <div 
                onclick={(e) => e.stopPropagation()}
                onkeydown={(e) => e.stopPropagation()}
                role="none"
              >
                <DropdownMenu>
                  {#snippet trigger()}
                    <button
                      class="h-6 w-6 flex items-center justify-center hover:bg-accent rounded-sm"
                    >
                      <MoreVertical class="h-4 w-4" />
                    </button>
                  {/snippet}

                  {#snippet content()}
                  <button
                    class="flex w-full items-center px-2 py-1.5 text-sm hover:bg-accent rounded-sm"
                    onclick={(e) => {
                      e.stopPropagation()
                      openRenameDialog(group)
                    }}
                  >
                    <Pencil class="mr-2 h-4 w-4" />
                    {$_('group.renameGroup')}
                  </button>
                  <button
                    class="flex w-full items-center px-2 py-1.5 text-sm text-destructive hover:bg-accent rounded-sm"
                    onclick={(e) => {
                      e.stopPropagation()
                      openDeleteDialog(group)
                    }}
                  >
                    <Trash2 class="mr-2 h-4 w-4" />
                    {$_('group.deleteTitle')}
                  </button>
                  {/snippet}
                </DropdownMenu>
              </div>
            </div>

            {#if expandedGroups.has(group.name)}
              <div class="ml-6 mt-1">
                <div
                  use:dndzone={{ items: group.apis, flipDurationMs: 200, type: `api-${group.id}`, dragDisabled: apiDragDisabled[group.id] ?? true }}
                  onconsider={handleDndConsider(group.id)}
                  onfinalize={handleDndFinalize(group.id)}
                >
                  {#each group.apis as api (api.id)}
                    <div
                      class={cn(
                        'flex items-center gap-2 p-2 text-sm rounded-md select-none mb-1 w-full',
                        selectedApiId === api.id
                          ? 'bg-primary/10 text-primary font-medium'
                          : 'hover:bg-muted cursor-pointer'
                      )}
                      onclick={(e) => {
                        if (!e.defaultPrevented) {
                          selectApi(api.id)
                        }
                      }}
                      role="button"
                      tabindex="0"
                      onkeydown={(e) => {
                        if (e.key === 'Enter') {
                          selectApi(api.id)
                        }
                      }}
                    >
                      <div 
                        class="cursor-grab active:cursor-grabbing p-1 hover:bg-accent rounded shrink-0"
                        onmouseenter={() => apiDragDisabled = { ...apiDragDisabled, [group.id]: false }}
                        onmouseleave={() => apiDragDisabled = { ...apiDragDisabled, [group.id]: true }}
                        onmousedown={(e) => e.stopPropagation()}
                        onclick={(e) => e.stopPropagation()}
                        onkeydown={(e) => e.stopPropagation()}
                        role="button"
                        tabindex="-1"
                        aria-label="Drag handle"
                      >
                        <GripVertical class="h-4 w-4 text-muted-foreground" />
                      </div>
                      <span class="flex-1 min-w-0">
                        {api.name}
                      </span>
                    </div>
                  {/each}
                </div>

                <CreateApiDialog groupName={group.name} onSuccess={onDataChange} />
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

{#if selectedGroupForAction}
  <RenameGroupDialog
    groupId={selectedGroupForAction.id}
    currentName={selectedGroupForAction.name}
    bind:open={renameDialogOpen}
    onSuccess={onDataChange}
  />
  <DeleteGroupDialog
    groupId={selectedGroupForAction.id}
    groupName={selectedGroupForAction.name}
    apiCount={selectedGroupForAction.apiCount}
    bind:open={deleteDialogOpen}
    onSuccess={onDataChange}
  />
{/if}
