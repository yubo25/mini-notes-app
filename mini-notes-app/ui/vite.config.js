import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: './',
  build: {
    outDir: '../bundle', 
    assetsDir: 'assets',
    emptyOutDir: true,
  }
})