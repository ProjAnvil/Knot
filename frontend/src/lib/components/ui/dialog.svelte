<script lang="ts">
  import { X } from 'lucide-svelte'
  import { cn } from '$lib/utils'

  let {
    open = $bindable(false),
    onOpenChange,
    trigger,
    children,
    class: className = ''
  }: {
    open?: boolean
    onOpenChange?: (open: boolean) => void
    trigger?: any
    children: any
    class?: string
  } = $props()

  function handleClose() {
    open = false
    onOpenChange?.(false)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      handleClose()
    }
  }

  function openDialog() {
    open = true
    onOpenChange?.(true)
  }
</script>

{#if trigger}
  <div onclick={openDialog} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && openDialog()}>
    {@render trigger()}
  </div>
{/if}

{#if open}
  <div
    class="fixed inset-0 z-50 bg-black/80"
    onclick={handleClose}
    onkeydown={handleKeydown}
    role="button"
    tabindex="-1"
  >
    <div
      class={cn(
        "fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 sm:rounded-lg",
        className
      )}
      onclick={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
    >
      <button
        class="absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
        onclick={handleClose}
      >
        <X class="h-4 w-4" />
        <span class="sr-only">Close</span>
      </button>
      {@render children()}
    </div>
  </div>
{/if}
