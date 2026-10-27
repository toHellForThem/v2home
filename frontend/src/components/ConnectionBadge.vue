<script setup lang="ts">
import { computed } from 'vue'

import type { ConnectionState } from '@/types'

const props = defineProps<{
  state: ConnectionState
  label: string
}>()

const dotClass = computed(() => {
  switch (props.state) {
    case 'open':
      return 'bg-emerald-400 shadow-[0_0_10px_rgba(52,211,153,0.9)]'
    case 'connecting':
    case 'closed':
      return 'bg-amber-400 animate-pulse'
    default:
      return 'bg-rose-500'
  }
})

const textClass = computed(() => {
  switch (props.state) {
    case 'open':
      return 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200'
    case 'connecting':
    case 'closed':
      return 'border-amber-400/30 bg-amber-400/10 text-amber-200'
    default:
      return 'border-rose-500/30 bg-rose-500/10 text-rose-200'
  }
})
</script>

<template>
  <span
    class="inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium"
    :class="textClass"
  >
    <span class="h-1.5 w-1.5 rounded-full" :class="dotClass"></span>
    {{ label }}
  </span>
</template>
