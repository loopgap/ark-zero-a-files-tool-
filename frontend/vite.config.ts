import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
	plugins: [react()],
	build: {
		outDir: 'build',
		emptyOutDir: true,
		rollupOptions: {
			output: {
				manualChunks(id) {
					if (id.includes('node_modules/react') || id.includes('node_modules/react-dom')) {
						return 'react-vendor';
					}
					if (
						id.includes('node_modules/@codemirror/state') ||
						id.includes('node_modules/@codemirror/view') ||
						id.includes('node_modules/@codemirror/language') ||
						id.includes('node_modules/@codemirror/search') ||
						id.includes('node_modules/@codemirror/commands') ||
						id.includes('node_modules/@codemirror/autocomplete') ||
						id.includes('node_modules/codemirror/') ||
						id.includes('node_modules/@lezer/highlight')
					) {
						return 'editor-core';
					}
					if (id.includes('node_modules/@codemirror/language-data')) {
						return 'editor-language-data';
					}
					if (id.includes('node_modules/lucide-react')) {
						return 'icon-vendor';
					}
					if (id.includes('node_modules/xlsx')) {
						return 'spreadsheet-vendor';
					}
					if (id.includes('node_modules/mammoth')) {
						return 'doc-preview-vendor';
					}
					if (id.includes('node_modules/jszip')) {
						return 'doc-preview-zip';
					}
					if (id.includes('@tauri-apps/api') || id.includes('@wailsio/runtime')) {
						return 'desktop-vendor';
					}
					return undefined;
				}
			}
		}
	},
	server: {
		open: false
	}
});
