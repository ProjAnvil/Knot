<script lang="ts">
  import { cn } from '$lib/utils'

  type SelectOption = {
    value: string
    label: string
  }

  let {
    class: className = '',
    value = $bindable(''),
    onValueChange,
    options = [],
    ...props
  }: {
    class?: string
    value?: string
    onValueChange?: (value: string) => void
    options?: SelectOption[]
    [key: string]: any
  } = $props()

  function handleChange(event: Event) {
    const target = event.target as HTMLSelectElement
    value = target.value
    onValueChange?.(target.value)
  }
</script>

<select
  bind:value
  onchange={handleChange}
  class={cn(
    'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
    className
  )}
  {...props}
>
  {#each options as option}
    <option value={option.value}>{option.label}</option>
  {/each}
</select>
