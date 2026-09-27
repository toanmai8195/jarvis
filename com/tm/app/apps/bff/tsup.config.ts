import { defineConfig } from 'tsup';

// Bundle ESM cho Node 22; dependency (fastify, mongodb, pino, @opentelemetry/*)
// để external — `node dist/server.js` import chúng từ node_modules.
// Hai entry: `instrumentation.js` (OTel SDK, nạp bằng `node --import`) phải
// tách khỏi `server.js` để SDK start trước khi fastify/node:http được nạp.
// Module dùng chung (logger, config) tách thành chunk.
export default defineConfig({
  entry: ['src/instrumentation.ts', 'src/server.ts'],
  format: ['esm'],
  platform: 'node',
  target: 'node22',
  outDir: 'dist',
  clean: true,
  sourcemap: true,
  dts: false,
});
