import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import path from 'path'

export default defineConfig({
  plugins: [
    vue(),
    Components({
      resolvers: [
        ElementPlusResolver({
          importStyle: 'css'
        })
      ]
    })
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return
          }

          if (id.includes('@element-plus/icons-vue')) {
            return 'vendor-ep-icons'
          }

          if (id.includes('/node_modules/element-plus/')) {
            const match = id.match(/\/element-plus\/es\/components\/([^/]+)\//)
            if (match?.[1]) {
              return `vendor-ep-${match[1]}`
            }
            return 'vendor-ep-core'
          }

          if (
            id.includes('/node_modules/vue/') ||
            id.includes('/node_modules/@vue/') ||
            id.includes('/node_modules/pinia/') ||
            id.includes('/node_modules/vue-router/')
          ) {
            return 'vendor-vue'
          }

          if (id.includes('/node_modules/axios/')) {
            return 'vendor-axios'
          }

          return 'vendor'
        }
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})
