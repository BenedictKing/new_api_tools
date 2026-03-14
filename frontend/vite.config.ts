import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'index.html'),
        embed: path.resolve(__dirname, 'embed.html'),
      },
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return undefined
          }

          if (id.includes('echarts') || id.includes('zrender')) {
            return 'vendor-echarts'
          }

          if (id.includes('@lobehub/icons') || id.includes('lucide-react')) {
            return 'vendor-icons'
          }

          if (id.includes('@radix-ui')) {
            return 'vendor-radix'
          }

          if (
            id.includes('/node_modules/react/') ||
            id.includes('/node_modules/react-dom/') ||
            id.includes('/node_modules/scheduler/')
          ) {
            return 'vendor-react'
          }

          return 'vendor'
        },
      },
    },
  },
  server: {
    port: 3188,
    proxy: {
      '/api': {
        target: 'http://localhost:3088',
        changeOrigin: true,
      },
    },
  },
})
