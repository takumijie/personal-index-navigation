/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, never>, Record<string, never>, unknown>
  export default component
}

interface NavigatorUAData {
  platform?: string
  getHighEntropyValues?: (hints: string[]) => Promise<{ platform?: string; platformVersion?: string }>
}

interface Navigator {
  userAgentData?: NavigatorUAData
}
