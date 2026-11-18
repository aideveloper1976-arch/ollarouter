import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/admin/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsDir: '',
  },
  server: {
    proxy: {
      '/api': 'http://localhost:11435',
      '/admin/api': 'http://localhost:11435',
      '/health': 'http://localhost:11435',
    },
  },
})
