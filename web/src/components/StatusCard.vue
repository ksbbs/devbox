<script setup lang="ts">
defineProps<{ mirror: any }>()
</script>

<template>
  <div class="bg-slate-800 border border-slate-700 rounded-lg p-4">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-lg font-medium text-white">{{ mirror.name }}</h3>
      <span v-if="mirror.status === 'healthy'"
        class="bg-emerald-500/20 text-emerald-400 px-2 py-0.5 rounded text-xs font-medium">
        Healthy
      </span>
      <span v-else
        class="bg-red-500/20 text-red-400 px-2 py-0.5 rounded text-xs font-medium">
        Unhealthy
      </span>
      <span v-if="!mirror.enabled"
        class="bg-slate-500/20 text-slate-400 px-2 py-0.5 rounded text-xs font-medium">
        Disabled
      </span>
    </div>
    <div class="text-sm text-slate-400">
      <div>Pattern: <span class="text-slate-300">{{ mirror.pattern }}</span></div>
      <div>Upstream: <span class="text-slate-300">{{ mirror.upstream }}</span></div>
      <div v-if="mirror.error" class="text-red-400 mt-1">{{ mirror.error }}</div>
    </div>
  </div>
</template>