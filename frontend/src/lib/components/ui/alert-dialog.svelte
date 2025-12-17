<script lang="ts">
  import { cn } from '$lib/utils'

  let {
    open = $bindable(false),
    onOpenChange,
    children
  }: {
    open?: boolean
    onOpenChange?: (open: boolean) => void
    children: any
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
</script>

{#if open}
  <div
    class="fixed inset-0 z-50 bg-black/80"
    onkeydown={handleKeydown}
    role="button"
    tabindex="-1"
  >
    <div
      class="fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 sm:rounded-lg"
      onclick={(e) => e.stopPropagation()}
      role="alertdialog"
      aria-modal="true"
    >
      {@render children()}
    </div>
  </div>
{/if}
