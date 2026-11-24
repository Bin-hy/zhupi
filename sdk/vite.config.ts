import { defineConfig } from 'vite'
import { resolve } from 'node:path'
import { createRequire } from 'node:module'
import dts from 'vite-plugin-dts'

const require = createRequire(import.meta.url)
const optimalSelectEntry = require.resolve('optimal-select/lib/index.js')

export default defineConfig({
  plugins: [dts({ insertTypesEntry: true, include: ['src'] })],
  resolve: {
    alias: {
      'optimal-select': optimalSelectEntry,
    },
  },
  build: {
    emptyOutDir: true,
    sourcemap: true,
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'AnnotationSDK',
      formats: ['es', 'iife'],
      fileName: (format) => (format === 'es' ? 'annotation-sdk.esm.js' : 'annotation-sdk.js'),
    },
  },
})
