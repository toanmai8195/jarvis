import { defineConfig } from 'tsup';

// Bundle một entry ESM cho Node 22; dependency (fastify, mongodb, pino) để
// external — `node dist/server.js` import chúng từ node_modules.
export default defineConfig({
  entry: ['src/server.ts'],
  format: ['esm'],
  platform: 'node',
  target: 'node22',
  outDir: 'dist',
  clean: true,
  sourcemap: true,
  dts: false,
});
