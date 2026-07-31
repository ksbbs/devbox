<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { searchMirrors, getPublicConfig } from '../api/client'
import Panel from '../components/Panel.vue'
import CopyButton from '../components/CopyButton.vue'
import EmptyState from '../components/EmptyState.vue'
import Banner from '../components/Banner.vue'
import StatusDot from '../components/StatusDot.vue'

const query = ref('')
const results = ref<any[]>([])
const loading = ref(false)
const errorMsg = ref('')
const publicUrl = ref('')
const selectedRegistry = ref('')
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
  { label: '全部', value: '' },
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
    hasMore.value = data.has_more ?? data.hasMore ?? false
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
      <span class="page-kicker">检索</span>
      <h1 class="page-title">搜索</h1>
      <p class="page-subtitle">搜索 7 个包源并复制可直接使用的安装命令。</p>
    </section>

    <Panel class="mb-5">
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_160px_auto]">
        <input
          v-model="query"
          class="input w-full"
          type="text"
          placeholder="包名 / 镜像名"
          @keyup.enter="doSearch"
        />
        <select v-model="selectedRegistry" class="select w-full">
          <option v-for="opt in registryOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <button class="btn btn-primary" :disabled="loading" @click="doSearch">
          <svg v-if="!loading" class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" aria-hidden="true">
            <circle cx="7" cy="7" r="4.5" />
            <path d="m10.5 10.5 3 3" />
          </svg>
          {{ loading ? '搜索中' : '搜索' }}
        </button>
      </div>
      <p v-if="selectedRegistry === 'pypi'" class="mt-2 flex items-center gap-1.5 text-xs text-slate-500">
        <StatusDot tone="warn" /> PyPI 当前仅支持精确包名搜索。
      </p>
      <p v-if="selectedRegistry === 'conda'" class="mt-2 flex items-center gap-1.5 text-xs text-slate-500">
        <StatusDot tone="warn" /> Conda 当前仅支持精确包名搜索。
      </p>
    </Panel>

    <Banner v-if="errorMsg" :message="errorMsg" />

    <div class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>仓库源</th>
            <th>名称</th>
            <th>描述</th>
            <th>安装命令</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in results" :key="r.registry + r.name">
            <td><span class="tag" :class="tagClass(r.registry)">{{ r.registry }}</span></td>
            <td class="font-semibold text-slate-100">{{ r.name }}</td>
            <td class="max-w-[360px] truncate text-slate-500">{{ r.desc || '-' }}</td>
            <td class="max-w-[420px] truncate font-mono text-xs text-emerald-300">{{ installCommand(r.name, r.registry) }}</td>
            <td>
              <CopyButton :text="installCommand(r.name, r.registry)" />
            </td>
          </tr>
        </tbody>
      </table>
      <div class="flex items-center justify-between px-4 py-3">
        <div class="text-sm text-slate-500" v-if="results.length">
          第 {{ page }} 页
        </div>
        <div class="ml-auto flex gap-2">
          <button class="btn" :disabled="page <= 1" @click="prevPage">上一页</button>
          <button class="btn" :disabled="!hasMore" @click="nextPage">下一页</button>
        </div>
      </div>
      <EmptyState v-if="!results.length && !loading && query.trim()" message="搜索结果为空，请尝试其他关键词。" />
      <EmptyState v-if="!results.length && !loading && !query.trim()" message="输入关键词后开始搜索。" />
      <div v-if="loading" class="flex items-center justify-center gap-2 px-6 py-6 text-sm text-cyan-400">
        <StatusDot tone="accent" pulse /> 搜索中...
      </div>
    </div>
  </div>
</template>
