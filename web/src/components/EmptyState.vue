<script setup lang="ts">
defineProps<{
  message: string
}>()
</script>

<template>
  <div class="flex items-center justify-center gap-3 px-6 py-10 text-center">
    <svg class="h-8 w-8 text-slate-700" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" aria-hidden="true">
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M3 9h18" />
      <path d="M7 14h6" stroke-linecap="round" />
    </svg>
    <p class="text-sm text-slate-500">{{ message }}</p>
  </div>
</template>
