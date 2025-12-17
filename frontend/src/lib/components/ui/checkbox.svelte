<script lang="ts">
  import { cn } from '$lib/utils'

  let {
    class: className = '',
    checked = $bindable(false),
    onCheckedChange,
    disabled = false,
    ...props
  }: {
    class?: string
    checked?: boolean
    onCheckedChange?: (checked: boolean) => void
    disabled?: boolean
    [key: string]: any
  } = $props()

  function handleChange(event: Event) {
    const target = event.target as HTMLInputElement
    checked = target.checked
    onCheckedChange?.(target.checked)
  }
</script>

<input
  type="checkbox"
  bind:checked
  {disabled}
  onchange={handleChange}
  class={cn(
    'h-4 w-4 rounded border border-input bg-background text-primary ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
    className
  )}
  {...props}
/>
