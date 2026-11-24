import { defineConfig } from 'vite'
import { resolve } from 'node:path'
import { createRequire } from 'node:module'
import dts from 'vite-plugin-dts'

const require = createRequire(import.meta.url)
const optimalSelectEntry = require.resolve('optimal-select/lib/index.js')

export default defineConfig({
  plugins: [dts({ insertTypesEntry: true, include: ['src'], exclude: ['**/*.test.ts'] })],
  resolve: {
    alias: {
      'optimal-select': optimalSelectEntry,
    },
  },
  build: {
    emptyOutDir: true,
    sourcemap: false,
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'ZhupiSDK',
      formats: ['es', 'iife'],
      fileName: (format) => (format === 'es' ? 'zhupi.esm.js' : 'zhupi.js'),
    },
  },
})
