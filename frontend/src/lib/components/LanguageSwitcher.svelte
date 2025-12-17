<script lang="ts">
  import { Globe } from 'lucide-svelte'
  import { locale } from 'svelte-i18n'
  import Button from './ui/button.svelte'

  let isPending = $state(false)

  function toggleLanguage() {
    const currentLocale = $locale || 'en'
    // Toggle between en and zh-CN
    const newLocale = currentLocale.startsWith('zh') ? 'en' : 'zh-CN'

    // Set the locale directly
    locale.set(newLocale)

    // Also set cookie for persistence
    document.cookie = `locale=${newLocale}; path=/; max-age=31536000`
  }
</script>

<Button variant="ghost" size="sm" onclick={toggleLanguage} disabled={isPending} class="gap-2">
  <Globe class="h-4 w-4" />
  {$locale && $locale.startsWith('zh') ? 'EN' : '中文'}
</Button>
