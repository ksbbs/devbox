<script setup lang="ts">
defineProps<{ mirror: any }>()
</script>

<template>
  <div class="card-hover bg-slate-800 border border-slate-700/60 rounded-xl p-5 transition-all duration-300 hover:border-sky-500/30 hover:shadow-lg hover:shadow-sky-500/5">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-lg font-semibold text-white">{{ mirror.name }}</h3>
      <span v-if="mirror.status === 'healthy' && mirror.enabled"
        class="status-badge bg-emerald-500/20 text-emerald-400">
        Healthy
      </span>
      <span v-else-if="mirror.status !== 'healthy' && mirror.enabled"
        class="status-badge bg-red-500/20 text-red-400">
        Unhealthy
      </span>
      <span v-else
        class="status-badge bg-slate-600/30 text-slate-500">
        Disabled
      </span>
    </div>
    <div class="text-sm text-slate-400 space-y-1.5">
      <div>Pattern: <span class="text-slate-300 font-mono">{{ mirror.pattern }}</span></div>
      <div>Upstream: <span class="text-slate-300">{{ mirror.upstream }}</span></div>
      <div v-if="mirror.error" class="text-red-400 mt-1 text-xs">{{ mirror.error }}</div>
    </div>
  </div>
</template>

<style>
.status-badge {
  padding: 0.2rem 0.5rem;
  border-radius: 0.375rem;
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.02em;
}
</style>