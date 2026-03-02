import { defineConfig } from 'vitest/config'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
	plugins: [svelte({ hot: !process.env.VITEST })],
	test: {
		globals: true,
		environment: 'happy-dom',
		include: ['src/**/*.{test,spec}.{js,ts}'],
		exclude: ['node_modules', 'dist'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'json', 'html'],
			exclude: [
				'node_modules/',
				'src/**/*.test.{js,ts}',
				'src/**/*.spec.{js,ts}',
				'src/**/*.d.ts',
				'src/vite-env.d.ts'
			]
		},
		setupFiles: ['./vitest.setup.ts']
	}
})
