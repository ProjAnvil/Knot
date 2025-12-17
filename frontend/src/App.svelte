<script lang="ts">
  import { onMount } from 'svelte'
  import { Toaster } from 'svelte-sonner'
  import { _, isLoading } from 'svelte-i18n'
  import Sidebar from './lib/components/Sidebar.svelte'
  import DocViewer from './lib/components/DocViewer.svelte'
  import type { GroupWithApis, ApiData } from './lib/types'
  import { getGroupsWithApis, getApi } from './lib/api'
  import './lib/i18n'
  import './app.css'

  let groups = $state<GroupWithApis[]>([])
  let selectedApi = $state<ApiData | null>(null)
  let selectedApiId = $state<number | undefined>(undefined)
  let selectedGroupId = $state<number | undefined>(undefined)
  let loading = $state(true)
  let i18nReady = $state(false)

  // Wait for i18n to be ready
  $effect(() => {
    if (!$isLoading) {
      i18nReady = true
    }
  })

  async function loadGroups() {
    loading = true
    try {
      const result = await getGroupsWithApis()
      if (result.success && result.data) {
        groups = result.data
      }
    } catch (error) {
      console.error('Failed to load groups:', error)
    } finally {
      loading = false
    }
  }

  async function loadApiData(apiId: number, updateUrl: boolean = true) {
    try {
      const result = await getApi(apiId)
      if (result.success && result.data) {
        const apiData = result.data
        selectedApi = apiData
        selectedApiId = apiId
        
        // Update the API info in groups array without reloading everything
        groups = groups.map(group => ({
          ...group,
          apis: group.apis.map(api => 
            api.id === apiId 
              ? { ...api, name: apiData.name, endpoint: apiData.endpoint, note: apiData.note }
              : api
          )
        }))
        
        // Set the group ID to auto-expand in sidebar
        if (apiData.group) {
          selectedGroupId = apiData.group.id
        } else if (apiData.groupId) {
          selectedGroupId = apiData.groupId
        }
        
        // Update URL without reloading the page
        if (updateUrl) {
          const url = new URL(window.location.href)
          url.searchParams.set('api', apiId.toString())
          window.history.pushState({}, '', url.toString())
        }
      }
    } catch (error) {
      console.error('Failed to load API:', error)
    }
  }

  function handleApiSelect(apiId: number) {
    loadApiData(apiId, true)
  }

  function handleDataChange() {
    // Only reload the current API data, not the entire groups list
    if (selectedApiId) {
      loadApiData(selectedApiId, false)
    }
  }
  
  function handleStructuralChange() {
    // For structural changes (create, delete, move), reload everything
    loadGroups()
    if (selectedApiId) {
      loadApiData(selectedApiId, false)
    }
  }

  // Handle browser back/forward buttons
  function handlePopState() {
    const params = new URLSearchParams(window.location.search)
    const apiId = params.get('api')
    if (apiId) {
      const id = parseInt(apiId, 10)
      if (!isNaN(id)) {
        loadApiData(id, false)
      }
    } else {
      selectedApi = null
      selectedApiId = undefined
    }
  }

  onMount(() => {
    loadGroups()
    
    // Check if there's an API ID in the URL
    const params = new URLSearchParams(window.location.search)
    const apiId = params.get('api')
    if (apiId) {
      const id = parseInt(apiId, 10)
      if (!isNaN(id)) {
        loadApiData(id, false)
      }
    }
    
    // Listen for browser navigation
    window.addEventListener('popstate', handlePopState)
    
    return () => {
      window.removeEventListener('popstate', handlePopState)
    }
  })
</script>

<Toaster />

{#if !i18nReady}
  <div class="flex h-screen items-center justify-center">
    <div class="text-muted-foreground">Loading...</div>
  </div>
{:else}
  <div class="flex h-screen overflow-hidden">
    <aside class="w-[30%] border-r bg-muted/10 overflow-y-auto">
      {#if loading}
        <div class="flex h-full items-center justify-center text-muted-foreground">
          {$_('common.loading')}
        </div>
      {:else}
        <Sidebar
          {groups}
          {selectedApiId}
          {selectedGroupId}
          onApiSelect={handleApiSelect}
          onDataChange={handleStructuralChange}
        />
      {/if}
    </aside>
    <main class="w-[70%] overflow-y-auto">
      {#if selectedApi}
        <DocViewer apiData={selectedApi} onDataChange={handleDataChange} onStructuralChange={handleStructuralChange} />
      {:else}
        <div class="flex h-full items-center justify-center text-muted-foreground">
          Select an API to view details
        </div>
      {/if}
    </main>
  </div>
{/if}
