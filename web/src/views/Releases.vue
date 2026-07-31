<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  createReleaseDownloadTicket,
  createReleaseSource,
  deleteReleaseSource,
  getReleaseDownloadUrl,
  getReleaseSources,
  type ReleaseSource,
} from '../api/client'
import Panel from '../components/Panel.vue'
import Banner from '../components/Banner.vue'
import StatusDot from '../components/StatusDot.vue'

const sources = ref<ReleaseSource[]>([])
const loading = ref(true)
const refreshing = ref(false)
const saving = ref(false)
const errorMsg = ref('')
const statusMsg = ref('')
const downloading = ref<number | null>(null)
const deleting = ref<number | null>(null)
const confirmDelete = ref<number | null>(null)
const name = ref('')
const releaseUrl = ref('')
const assetName = ref('')

onMounted(loadSources)

async function loadSources(refresh = false) {
  if (refresh) refreshing.value = true
  else loading.value = true
  errorMsg.value = ''
  try {
    sources.value = await getReleaseSources(refresh)
  } catch (e: any) {
    errorMsg.value = responseMessage(e, '加载 Release 源失败')
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

async function addSource() {
  saving.value = true
  errorMsg.value = ''
  statusMsg.value = ''
  try {
    const source = await createReleaseSource({
      name: name.value.trim(),
      releaseUrl: releaseUrl.value.trim(),
      assetName: assetName.value.trim(),
    })
    sources.value.push(source)
    name.value = ''
    releaseUrl.value = ''
    assetName.value = ''
    flashStatus('Release 源已保存')
  } catch (e: any) {
    errorMsg.value = responseMessage(e, '添加 Release 源失败')
  } finally {
    saving.value = false
  }
}

async function download(source: ReleaseSource) {
  downloading.value = source.id
  errorMsg.value = ''
  statusMsg.value = ''
  try {
    const result = await createReleaseDownloadTicket(source.id)
    window.location.href = getReleaseDownloadUrl(result.ticket)
    flashStatus(`${source.name} ${result.tagName} 已开始下载`)
  } catch (e: any) {
    errorMsg.value = responseMessage(e, '准备下载失败')
  } finally {
    downloading.value = null
  }
}

async function removeSource(source: ReleaseSource) {
  if (confirmDelete.value !== source.id) {
    confirmDelete.value = source.id
    return
  }
  deleting.value = source.id
  errorMsg.value = ''
  try {
    await deleteReleaseSource(source.id)
    sources.value = sources.value.filter((item) => item.id !== source.id)
    confirmDelete.value = null
    flashStatus(`${source.name} 已删除`)
  } catch (e: any) {
    errorMsg.value = responseMessage(e, '删除 Release 源失败')
  } finally {
    deleting.value = null
  }
}

function responseMessage(error: any, fallback: string) {
  const data = error.response?.data
  if (typeof data === 'string' && data.trim()) return data.trim()
  return error.response?.statusText || fallback
}

function flashStatus(message: string) {
  statusMsg.value = message
  setTimeout(() => {
    if (statusMsg.value === message) statusMsg.value = ''
  }, 2500)
}

function formatBytes(value?: number) {
  if (value === undefined || !Number.isFinite(value)) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
}

function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">下载</span>
      <h1 class="page-title">发行版</h1>
      <p class="page-subtitle">保存 GitHub Release 源，通过 DevBox 下载最新稳定版本中的指定资产。</p>
    </section>

    <Banner v-if="errorMsg" :message="errorMsg" />
    <Banner v-if="statusMsg" :message="statusMsg" tone="success" />

    <Panel class="mb-5">
      <template #title>添加下载源</template>
      <template #desc>保存后 DevBox 会立即校验仓库与资产是否存在；下载时自动定位最新稳定版本。仅支持公开仓库，源数据持久化于服务器数据库。</template>
      <form class="grid grid-cols-1 gap-3 lg:grid-cols-[minmax(160px,0.7fr)_minmax(320px,1.5fr)_minmax(160px,0.7fr)_auto]" @submit.prevent="addSource">
        <label class="grid gap-1">
          <span class="text-xs uppercase tracking-[0.16em] text-slate-500">名称</span>
          <input v-model="name" class="input w-full" maxlength="80" required placeholder="例如 March7thAssistant" />
          <span class="text-[11px] text-slate-600">用于区分多个下载源，最长 80 字符。</span>
        </label>
        <label class="grid gap-1">
          <span class="text-xs uppercase tracking-[0.16em] text-slate-500">GitHub Releases 地址</span>
          <input v-model="releaseUrl" type="url" class="input w-full" required placeholder="https://github.com/owner/repo/releases" />
          <span class="text-[11px] text-slate-600">仓库的 Releases 页面地址，保存时解析 owner 与 repo 并校验仓库存在。</span>
        </label>
        <label class="grid gap-1">
          <span class="text-xs uppercase tracking-[0.16em] text-slate-500">资产文件名</span>
          <input v-model="assetName" class="input w-full" maxlength="255" required placeholder="例如 update.7z" />
          <span class="text-[11px] text-slate-600">需与 Release 资产名完全一致（精确匹配），保存时校验资产存在。</span>
        </label>
        <button class="btn btn-primary self-end" :disabled="saving">
          {{ saving ? '校验中' : '添加源' }}
        </button>
      </form>
    </Panel>

    <section class="glass">
      <div class="flex items-center justify-between gap-3 border-b border-slate-800/80 px-4 py-3">
        <div>
          <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">已保存的下载源</h2>
          <p class="mt-1 text-xs text-slate-600">已配置 {{ sources.length }} 个</p>
        </div>
        <button class="btn" :disabled="loading || refreshing" @click="loadSources(true)">
          <svg v-if="!refreshing" class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M13 8a5 5 0 1 1-1.5-3.6" />
            <path d="M13 2.5V5.5H10" />
          </svg>
          {{ refreshing ? '刷新中' : '刷新' }}
        </button>
      </div>

      <div v-if="loading" class="flex items-center gap-2 p-6 text-sm text-cyan-400">
        <StatusDot tone="accent" pulse /> 正在加载下载源...
      </div>
      <div v-else-if="!sources.length" class="flex items-center justify-center gap-3 px-6 py-10 text-sm text-slate-500">
        <svg class="h-8 w-8 text-slate-700" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" aria-hidden="true">
          <path d="M12 3v10" stroke-linecap="round" />
          <path d="m7 12 5 5 5-5" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M4 19h16" stroke-linecap="round" />
        </svg>
        <span>暂无下载源，使用上方表单添加一个 GitHub Release 源。</span>
      </div>
      <div v-else class="divide-y divide-slate-800/60">
        <article v-for="source in sources" :key="source.id" class="grid grid-cols-1 gap-4 px-4 py-4 transition-colors duration-150 hover:bg-cyan-400/[0.03] xl:grid-cols-[minmax(240px,1.1fr)_minmax(180px,0.75fr)_minmax(220px,1fr)_auto] xl:items-center">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="font-semibold text-slate-100">{{ source.name }}</h3>
              <span class="tag" :class="source.available ? 'tag-ok' : 'tag-off'">
                <StatusDot :tone="source.available ? 'ok' : 'off'" />
                {{ source.available ? '可用' : '不可用' }}
              </span>
            </div>
            <a :href="source.repositoryUrl" target="_blank" rel="noreferrer" class="muted-link mt-1 block truncate text-xs">
              {{ source.owner }}/{{ source.repo }}
            </a>
            <p v-if="source.error" class="mt-2 text-xs text-red-300">{{ source.error }}</p>
          </div>

          <dl class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs xl:grid-cols-1">
            <div class="flex gap-2">
              <dt class="text-slate-600">版本</dt>
              <dd class="truncate text-slate-300">{{ source.tagName || '-' }}</dd>
            </div>
            <div class="flex gap-2">
              <dt class="text-slate-600">发布时间</dt>
              <dd class="text-slate-400">{{ formatDate(source.publishedAt) }}</dd>
            </div>
          </dl>

          <div class="min-w-0">
            <div class="font-mono text-sm text-cyan-300">{{ source.assetName }}</div>
            <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-500">
              <span>{{ formatBytes(source.assetSize) }}</span>
              <span v-if="source.digest" class="max-w-full truncate font-mono" :title="source.digest">{{ source.digest }}</span>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2 xl:justify-end">
            <button class="btn btn-primary" :disabled="!source.available || downloading === source.id" @click="download(source)">
              <svg v-if="downloading !== source.id" class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M8 2.5v7" />
                <path d="m4.5 6.5 3.5 3.5 3.5-3.5" />
                <path d="M2.5 13.5h11" />
              </svg>
              {{ downloading === source.id ? '准备中' : '下载' }}
            </button>
            <button class="btn" :class="confirmDelete === source.id ? 'btn-danger' : ''" :disabled="deleting === source.id" @click="removeSource(source)">
              {{ deleting === source.id ? '删除中' : confirmDelete === source.id ? '确认删除' : '删除' }}
            </button>
            <button v-if="confirmDelete === source.id" class="btn" @click="confirmDelete = null">取消</button>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>
