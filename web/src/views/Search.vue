<script setup lang="ts">
import { ref } from 'vue'
import { searchMirrors, getPublicConfig } from '../api/client'

const query = ref('')
const results = ref<any[]>([])
const loading = ref(false)
const publicUrl = ref('')
const selectedRegistry = ref('')
const copiedName = ref<string | null>(null)

getPublicConfig().then(c => publicUrl.value = c.publicUrl || '').catch(() => {})

const registryOptions = [
  { label: '全部', value: '' },
  { label: 'npm', value: 'npm' },
  { label: 'Docker Hub', value: 'docker' },
  { label: 'PyPI', value: 'pypi' },
]

const registryColors: Record<string, string> = {
  npm: '#38bdf8',
  docker: '#10b981',
  pypi: '#a78bfa',
}

async function doSearch() {
  if (!query.value.trim()) return
  loading.value = true
  try {
    results.value = await searchMirrors(query.value, selectedRegistry.value)
  } catch {
    results.value = []
  }
  loading.value = false
}

function copyInstall(name: string, registry: string) {
  const base = publicUrl.value || 'http://localhost:8080'
  let cmd = ''
  if (registry === 'npm') cmd = `npm install ${name} --registry ${base}/npm`
  else if (registry === 'docker') cmd = `docker pull ${name}`
  else if (registry === 'pypi') cmd = `pip install ${name} -i ${base}/pypi`
  navigator.clipboard.writeText(cmd)
  copiedName.value = name
  setTimeout(() => copiedName.value = null, 2000)
}
</script>

<template>
  <div>
    <div class="mb-8">
      <h1 class="text-4xl font-bold mb-2">
        <span class="bg-gradient-to-r from-sky-400 via-cyan-400 to-teal-400 bg-clip-text text-transparent">
          Search
        </span>
      </h1>
      <p class="text-slate-400">搜索 npm、Docker Hub、PyPI 镜像包</p>
    </div>

    <!-- 搜索栏 -->
    <div class="search-card p-6 rounded-2xl mb-6">
      <div class="flex gap-3">
        <div class="flex-1 relative">
          <input
            v-model="query"
            @keyup.enter="doSearch"
            type="text"
            placeholder="输入包名搜索..."
            class="w-full px-4 py-3 rounded-xl bg-white/5 border border-white/10 text-white placeholder-slate-500 focus:border-sky-500/50 focus:outline-none transition-all"
          />
        </div>
        <select
          v-model="selectedRegistry"
          class="px-4 py-3 rounded-xl bg-white/5 border border-white/10 text-slate-300 focus:border-sky-500/50 focus:outline-none"
        >
          <option v-for="opt in registryOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <button
          @click="doSearch"
          :disabled="loading"
          class="px-6 py-3 rounded-xl bg-sky-500/20 text-sky-400 border border-sky-500/30 hover:bg-sky-500/30 transition-all disabled:opacity-50"
        >
          {{ loading ? '搜索中...' : '搜索' }}
        </button>
      </div>
    </div>

    <!-- 搜索结果 -->
    <div v-if="results.length" class="space-y-3">
      <div v-for="r in results" :key="r.registry + r.name"
        class="search-card group p-4 rounded-xl flex items-center gap-4 transition-all duration-300 hover:border-opacity-50"
        :style="{ '--color': registryColors[r.registry] || '#94a3b8' }">
        <span class="px-2 py-1 rounded-full text-xs font-medium"
          :style="{ color: registryColors[r.registry] || '#94a3b8', background: (registryColors[r.registry] || '#94a3b8') + '20' }">
          {{ r.registry }}
        </span>
        <div class="flex-1">
          <div class="font-medium text-white">{{ r.name }}</div>
          <div v-if="r.desc" class="text-sm text-slate-400 mt-1 truncate">{{ r.desc }}</div>
        </div>
        <button
          @click="copyInstall(r.name, r.registry)"
          class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
          :class="copiedName === r.name ? 'bg-emerald-500/20 text-emerald-400' : 'bg-white/5 text-slate-400 hover:bg-white/10 hover:text-white'"
        >
          {{ copiedName === r.name ? '已复制!' : '复制安装命令' }}
        </button>
      </div>
    </div>

    <div v-if="!results.length && !loading && query" class="text-center text-slate-500 py-12">
      搜索结果为空，请尝试其他关键词
    </div>
  </div>
</template>

<style>
.search-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.05) 0%, rgba(255, 255, 255, 0.02) 100%);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(10px);
}

.search-card:hover {
  border-color: var(--color, rgba(56, 189, 248, 0.3));
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
}
</style>