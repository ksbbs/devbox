<script setup lang="ts">
withDefaults(defineProps<{
  message?: string
  tone?: 'error' | 'success' | 'info'
}>(), {
  tone: 'error',
})

defineSlots<{
  default: () => unknown
}>()
</script>

<template>
  <div
    v-if="message || $slots.default"
    class="mb-4 flex items-start gap-2.5 rounded-md border px-3 py-2 text-sm"
    :class="{
      'border-red-500/40 bg-red-950/25 text-red-300': tone === 'error',
      'border-emerald-500/30 bg-emerald-950/20 text-emerald-300': tone === 'success',
      'border-cyan-500/30 bg-cyan-950/20 text-cyan-300': tone === 'info',
    }"
  >
    <svg
      v-if="tone === 'error'"
      class="mt-0.5 h-4 w-4 shrink-0"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      stroke-width="1.4"
      stroke-linecap="round"
      aria-hidden="true"
    >
      <circle cx="8" cy="8" r="6.5" />
      <path d="M8 4.8v4" />
      <circle cx="8" cy="11.4" r="0.6" fill="currentColor" stroke="none" />
    </svg>
    <svg
      v-else-if="tone === 'success'"
      class="mt-0.5 h-4 w-4 shrink-0"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      stroke-width="1.4"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <circle cx="8" cy="8" r="6.5" />
      <polyline points="5 8.2 7.2 10.3 11 6" />
    </svg>
    <span class="min-w-0">
      <slot>{{ message }}</slot>
    </span>
  </div>
</template>
