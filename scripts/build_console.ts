import { mkdir } from 'node:fs/promises'
import { join } from 'node:path'

const root = join(import.meta.dir, '..')
const output = join(root, 'internal', 'console', 'assets')
await mkdir(output, { recursive: true })
process.env.NODE_ENV = 'production'

const result = await Bun.build({
  entrypoints: [join(root, 'web', 'console.ts')],
  outdir: output,
  naming: 'console.js',
  format: 'esm',
  target: 'browser',
  conditions: ['browser', 'production'],
  define: { 'process.env.NODE_ENV': '"production"' },
  minify: true,
})
if (!result.success) {
  for (const log of result.logs) console.error(log)
  process.exit(1)
}

await Bun.write(
  join(output, 'datastar.js'),
  Bun.file(join(root, 'web', 'vendor', 'datastar-1.0.2.js')),
)
await Bun.write(
  join(output, 'favicon.svg'),
  Bun.file(join(root, 'site', 'assets', 'favicon.svg')),
)
