<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { searchMirrors, getPublicConfig } from '../api/client'

const query = ref('')
const results = ref<any[]>([])
const loading = ref(false)
const errorMsg = ref('')
const publicUrl = ref('')
const selectedRegistry = ref('')
const copiedName = ref<string | null>(null)

onMounted(async () => {
  try {
    const config = await getPublicConfig()
    publicUrl.value = config.publicUrl || window.location.origin
  } catch {
    publicUrl.value = window.location.origin
  }
})

const registryOptions = [
  { label: 'all', value: '' },
  { label: 'npm', value: 'npm' },
  { label: 'docker', value: 'docker' },
  { label: 'pypi', value: 'pypi' },
]

async function doSearch() {
  if (!query.value.trim()) return
  loading.value = true
  errorMsg.value = ''
  try {
    results.value = await searchMirrors(query.value, selectedRegistry.value)
  } catch (e: any) {
    results.value = []
    errorMsg.value = e.response?.statusText || '搜索失败，请稍后重试'
  }
  loading.value = false
}

function installCommand(name: string, registry: string) {
  const base = publicUrl.value || 'http://localhost:8080'
  if (registry === 'npm') return `npm install ${name} --registry ${base}/npm`
  if (registry === 'docker') return `docker pull ${name}`
  if (registry === 'pypi') return `pip install ${name} -i ${base}/pypi`
  return name
}

function copyInstall(name: string, registry: string) {
  navigator.clipboard.writeText(installCommand(name, registry))
  copiedName.value = name
  setTimeout(() => copiedName.value = null, 1500)
}
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">lookup</span>
      <h1 class="page-title">Search</h1>
      <p class="page-subtitle">搜索 npm、Docker Hub、PyPI，并复制可直接使用的安装命令。</p>
    </section>

    <section class="panel-pad mb-5">
      <div class="grid gap-3 lg:grid-cols-[1fr_160px_auto]">
        <input
          v-model="query"
          class="input w-full"
          type="text"
          placeholder="package / image name"
          @keyup.enter="doSearch"
        />
        <select v-model="selectedRegistry" class="select w-full">
          <option v-for="opt in registryOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <button class="btn btn-primary" :disabled="loading" @click="doSearch">
          {{ loading ? 'searching' : 'search' }}
        </button>
      </div>
      <p v-if="selectedRegistry === 'pypi'" class="mt-2 text-xs text-slate-500">PyPI 当前仅支持精确包名搜索。</p>
    </section>

    <div v-if="errorMsg" class="mb-4 border border-red-500/40 bg-red-950/30 px-3 py-2 text-sm text-red-300">
      {{ errorMsg }}
    </div>

    <div class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>registry</th>
            <th>name</th>
            <th>description</th>
            <th>install command</th>
            <th>action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in results" :key="r.registry + r.name">
            <td><span class="tag" :class="r.registry === 'docker' ? 'tag-ok' : r.registry === 'pypi' ? 'tag-warn' : ''">{{ r.registry }}</span></td>
            <td class="font-semibold text-slate-100">{{ r.name }}</td>
            <td class="max-w-[360px] truncate text-slate-500">{{ r.desc || '-' }}</td>
            <td class="max-w-[420px] truncate font-mono text-xs text-emerald-300">{{ installCommand(r.name, r.registry) }}</td>
            <td>
              <button class="btn" @click="copyInstall(r.name, r.registry)">
                {{ copiedName === r.name ? 'copied' : 'copy' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!results.length && !loading && query" class="p-6 text-center text-sm text-slate-500">搜索结果为空，请尝试其他关键词。</div>
      <div v-if="!results.length && !loading && !query" class="p-6 text-center text-sm text-slate-600">输入关键词后开始搜索。</div>
      <div v-if="loading" class="p-6 text-center text-sm text-cyan-400">searching...</div>
    </div>
  </div>
</template>
