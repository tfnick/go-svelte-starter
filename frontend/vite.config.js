import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

const backendPort = process.env.BACKEND_PORT || '3000';

export default defineConfig({
  plugins: [svelte()],
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${backendPort}`,
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
