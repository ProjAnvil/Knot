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

<div class="relative w-full overflow-auto">
  <table
    class={cn('w-full caption-bottom text-sm', className)}
    {...props}
  >
    {@render children()}
  </table>
</div>
