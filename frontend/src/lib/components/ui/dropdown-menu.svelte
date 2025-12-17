<script lang="ts">
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'

  let {
    open = $bindable(false),
    trigger,
    content
  }: {
    open?: boolean
    trigger: Snippet
    content: Snippet
  } = $props()

  function toggleMenu() {
    open = !open
  }

  function close() {
    open = false
  }
</script>

<div class="relative inline-block">
  <div onclick={toggleMenu} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && toggleMenu()}>
    {@render trigger()}
  </div>

  {#if open}
    <div
      class="fixed inset-0 z-40"
      onclick={close}
      role="button"
      tabindex="-1"
    ></div>
    <div
      class="absolute right-0 z-50 mt-2 min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md"
      role="menu"
    >
      {@render content()}
    </div>
  {/if}
</div>
