import { defineConfig } from 'vite';

export default defineConfig({
  base: './',
  server: {
    port: 34115,
    strictPort: true,
    host: 'localhost',
    hmr: {
      protocol: 'ws',
      host: 'localhost',
      port: 34115
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
