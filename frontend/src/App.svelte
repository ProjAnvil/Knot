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
  // Holds the id of the API currently being fetched, so the viewer can show
  // progress instead of looking frozen on a slow connection.
  let loadingApiId = $state<number | null>(null)

  // Fetched API documents, kept for the page lifetime so revisiting an API
  // renders instantly. Entries stay fresh for a short window, which saves a
  // round trip on every click without ever showing stale data for long.
  const API_CACHE_TTL_MS = 5 * 60 * 1000
  const apiCache = new Map<number, { data: ApiData; at: number }>()
  const prefetching = new Set<number>()
  // Monotonic guard: only the most recent selection may write to the viewer.
  let requestSeq = 0
  let activeRequest: AbortController | null = null

  // Wait for i18n to be ready
  $effect(() => {
    if (!$isLoading) {
      i18nReady = true
    }
  })

  async function loadGroups(silent = false) {
    if (!silent) loading = true
    try {
      const result = await getGroupsWithApis()
      if (result.success && result.data) {
        groups = result.data
      }
    } catch (error) {
      console.error('Failed to load groups:', error)
    } finally {
      if (!silent) loading = false
    }
  }

  function applyApiData(apiData: ApiData, updateUrl: boolean) {
    selectedApi = apiData
    selectedApiId = apiData.id

    // Set the group ID to auto-expand in sidebar
    if (apiData.group) {
      selectedGroupId = apiData.group.id
    } else if (apiData.groupId) {
      selectedGroupId = apiData.groupId
    }

    // Update URL without reloading the page
    if (updateUrl) {
      const url = new URL(window.location.href)
      url.searchParams.set('api', apiData.id.toString())
      window.history.pushState({}, '', url.toString())
    }
  }

  // Keep a single sidebar entry in sync after an edit. Rebuilding every group
  // and API object here used to re-render the whole 600+ entry sidebar (and
  // re-run dndzone) on every click.
  function syncSidebarEntry(apiData: ApiData) {
    const groupIndex = groups.findIndex((group) => group.apis.some((api) => api.id === apiData.id))
    if (groupIndex === -1) return

    const group = groups[groupIndex]
    const apiIndex = group.apis.findIndex((api) => api.id === apiData.id)
    const api = group.apis[apiIndex]
    if (api.name === apiData.name && api.endpoint === apiData.endpoint && api.note === apiData.note) {
      return
    }

    const apis = group.apis.slice()
    apis[apiIndex] = { ...api, name: apiData.name, endpoint: apiData.endpoint, note: apiData.note }

    const nextGroups = groups.slice()
    nextGroups[groupIndex] = { ...group, apis }
    groups = nextGroups
  }

  async function loadApiData(apiId: number, updateUrl = true, force = false) {
    // Highlight the clicked entry immediately, even before its document lands.
    selectedApiId = apiId

    // Serve a cached document immediately.
    const entry = apiCache.get(apiId)
    if (entry) {
      applyApiData(entry.data, updateUrl)
      loadingApiId = null
      // Fresh enough: skip the request entirely. This is what makes repeat
      // clicks instant on a slow link.
      if (!force && Date.now() - entry.at < API_CACHE_TTL_MS) {
        return
      }
    } else {
      loadingApiId = apiId
    }

    const seq = ++requestSeq
    // Cancel the in-flight request: on a slow link this frees bandwidth for the
    // API the user actually asked for.
    activeRequest?.abort()
    const controller = new AbortController()
    activeRequest = controller

    try {
      const result = await getApi(apiId, controller.signal)
      if (seq !== requestSeq) return // a newer selection won

      if (result.success && result.data) {
        apiCache.set(apiId, { data: result.data, at: Date.now() })
        applyApiData(result.data, updateUrl)
        syncSidebarEntry(result.data)
      }
    } catch (error) {
      if ((error as Error)?.name !== 'AbortError') {
        console.error('Failed to load API:', error)
      }
    } finally {
      if (seq === requestSeq) {
        loadingApiId = null
      }
    }
  }

  // Warm the cache when the pointer rests on a sidebar entry: on a slow link
  // this turns the subsequent click into an instant render.
  function prefetchApi(apiId: number) {
    if (apiCache.has(apiId) || prefetching.has(apiId) || apiId === selectedApiId) return
    prefetching.add(apiId)
    getApi(apiId)
      .then((result) => {
        if (result.success && result.data) {
          apiCache.set(apiId, { data: result.data, at: Date.now() })
        }
      })
      .catch(() => {})
      .finally(() => prefetching.delete(apiId))
  }

  function handleApiSelect(apiId: number) {
    loadApiData(apiId, true)
  }

  function handleDataChange() {
    // Force a refresh after an edit; the cached copy keeps showing meanwhile.
    if (selectedApiId) {
      loadApiData(selectedApiId, false, true)
    }
  }

  function handleStructuralChange() {
    // For structural changes (create, delete, move), reload everything
    apiCache.clear()
    loadGroups(true)
    if (selectedApiId) {
      loadApiData(selectedApiId, false, true)
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
      activeRequest?.abort()
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
          onApiPrefetch={prefetchApi}
          onDataChange={handleStructuralChange}
        />
      {/if}
    </aside>
    <main class="w-[70%] overflow-y-auto relative">
      {#if loadingApiId !== null && selectedApi?.id !== loadingApiId}
        <div class="sticky top-0 h-0.5 w-full bg-primary/70 animate-pulse z-10" aria-hidden="true"></div>
      {/if}
      {#if selectedApi}
        <DocViewer apiData={selectedApi} onDataChange={handleDataChange} onStructuralChange={handleStructuralChange} />
      {:else if loadingApiId !== null}
        <div class="flex h-full items-center justify-center text-muted-foreground">
          {$_('common.loading')}
        </div>
      {:else}
        <div class="flex h-full items-center justify-center text-muted-foreground">
          {$_('common.selectApiPrompt')}
        </div>
      {/if}
    </main>
  </div>
{/if}
