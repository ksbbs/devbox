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
const page = ref(1)
const perPage = 10
const hasMore = ref(false)

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
  { label: 'conda', value: 'conda' },
  { label: 'rubygems', value: 'rubygems' },
  { label: 'cargo', value: 'cargo' },
  { label: 'nuget', value: 'nuget' },
]

async function doSearch() {
  if (!query.value.trim()) return
  page.value = 1
  await fetchResults()
}

async function fetchResults() {
  loading.value = true
  errorMsg.value = ''
  try {
    const data = await searchMirrors(query.value, selectedRegistry.value, page.value, perPage)
    results.value = data.results || []
    hasMore.value = data.hasMore || false
  } catch (e: any) {
    results.value = []
    hasMore.value = false
    errorMsg.value = e.response?.statusText || '搜索失败，请稍后重试'
  }
  loading.value = false
}

function nextPage() {
  page.value++
  fetchResults()
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    fetchResults()
  }
}

function installCommand(name: string, registry: string) {
  const base = publicUrl.value || 'http://localhost:8080'
  switch (registry) {
    case 'npm': return `npm install ${name} --registry ${base}/npm`
    case 'docker': return `docker pull ${name}`
    case 'pypi': return `pip install ${name} -i ${base}/pypi`
    case 'conda': return `conda install ${name} -c ${base}/conda`
    case 'rubygems': return `gem install ${name} --source ${base}/rubygems`
    case 'cargo': return `cargo install ${name} --registry ${base}/cargo`
    case 'nuget': return `dotnet add package ${name} -s ${base}/nuget/index.json`
    default: return name
  }
}

function copyInstall(name: string, registry: string) {
  navigator.clipboard.writeText(installCommand(name, registry))
  copiedName.value = name
  setTimeout(() => copiedName.value = null, 1500)
}

function tagClass(registry: string) {
  switch (registry) {
    case 'npm': return ''
    case 'docker': return 'tag-ok'
    case 'pypi':
    case 'conda': return 'tag-warn'
    default: return ''
  }
}
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">lookup</span>
      <h1 class="page-title">Search</h1>
      <p class="page-subtitle">搜索 7 个包源并复制可直接使用的安装命令。</p>
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
      <p v-if="selectedRegistry === 'conda'" class="mt-2 text-xs text-slate-500">Conda 当前仅支持精确包名搜索。</p>
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
            <td><span class="tag" :class="tagClass(r.registry)">{{ r.registry }}</span></td>
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
      <div class="flex items-center justify-between px-4 py-3">
        <div class="text-sm text-slate-500" v-if="results.length">
          page {{ page }}
        </div>
        <div class="flex gap-2">
          <button class="btn btn-sm" :disabled="page <= 1" @click="prevPage">prev</button>
          <button class="btn btn-sm" :disabled="!hasMore" @click="nextPage">next</button>
        </div>
      </div>
      <div v-if="!results.length && !loading && query" class="p-6 text-center text-sm text-slate-500">搜索结果为空，请尝试其他关键词。</div>
      <div v-if="!results.length && !loading && !query" class="p-6 text-center text-sm text-slate-600">输入关键词后开始搜索。</div>
      <div v-if="loading" class="p-6 text-center text-sm text-cyan-400">searching...</div>
    </div>
  </div>
</template>
