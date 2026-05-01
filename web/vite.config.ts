import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/npm': 'http://localhost:8080',
      '/pypi': 'http://localhost:8080',
      '/docker': 'http://localhost:8080',
      '/golang': 'http://localhost:8080',
      '/cran': 'http://localhost:8080',
      '/gh': 'http://localhost:8080',
      '/gl': 'http://localhost:8080',
    }
  }
})