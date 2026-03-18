import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/:token/download': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    emptyOutDir: true,
    outDir: "../ui"
  }
})
