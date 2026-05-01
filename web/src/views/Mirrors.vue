<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMirrorConfig, updateMirrorConfig } from '../api/client'

const mirrors = ref<any[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    mirrors.value = await getMirrorConfig()
  } catch { mirrors.value = [] }
  loading.value = false
})

async function toggleMirror(m: any) {
  await updateMirrorConfig(m.name, !m.enabled)
  m.enabled = !m.enabled
}

async function updateUpstream(m: any) {
  await updateMirrorConfig(m.name, m.enabled, m.upstream)
}
</script>

<template>
  <div>
    <h2 class="text-2xl font-semibold mb-6 text-white">Mirror Configuration</h2>
    <div v-if="loading" class="flex items-center gap-2 text-slate-400">
      <span class="loading-dot"></span><span class="loading-dot"></span><span class="loading-dot"></span>
    </div>
    <TransitionGroup appear name="card" v-else tag="div" class="space-y-3">
      <div v-for="m in mirrors" :key="m.name"
        class="card-hover bg-slate-800 border border-slate-700/60 rounded-xl p-4 flex items-center gap-4 transition-all duration-300 hover:border-sky-500/30">
        <div class="flex-1">
          <div class="font-semibold text-white text-lg">{{ m.name }}</div>
          <div class="text-sm text-slate-400 mt-1">Upstream: <span class="text-slate-300">{{ m.upstream }}</span></div>
          <div class="text-sm text-slate-400">Cache TTL: <span class="text-slate-300">{{ m.cacheTTL }}s</span></div>
          <input v-model="m.upstream" @change="updateUpstream(m)"
            class="mt-2 w-full bg-slate-700/50 border border-slate-600 rounded-lg px-3 py-1.5 text-sm text-slate-200 focus:border-sky-500/50 focus:outline-none focus:ring-1 focus:ring-sky-500/30 transition-all" />
        </div>
        <button @click="toggleMirror(m)"
          :class="m.enabled
            ? 'bg-emerald-500 hover:bg-emerald-600 shadow-emerald-500/20 shadow-sm'
            : 'bg-slate-600 hover:bg-slate-500 shadow-slate-600/20 shadow-sm'"
          class="px-4 py-2 rounded-lg text-white font-medium transition-all duration-200 min-w-[80px]">
          {{ m.enabled ? 'Enabled' : 'Disabled' }}
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style>
.loading-dot {
  width: 8px; height: 8px; border-radius: 50%; background: #38bdf8;
  animation: pulse 1.2s ease-in-out infinite;
}
.loading-dot:nth-child(2) { animation-delay: 0.2s; }
.loading-dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes pulse {
  0%, 100% { opacity: 0.3; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1); }
}
</style>