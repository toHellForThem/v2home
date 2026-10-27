<script setup lang="ts">
defineProps<{
  label: string
  value: string
  hint?: string
  accent?: boolean
}>()
</script>

<template>
  <div class="rounded-2xl border border-white/10 bg-white/[0.04] p-4">
    <p class="text-xs font-medium uppercase tracking-widest text-slate-400">{{ label }}</p>
    <p
      class="mt-2 truncate font-mono text-2xl tabular-nums"
      :class="accent ? 'text-emerald-300' : 'text-slate-100'"
      :title="value"
    >
      {{ value }}
    </p>
    <p v-if="hint" class="mt-1 truncate text-xs text-slate-500" :title="hint">{{ hint }}</p>
  </div>
</template>
