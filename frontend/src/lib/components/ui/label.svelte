<script lang="ts">
  import { cn } from '$lib/utils'

  let {
    class: className = '',
    children,
    ...props
  }: {
    class?: string
    children: any
    [key: string]: any
  } = $props()
</script>

<label
  class={cn(
    'text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70',
    className
  )}
  {...props}
>
  {@render children()}
</label>
