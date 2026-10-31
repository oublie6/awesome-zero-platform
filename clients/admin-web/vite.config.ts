import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
  plugins: [
    vue(),
    Components({
      dts: './src/components.d.ts',
      resolvers: [ElementPlusResolver({ importStyle: 'css', directives: true })],
    }),
  ],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: {
    port: 5173,
    proxy: { '/api': { target: 'http://127.0.0.1:8888', changeOrigin: true, rewrite: path => path.replace(/^\/api/, '') } }
  },
  build: { sourcemap: true, outDir: 'dist' }
})
