export const datastarRuntimeURL = '/app/assets/datastar.js'

export type DatastarRuntime = {
  effect(fn: () => void): () => void
  getPath<T = unknown>(path: string): T | undefined
  root: Record<string, unknown>
}

let runtimePromise: Promise<DatastarRuntime> | null = null

export function loadDatastarRuntime(): Promise<DatastarRuntime> {
  const runtimeURL = new URL(datastarRuntimeURL, window.location.href).href
  runtimePromise ??= import(runtimeURL) as Promise<DatastarRuntime>
  return runtimePromise
}
