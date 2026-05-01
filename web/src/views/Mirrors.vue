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
    <h2 class="text-2xl font-semibold mb-6">Mirror Configuration</h2>
    <div v-if="loading" class="text-slate-400">Loading...</div>
    <div v-else class="space-y-4">
      <div v-for="m in mirrors" :key="m.name"
        class="bg-slate-800 border border-slate-700 rounded-lg p-4 flex items-center gap-4">
        <div class="flex-1">
          <div class="font-medium text-white">{{ m.name }}</div>
          <div class="text-sm text-slate-400 mt-1">Upstream: {{ m.upstream }}</div>
          <div class="text-sm text-slate-400">Cache TTL: {{ m.cacheTTL }}s</div>
          <input v-model="m.upstream" @change="updateUpstream(m)"
            class="mt-2 w-full bg-slate-700 border border-slate-600 rounded px-3 py-1 text-sm text-slate-200" />
        </div>
        <button @click="toggleMirror(m)"
          :class="m.enabled ? 'bg-emerald-600 hover:bg-emerald-700' : 'bg-slate-600 hover:bg-slate-500'"
          class="px-4 py-2 rounded-lg text-white font-medium transition-colors">
          {{ m.enabled ? 'Enabled' : 'Disabled' }}
        </button>
      </div>
    </div>
  </div>
</template>