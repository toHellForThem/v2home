<script setup lang="ts">
import { computed } from 'vue'

import ConnectionBadge from '@/components/ConnectionBadge.vue'
import MetricCard from '@/components/MetricCard.vue'
import { useStatus } from '@/composables/useStatus'
import { formatBytes, formatDateTime, formatDuration } from '@/format'

const { snapshot, state, error, lastUpdated, refresh } = useStatus()

const badgeLabel = computed(() => {
  switch (state.value) {
    case 'open':
      return 'Live'
    case 'connecting':
      return 'Connecting'
    case 'closed':
      return 'Reconnecting'
    default:
      return 'Offline'
  }
})

const heapLabel = computed(() => {
  const runtime = snapshot.value?.runtime
  if (!runtime) {
    return '—'
  }
  return `${formatBytes(runtime.heapAllocBytes)} / ${formatBytes(runtime.heapSysBytes)}`
})

const rawJSON = computed(() => (snapshot.value ? JSON.stringify(snapshot.value, null, 2) : ''))

const updatedLabel = computed(() =>
  lastUpdated.value ? lastUpdated.value.toLocaleTimeString() : '—',
)
</script>

<template>
  <div class="relative min-h-screen overflow-hidden">
    <div class="pointer-events-none absolute inset-0 grid-overlay" aria-hidden="true"></div>

    <header class="relative border-b border-white/5 bg-slate-950/70 backdrop-blur">
      <div
        class="mx-auto flex max-w-6xl flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-10 w-10 items-center justify-center rounded-xl border border-emerald-400/30 bg-emerald-400/10 font-mono text-sm font-semibold text-emerald-300"
          >
            v2
          </div>
          <div>
            <h1 class="text-lg font-semibold tracking-tight">v2home</h1>
            <p class="text-xs text-slate-400">Lightweight tunnel manager · PoC control panel</p>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <ConnectionBadge :state="state" :label="badgeLabel" />
          <button
            type="button"
            class="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-xs font-medium text-slate-200 transition hover:border-emerald-400/40 hover:text-emerald-200 disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="state === 'connecting'"
            @click="refresh"
          >
            Refresh
          </button>
        </div>
      </div>
    </header>

    <main class="relative mx-auto max-w-6xl px-6 py-8">
      <div
        v-if="error"
        class="mb-6 rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200"
      >
        {{ error }}
      </div>

      <div
        v-if="!snapshot"
        class="rounded-2xl border border-white/10 bg-white/[0.03] p-10 text-center"
      >
        <p class="text-sm text-slate-400">Waiting for the first status snapshot…</p>
      </div>

      <template v-else>
        <section class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <MetricCard
            label="Host"
            :value="snapshot.hostname"
            :hint="`${snapshot.os} / ${snapshot.arch}`"
          />
          <MetricCard
            label="Version"
            :value="snapshot.version"
            :hint="`commit ${snapshot.commit}`"
            accent
          />
          <MetricCard
            label="Uptime"
            :value="formatDuration(snapshot.uptimeSeconds)"
            :hint="`since ${formatDateTime(snapshot.startedAt)}`"
          />
          <MetricCard
            label="Go runtime"
            :value="snapshot.goVersion"
            :hint="`${snapshot.runtime.numCpu} CPU · ${snapshot.runtime.goroutines} goroutines`"
          />
        </section>

        <section class="mt-4 grid gap-4 lg:grid-cols-3">
          <div class="rounded-2xl border border-white/10 bg-white/[0.04] p-5 lg:col-span-2">
            <div class="flex items-center justify-between">
              <h2 class="text-sm font-semibold text-slate-200">Runtime</h2>
              <span class="font-mono text-xs text-slate-500">updated {{ updatedLabel }}</span>
            </div>
            <dl class="mt-4 grid gap-4 sm:grid-cols-2">
              <div>
                <dt class="text-xs uppercase tracking-widest text-slate-500">Heap</dt>
                <dd class="mt-1 font-mono text-lg tabular-nums text-slate-100">{{ heapLabel }}</dd>
              </div>
              <div>
                <dt class="text-xs uppercase tracking-widest text-slate-500">GC cycles</dt>
                <dd class="mt-1 font-mono text-lg tabular-nums text-slate-100">
                  {{ snapshot.runtime.gcCount }}
                </dd>
              </div>
              <div>
                <dt class="text-xs uppercase tracking-widest text-slate-500">Goroutines</dt>
                <dd class="mt-1 font-mono text-lg tabular-nums text-slate-100">
                  {{ snapshot.runtime.goroutines }}
                </dd>
              </div>
              <div>
                <dt class="text-xs uppercase tracking-widest text-slate-500">Server time</dt>
                <dd class="mt-1 font-mono text-lg tabular-nums text-slate-100">
                  {{ formatDateTime(snapshot.now) }}
                </dd>
              </div>
            </dl>
          </div>

          <div class="rounded-2xl border border-white/10 bg-white/[0.04] p-5">
            <h2 class="text-sm font-semibold text-slate-200">Tunnels</h2>
            <p class="mt-1 text-xs text-slate-500">Placeholder for the future tunnel engine.</p>
            <div class="mt-6 flex items-end gap-6">
              <div>
                <p class="font-mono text-3xl tabular-nums text-slate-100">
                  {{ snapshot.tunnels.total }}
                </p>
                <p class="text-xs uppercase tracking-widest text-slate-500">total</p>
              </div>
              <div>
                <p class="font-mono text-3xl tabular-nums text-emerald-300">
                  {{ snapshot.tunnels.running }}
                </p>
                <p class="text-xs uppercase tracking-widest text-slate-500">running</p>
              </div>
            </div>
            <button
              type="button"
              disabled
              class="mt-6 w-full cursor-not-allowed rounded-lg border border-dashed border-white/10 px-3 py-2 text-xs text-slate-500"
            >
              Add tunnel (coming soon)
            </button>
          </div>
        </section>

        <section class="mt-4 rounded-2xl border border-white/10 bg-white/[0.03]">
          <details class="group">
            <summary
              class="cursor-pointer list-none px-5 py-4 text-sm font-medium text-slate-300 transition hover:text-emerald-200"
            >
              Raw snapshot
              <span class="ml-2 text-xs text-slate-500 group-open:hidden">show</span>
              <span class="ml-2 hidden text-xs text-slate-500 group-open:inline">hide</span>
            </summary>
            <pre
              class="max-h-96 overflow-auto border-t border-white/5 px-5 py-4 font-mono text-xs leading-relaxed text-slate-400"
            >{{ rawJSON }}</pre>
          </details>
        </section>
      </template>

      <footer class="mt-10 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-slate-500">
        <span>API:</span>
        <a
          class="transition hover:text-emerald-300"
          href="/api/status"
          target="_blank"
          rel="noreferrer"
          >/api/status</a
        >
        <a
          class="transition hover:text-emerald-300"
          href="/api/healthz"
          target="_blank"
          rel="noreferrer"
          >/api/healthz</a
        >
        <a
          class="transition hover:text-emerald-300"
          href="/api/version"
          target="_blank"
          rel="noreferrer"
          >/api/version</a
        >
        <span class="font-mono">ws://…/ws/status</span>
      </footer>
    </main>
  </div>
</template>
