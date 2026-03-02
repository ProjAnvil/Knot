<script lang="ts">
  import { ChevronDown, ChevronRight, GripVertical, MoreVertical, Pencil, Trash2, Search, X, ArrowUp, ArrowDown } from 'lucide-svelte'
  import { dndzone } from 'svelte-dnd-action'
  import { toast } from 'svelte-sonner'
  import { _ } from 'svelte-i18n'
  import type { GroupWithApis } from '$lib/types'
  import { updateApiOrders, updateGroupOrders } from '$lib/api'
  import { cn } from '$lib/utils'
  import { performSearch, getExpandedGroups, navigateNext, navigatePrevious, isCurrentGroupResult, isCurrentApiResult, type SearchResult } from '$lib/search'
  import CreateGroupDialog from './dialogs/CreateGroupDialog.svelte'
  import DeleteGroupDialog from './dialogs/DeleteGroupDialog.svelte'
  import RenameGroupDialog from './dialogs/RenameGroupDialog.svelte'
  import CreateApiDialog from './dialogs/CreateApiDialog.svelte'
  import ExportDialog from './dialogs/ExportDialog.svelte'
  import LanguageSwitcher from './LanguageSwitcher.svelte'
  import DropdownMenu from './ui/dropdown-menu.svelte'
  import Input from './ui/input.svelte'

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

  // Search state
  let searchQuery = $state('')
  let matchedGroupNames = $state<Set<string>>(new Set())
  let matchedApiIds = $state<Set<number>>(new Set())
  let searchResults = $state<SearchResult[]>([])
  let currentResultIndex = $state(-1)

  // Update localGroups when groups change
  $effect(() => {
    localGroups = [...groups]
  })

  // Reset search results when query changes
  $effect(() => {
    if (searchQuery) {
      // Reset results when query changes, so Enter will trigger new search
      searchResults = []
      currentResultIndex = -1
      matchedGroupNames = new Set()
      matchedApiIds = new Set()
    }
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

  // Search function
  function handleSearch() {
    const searchState = performSearch(searchQuery, groups)

    // If no results found, reset to default state (all groups visible but collapsed)
    if (searchState.results.length === 0) {
      matchedGroupNames = new Set()
      matchedApiIds = new Set()
      searchResults = []
      currentResultIndex = -1
      expandedGroups = new Set()
    } else {
      matchedGroupNames = searchState.matchedGroupNames
      matchedApiIds = searchState.matchedApiIds
      searchResults = searchState.results
      currentResultIndex = 0
      // Only expand groups that have matching results
      expandedGroups = getExpandedGroups(searchState, groups)
      // Navigate to first result
      navigateToResult(0)
    }
  }

  function clearSearch() {
    searchQuery = ''
    matchedGroupNames = new Set()
    matchedApiIds = new Set()
    searchResults = []
    currentResultIndex = -1
  }

  // Navigate to a specific search result
  function navigateToResult(index: number) {
    if (searchResults.length === 0 || index < 0 || index >= searchResults.length) return

    currentResultIndex = index
    const result = searchResults[index]

    if (result.type === 'api') {
      // Select the API
      selectApi(result.id)

      // Scroll into view
      setTimeout(() => {
        const element = document.querySelector(`[data-api-id="${result.id}"]`)
        element?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }, 100)
    } else {
      // For group results, just scroll to the group
      setTimeout(() => {
        const element = document.querySelector(`[data-group-id="${result.id}"]`)
        element?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }, 100)
    }
  }

  function handleNavigatePrevious() {
    if (searchResults.length === 0) return
    const newIndex = navigatePrevious(currentResultIndex, searchResults.length)
    navigateToResult(newIndex)
  }

  function handleNavigateNext() {
    if (searchResults.length === 0) return
    const newIndex = navigateNext(currentResultIndex, searchResults.length)
    navigateToResult(newIndex)
  }

  // Handle search input keydown
  function handleSearchKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      if (searchResults.length > 0) {
        // If already searched, navigate to next/previous based on shift key
        if (e.shiftKey) {
          handleNavigatePrevious()
        } else {
          handleNavigateNext()
        }
      } else {
        handleSearch()
      }
    } else if (e.key === 'Escape') {
      clearSearch()
    }
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
    <div class="mt-2 flex gap-2">
      <div class="relative flex-1">
        <Input
          type="text"
          bind:value={searchQuery}
          onkeydown={handleSearchKeydown}
          placeholder={$_('sidebar.searchPlaceholder')}
          class={searchResults.length > 0 ? 'pr-28' : 'pr-16'}
        />
        <div class="absolute right-1 top-1/2 -translate-y-1/2 flex items-center gap-0.5">
          {#if searchResults.length > 0}
            <span class="text-xs text-muted-foreground px-1 min-w-[2rem] text-center">
              {currentResultIndex + 1}/{searchResults.length}
            </span>
            <button
              onclick={handleNavigatePrevious}
              class="h-7 w-7 flex items-center justify-center hover:bg-accent rounded-sm text-muted-foreground hover:text-foreground"
              type="button"
              title="Previous (Shift+Enter)"
            >
              <ArrowUp class="h-4 w-4" />
            </button>
            <button
              onclick={handleNavigateNext}
              class="h-7 w-7 flex items-center justify-center hover:bg-accent rounded-sm text-muted-foreground hover:text-foreground"
              type="button"
              title="Next (Enter)"
            >
              <ArrowDown class="h-4 w-4" />
            </button>
          {/if}
          {#if searchQuery}
            <button
              onclick={clearSearch}
              class="h-7 w-7 flex items-center justify-center hover:bg-accent rounded-sm text-muted-foreground hover:text-foreground"
              type="button"
              title="Clear"
            >
              <X class="h-4 w-4" />
            </button>
          {/if}
          <button
            onclick={handleSearch}
            class="h-7 w-7 flex items-center justify-center hover:bg-accent rounded-sm text-muted-foreground hover:text-foreground"
            type="button"
            title="Search"
          >
            <Search class="h-4 w-4" />
          </button>
        </div>
      </div>
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
          <div class="mb-2" data-group-id={group.id}>
            <div
              class={cn(
                'flex items-center p-2 cursor-pointer hover:bg-muted rounded-md select-none gap-2 w-full',
                expandedGroups.has(group.name) && 'bg-muted',
                matchedGroupNames.has(group.name) && 'bg-yellow-100 dark:bg-yellow-900/30',
                isCurrentGroupResult(searchResults, currentResultIndex, group.id) && 'ring-2 ring-primary'
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
                      data-api-id={api.id}
                      class={cn(
                        'flex items-center gap-2 p-2 text-sm rounded-md select-none mb-1 w-full',
                        selectedApiId === api.id
                          ? 'bg-primary/10 text-primary font-medium'
                          : 'hover:bg-muted cursor-pointer',
                        matchedApiIds.has(api.id) && 'bg-yellow-100 dark:bg-yellow-900/30',
                        isCurrentApiResult(searchResults, currentResultIndex, api.id) && 'ring-2 ring-primary'
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
