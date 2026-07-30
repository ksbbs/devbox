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
    const link = document.createElement('a')
    link.href = getReleaseDownloadUrl(result.ticket)
    link.download = result.fileName
    document.body.appendChild(link)
    link.click()
    link.remove()
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
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">downloads</span>
      <h1 class="page-title">Releases</h1>
      <p class="page-subtitle">保存 GitHub Release 源，通过 DevBox 下载最新稳定版本中的固定资产。</p>
    </section>

    <div v-if="errorMsg" class="mb-4 border border-red-500/40 bg-red-950/30 px-3 py-2 text-sm text-red-300">
      {{ errorMsg }}
    </div>
    <div v-if="statusMsg" class="mb-4 border border-emerald-500/30 bg-emerald-950/20 px-3 py-2 text-sm text-emerald-300">
      {{ statusMsg }}
    </div>

    <section class="panel-pad mb-5">
      <div class="mb-4 border-b border-slate-900 pb-3">
        <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">add source</h2>
      </div>
      <form class="grid gap-3 lg:grid-cols-[minmax(160px,0.7fr)_minmax(320px,1.5fr)_minmax(160px,0.7fr)_auto]" @submit.prevent="addSource">
        <label class="grid gap-1">
          <span class="text-xs uppercase tracking-[0.16em] text-slate-500">name</span>
          <input v-model="name" class="input w-full" maxlength="80" required placeholder="March7thAssistant" />
        </label>
        <label class="grid gap-1">
          <span class="text-xs uppercase tracking-[0.16em] text-slate-500">GitHub Releases URL</span>
          <input v-model="releaseUrl" type="url" class="input w-full" required placeholder="https://github.com/owner/repo/releases" />
        </label>
        <label class="grid gap-1">
          <span class="text-xs uppercase tracking-[0.16em] text-slate-500">asset name</span>
          <input v-model="assetName" class="input w-full" maxlength="255" required placeholder="update.7z" />
        </label>
        <button class="btn btn-primary self-end" :disabled="saving">
          {{ saving ? 'validating' : 'add source' }}
        </button>
      </form>
    </section>

    <section class="table-wrap">
      <div class="flex items-center justify-between gap-3 border-b border-slate-800 px-4 py-3">
        <div>
          <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">saved sources</h2>
          <p class="mt-1 text-xs text-slate-600">{{ sources.length }} configured</p>
        </div>
        <button class="btn" :disabled="loading || refreshing" @click="loadSources(true)">
          {{ refreshing ? 'refreshing' : 'refresh' }}
        </button>
      </div>

      <div v-if="loading" class="p-6 text-sm text-cyan-400">loading release sources...</div>
      <div v-else-if="!sources.length" class="p-6 text-center text-sm text-slate-500">暂无 Release 源。</div>
      <div v-else class="divide-y divide-slate-900">
        <article v-for="source in sources" :key="source.id" class="grid gap-4 px-4 py-4 transition-colors duration-150 hover:bg-slate-900/40 xl:grid-cols-[minmax(240px,1.1fr)_minmax(180px,0.75fr)_minmax(220px,1fr)_auto] xl:items-center">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="font-semibold text-slate-100">{{ source.name }}</h3>
              <span class="tag" :class="source.available ? 'tag-ok' : 'tag-off'">
                {{ source.available ? 'available' : 'unavailable' }}
              </span>
            </div>
            <a :href="source.repositoryUrl" target="_blank" rel="noreferrer" class="muted-link mt-1 block truncate text-xs">
              {{ source.owner }}/{{ source.repo }}
            </a>
            <p v-if="source.error" class="mt-2 text-xs text-red-300">{{ source.error }}</p>
          </div>

          <dl class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs xl:grid-cols-1">
            <div class="flex gap-2">
              <dt class="text-slate-600">release</dt>
              <dd class="truncate text-slate-300">{{ source.tagName || '-' }}</dd>
            </div>
            <div class="flex gap-2">
              <dt class="text-slate-600">published</dt>
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
              {{ downloading === source.id ? 'preparing' : 'download' }}
            </button>
            <button class="btn" :class="confirmDelete === source.id ? 'btn-danger' : ''" :disabled="deleting === source.id" @click="removeSource(source)">
              {{ deleting === source.id ? 'deleting' : confirmDelete === source.id ? 'confirm delete' : 'delete' }}
            </button>
            <button v-if="confirmDelete === source.id" class="btn" @click="confirmDelete = null">cancel</button>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>
