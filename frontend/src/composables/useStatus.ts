import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'

import { fetchStatus, statusSocketURL } from '@/api'
import type { ConnectionState, StatusSnapshot } from '@/types'

const RECONNECT_MIN_MS = 1000
const RECONNECT_MAX_MS = 15000

export function useStatus() {
  const snapshot = shallowRef<StatusSnapshot | null>(null)
  const state = ref<ConnectionState>('connecting')
  const error = ref<string | null>(null)
  const lastUpdated = ref<Date | null>(null)

  let socket: WebSocket | null = null
  let retryTimer: number | undefined
  let retryDelay = RECONNECT_MIN_MS
  let stopped = false

  async function refresh() {
    try {
      snapshot.value = await fetchStatus()
      lastUpdated.value = new Date()
      error.value = null
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : String(cause)
    }
  }

  function scheduleReconnect() {
    window.clearTimeout(retryTimer)
    retryTimer = window.setTimeout(() => {
      retryDelay = Math.min(retryDelay * 2, RECONNECT_MAX_MS)
      connect()
    }, retryDelay)
  }

  function connect() {
    if (stopped) {
      return
    }

    state.value = 'connecting'
    const ws = new WebSocket(statusSocketURL())
    socket = ws

    ws.onopen = () => {
      state.value = 'open'
      retryDelay = RECONNECT_MIN_MS
      error.value = null
    }

    ws.onmessage = (event) => {
      try {
        snapshot.value = JSON.parse(event.data as string) as StatusSnapshot
        lastUpdated.value = new Date()
      } catch {
        // Ignore malformed frames; the next tick will replace them.
      }
    }

    ws.onerror = () => {
      state.value = 'error'
    }

    ws.onclose = () => {
      if (socket === ws) {
        socket = null
      }
      if (stopped) {
        return
      }
      state.value = 'closed'
      scheduleReconnect()
    }
  }

  onMounted(() => {
    void refresh()
    connect()
  })

  onBeforeUnmount(() => {
    stopped = true
    window.clearTimeout(retryTimer)
    socket?.close()
  })

  return { snapshot, state, error, lastUpdated, refresh }
}
