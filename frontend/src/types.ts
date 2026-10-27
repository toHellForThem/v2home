export interface RuntimeStats {
  goroutines: number
  numCpu: number
  heapAllocBytes: number
  heapSysBytes: number
  gcCount: number
}

export interface TunnelSummary {
  total: number
  running: number
}

export interface StatusSnapshot {
  app: string
  version: string
  commit: string
  buildTime: string
  goVersion: string
  os: string
  arch: string
  hostname: string
  startedAt: string
  uptimeSeconds: number
  now: string
  runtime: RuntimeStats
  tunnels: TunnelSummary
}

export type ConnectionState = 'connecting' | 'open' | 'closed' | 'error'
