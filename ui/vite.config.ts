import { defineConfig } from 'vite'
import path from 'path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'


function figmaAssetResolver() {
  return {
    name: 'figma-asset-resolver',
    resolveId(id) {
      if (id.startsWith('figma:asset/')) {
        const filename = id.replace('figma:asset/', '')
        return path.resolve(__dirname, 'src/assets', filename)
      }
    },
  }
}

export default defineConfig({
  plugins: [
    figmaAssetResolver(),
    // The React and Tailwind plugins are both required for Make, even if
    // Tailwind is not being actively used – do not remove them
    react(),
    tailwindcss(),
  ],

  resolve: {
    alias: {
      // Alias @ to the src directory
      '@': path.resolve(__dirname, './src'),
    },
  },

  // Dev server configuration — proxy API calls to Go backend
  server: {
    port: 5173,
    host: true,
    proxy: {
      // Proxy all /v1/* requests to the memory monolith backend
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
    },
  },

  // File types to support raw imports. Never add .css, .tsx, or .ts files to this.
  assetsInclude: ['**/*.svg', '**/*.csv'],

  build: {
    // Enterprise-grade chunking strategy
    rollupOptions: {
      output: {
        manualChunks: {
          // React core — always needed
          'vendor-react': ['react', 'react-dom'],
          // Data fetching layer
          'vendor-query': ['@tanstack/react-query'],
          // Charts (recharts is large)
          'vendor-charts': ['recharts'],
          // Icons
          'vendor-icons': ['lucide-react'],
        },
      },
    },
    // Raise warning threshold to 700kB (each chunk should be fine after split)
    chunkSizeWarningLimit: 700,
  },
})
