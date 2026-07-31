<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  tone?: 'ok' | 'warn' | 'danger' | 'off' | 'brand' | 'accent'
  pulse?: boolean
}>(), {
  tone: 'off',
  pulse: false,
})

const ringClass = computed(() => ({
  ok: 'bg-emerald-400/60',
  warn: 'bg-amber-400/60',
  danger: 'bg-red-400/60',
  off: 'bg-slate-500/40',
  brand: 'bg-violet-400/60',
  accent: 'bg-cyan-400/60',
}[props.tone]))

const dotClass = computed(() => ({
  ok: 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]',
  warn: 'bg-amber-400 shadow-[0_0_8px_rgba(251,191,36,0.8)]',
  danger: 'bg-red-400 shadow-[0_0_8px_rgba(248,113,113,0.8)]',
  off: 'bg-slate-600 shadow-[0_0_6px_rgba(71,85,105,0.6)]',
  brand: 'bg-violet-400 shadow-[0_0_8px_rgba(167,139,250,0.8)]',
  accent: 'bg-cyan-400 shadow-[0_0_8px_rgba(34,211,238,0.8)]',
}[props.tone]))
</script>

<template>
  <span class="relative inline-flex h-2 w-2 shrink-0">
    <span
      v-if="pulse"
      class="absolute inset-0 rounded-full"
      :class="ringClass"
      style="animation: status-ping 1.8s cubic-bezier(0, 0, 0.2, 1) infinite"
    />
    <span class="relative inline-flex h-2 w-2 rounded-full" :class="dotClass" />
  </span>
</template>

<style>
@keyframes status-ping {
  75%, 100% {
    transform: scale(2.4);
    opacity: 0;
  }
}
</style>
