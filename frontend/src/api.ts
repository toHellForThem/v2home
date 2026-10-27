import type { StatusSnapshot } from './types'

export async function fetchStatus(signal?: AbortSignal): Promise<StatusSnapshot> {
  const response = await fetch('/api/status', {
    signal,
    headers: { Accept: 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`GET /api/status failed with HTTP ${response.status}`)
  }

  return (await response.json()) as StatusSnapshot
}

export function statusSocketURL(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/status`
}
