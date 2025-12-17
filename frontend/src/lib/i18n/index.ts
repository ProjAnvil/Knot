import { getLocaleFromNavigator, init, register } from 'svelte-i18n'

// Register locale files
register('en', () => import('./locales/en.json'))
register('zh-CN', () => import('./locales/zh-CN.json'))
register('zh', () => import('./locales/zh-CN.json')) // Fallback for 'zh'

// Get locale from cookie or browser
function getInitialLocale() {
	// Check for locale cookie first
	const localeCookie = document.cookie
		.split('; ')
		.find((row) => row.startsWith('locale='))
		?.split('=')[1]

	if (localeCookie) {
		return localeCookie
	}

	// Fallback to browser locale
	return getLocaleFromNavigator()
}

// Initialize with cookie, browser locale, or fallback to English
init({
	fallbackLocale: 'en',
	initialLocale: getInitialLocale(),
})
