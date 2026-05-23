/**
 * Cookie utility functions with proper security attributes
 */

/**
 * Set a cookie with proper security attributes
 * @param name - Cookie name
 * @param value - Cookie value
 * @param days - Number of days until expiry (default: 365)
 */
export function setCookie(name: string, value: string, days = 365): void {
	const maxAge = days * 24 * 60 * 60
	document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=${maxAge}; SameSite=Strict`
}

/**
 * Get a cookie value by name
 * @param name - Cookie name
 * @returns The cookie value or null if not set
 */
export function getCookie(name: string): string | null {
	const match = document.cookie
		.split('; ')
		.find((row) => row.startsWith(`${name}=`))

	if (match) {
		const value = match.split('=')[1]
		return value ? decodeURIComponent(value) : null
	}

	return null
}

/**
 * Delete a cookie by name
 * @param name - Cookie name to delete
 */
export function deleteCookie(name: string): void {
	document.cookie = `${name}=; path=/; max-age=0; SameSite=Strict`
}
