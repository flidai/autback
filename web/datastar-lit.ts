import type { ReactiveElement } from 'lit'
import { loadDatastarRuntime, type DatastarRuntime } from './datastar-runtime'

type Constructor<T = object> = new (...args: any[]) => T
type Runtime = Pick<DatastarRuntime, 'effect' | 'getPath' | 'root'>

let loadedRuntime: Runtime | null = null
let runtimePromise: Promise<Runtime> | null = null

export interface DatastarLitHost {
  signal<T>(path: string, fallback: T): T
}

export function DatastarLit<T extends Constructor<ReactiveElement>>(
  Base: T,
): T & Constructor<DatastarLitHost> {
  abstract class DatastarLitElement extends Base {
    #renderDispose: (() => void) | null = null
    #connected = false

    override connectedCallback(): void {
      this.#connected = true
      super.connectedCallback()
      void loadRuntime().then(async () => {
        if (!this.#connected) return
        this.requestUpdate()
        await this.updateComplete
        await afterInitialSignalScan()
        if (this.#connected) this.requestUpdate()
      })
    }

    override performUpdate(): void {
      if (!this.isUpdatePending) return
      const activeRuntime = loadedRuntime
      if (!activeRuntime) {
        super.performUpdate()
        return
      }
      this.#renderDispose?.()
      let updateFromLit = true
      this.#renderDispose = activeRuntime.effect(() => {
        Object.keys(activeRuntime.root)
        if (updateFromLit) {
          updateFromLit = false
          super.performUpdate()
          return
        }
        this.requestUpdate()
      })
    }

    override disconnectedCallback(): void {
      this.#connected = false
      this.#renderDispose?.()
      this.#renderDispose = null
      super.disconnectedCallback()
    }

    signal<Value>(path: string, fallback: Value): Value {
      const value = loadedRuntime?.getPath<Value>(path)
      return materialize(value === undefined ? fallback : value)
    }
  }
  return DatastarLitElement as unknown as T & Constructor<DatastarLitHost>
}

async function loadRuntime(): Promise<Runtime> {
  if (loadedRuntime) return loadedRuntime
  runtimePromise ??= loadDatastarRuntime()
  loadedRuntime = await runtimePromise
  return loadedRuntime
}

async function afterInitialSignalScan(): Promise<void> {
  await Promise.resolve()
  await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
}

function materialize<Value>(value: Value): Value {
  if (Array.isArray(value)) return value.map((item) => materialize(item)) as Value
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>).map(([key, item]) => [key, materialize(item)]),
    ) as Value
  }
  return value
}
