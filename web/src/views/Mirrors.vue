<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMirrorConfig, updateMirrorConfig, getGitProxyConfig, updateGitProxyConfig } from '../api/client'
import Panel from '../components/Panel.vue'
import StatusDot from '../components/StatusDot.vue'
import EmptyState from '../components/EmptyState.vue'
import Banner from '../components/Banner.vue'

const mirrors = ref<any[]>([])
const loading = ref(true)
const errorMsg = ref('')
const updating = ref<string | null>(null)
const saved = ref<string | null>(null)

const gitTtl = ref('')
const gitSaving = ref(false)
const gitMsg = ref('')
const gitLoaded = ref(false)

const TTL_OPTIONS = [
  { value: '0', label: '永久缓存' },
  { value: '1h', label: '1 小时' },
  { value: '6h', label: '6 小时' },
  { value: '12h', label: '12 小时' },
  { value: '1d', label: '1 天' },
  { value: '3d', label: '3 天' },
  { value: '7d', label: '7 天' },
  { value: '30d', label: '30 天' },
]

const GIT_TTL_OPTIONS = [
  { value: '0', label: '禁用缓存' },
  ...TTL_OPTIONS.slice(1),
]

// 走流式代理不参与缓存的镜像，TTL 配置不生效
const NO_CACHE_MIRRORS = ['docker', 'ghcr', 'quay', 'mcr', 'apt', 'alpine', 'homebrew', 'hf']
const isNoCache = (name: string) => NO_CACHE_MIRRORS.includes(name)

function parseTTLSeconds(value: number | string): number {
  if (value === null || value === undefined || value === '') return 0
  if (typeof value === 'number') return value
  const m = String(value).match(/^(\d+)([smhd])$/)
  if (m) {
    const mult: Record<string, number> = { s: 1, m: 60, h: 3600, d: 86400 }
    return Number(m[1]) * mult[m[2]]
  }
  return Number(value) || 0
}

function ttlText(value: number | string) {
  const ttl = parseTTLSeconds(value)
  if (!ttl) return '永久缓存'
  if (ttl >= 86400) return `${Math.round(ttl / 86400)} 天`
  if (ttl >= 3600) return `${Math.round(ttl / 3600)} 小时`
  if (ttl >= 60) return `${Math.round(ttl / 60)} 分钟`
  return `${ttl} 秒`
}

onMounted(async () => {
  try {
    mirrors.value = await getMirrorConfig()
  } catch (e: any) {
    mirrors.value = []
    errorMsg.value = e.response?.statusText || '加载镜像配置失败'
  }
  loading.value = false
  try {
    const cfg = await getGitProxyConfig()
    gitTtl.value = cfg.cacheTTL || '0'
    gitLoaded.value = true
  } catch { /* git 代理面板非关键，失败时隐藏 */ }
})

async function reloadMirrors() {
  try {
    mirrors.value = await getMirrorConfig()
  } catch { /* 忽略刷新失败 */ }
}

async function toggleMirror(m: any) {
  updating.value = m.name
  errorMsg.value = ''
  try {
    const res = await updateMirrorConfig(m.name, !m.enabled)
    if (res?.status !== 'ok') throw new Error(res?.error || '操作失败')
    m.enabled = !m.enabled
    flashSaved(m.name)
  } catch (e: any) {
    errorMsg.value = e.response?.statusText || e.message || '操作失败'
  } finally {
    updating.value = null
  }
}

async function updateUpstream(m: any) {
  updating.value = m.name
  errorMsg.value = ''
  try {
    const res = await updateMirrorConfig(m.name, m.enabled, m.upstream)
    if (res?.status !== 'ok') throw new Error(res?.error || '保存失败')
    flashSaved(m.name)
  } catch (e: any) {
    errorMsg.value = e.response?.statusText || e.message || '保存失败'
    await reloadMirrors()
  } finally {
    updating.value = null
  }
}

async function updateTTL(m: any) {
  updating.value = m.name
  errorMsg.value = ''
  try {
    const res = await updateMirrorConfig(m.name, m.enabled, m.upstream, m.cacheTTL)
    if (res?.status !== 'ok') throw new Error(res?.error || '保存失败')
    flashSaved(m.name)
  } catch (e: any) {
    errorMsg.value = e.response?.statusText || e.message || '保存失败'
    await reloadMirrors()
  } finally {
    updating.value = null
  }
}

async function saveGitTtl() {
  gitSaving.value = true
  gitMsg.value = ''
  try {
    const res = await updateGitProxyConfig(gitTtl.value)
    gitTtl.value = res.cacheTTL || gitTtl.value
    gitMsg.value = 'saved'
    setTimeout(() => gitMsg.value = '', 1500)
  } catch (e: any) {
    gitMsg.value = e.response?.statusText || '保存失败'
  } finally {
    gitSaving.value = false
  }
}

function flashSaved(name: string) {
  saved.value = name
  setTimeout(() => saved.value = null, 1500)
}
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">配置</span>
      <h1 class="page-title">镜像源</h1>
      <p class="page-subtitle">启停镜像服务并修改上游地址，保存后立即生效。</p>
    </section>

    <div v-if="loading" class="glass flex items-center gap-2 p-4 text-sm text-cyan-400">
      <StatusDot tone="accent" pulse /> 正在加载镜像配置...
    </div>

    <Banner v-if="errorMsg" :message="errorMsg" />

    <Panel v-if="gitLoaded" class="mb-5">
      <template #title>Git 代理缓存 TTL</template>
      <template #desc>控制 /gh/ 与 /gl/ 代理的缓存时长，0 表示禁用缓存。</template>
      <div class="flex flex-wrap items-center gap-3">
        <select v-model="gitTtl" class="input w-40" :disabled="gitSaving">
          <option v-for="opt in GIT_TTL_OPTIONS" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <button class="btn btn-primary" :disabled="gitSaving" @click="saveGitTtl">
          {{ gitSaving ? '保存中' : '保存' }}
        </button>
        <span v-if="gitMsg" class="text-xs" :class="gitMsg === 'saved' ? 'text-emerald-300' : 'text-red-300'">
          {{ gitMsg === 'saved' ? '已保存' : gitMsg }}
        </span>
        <span class="text-xs text-slate-600">新 TTL 仅对之后写入的缓存生效，已有缓存按原时间过期。</span>
      </div>
    </Panel>

    <div v-if="!loading" class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>镜像</th>
            <th>状态</th>
            <th>缓存 TTL</th>
            <th>上游地址</th>
            <th>操作</th>
            <th>保存</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in mirrors" :key="m.name" :class="{ 'opacity-60': updating === m.name }">
            <td>
              <div class="font-semibold text-slate-100">{{ m.name }}</div>
              <div class="text-xs text-slate-600">/{{ m.name }}</div>
            </td>
            <td>
              <span class="tag" :class="m.enabled ? 'tag-ok' : 'tag-off'">
                <StatusDot :tone="m.enabled ? 'ok' : 'off'" />
                {{ m.enabled ? '已启用' : '已停用' }}
              </span>
            </td>
            <td>
              <template v-if="!isNoCache(m.name)">
                <select
                  v-model="m.cacheTTL"
                  class="input w-32"
                  :disabled="updating === m.name"
                  @change="updateTTL(m)"
                >
                  <option v-for="opt in TTL_OPTIONS" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
                <div v-if="!TTL_OPTIONS.some(o => o.value === m.cacheTTL)" class="mt-1 text-xs text-slate-500">
                  当前：{{ ttlText(m.cacheTTL) }}
                </div>
              </template>
              <span v-else class="text-xs text-slate-600">该镜像不缓存</span>
            </td>
            <td class="min-w-[360px]">
              <input
                v-model="m.upstream"
                class="input w-full"
                :disabled="updating === m.name"
                placeholder="上游地址"
                @change="updateUpstream(m)"
                @keydown.enter="($event.target as HTMLInputElement).blur()"
              />
            </td>
            <td>
              <button class="btn" :class="m.enabled ? 'btn-danger' : 'btn-primary'" :disabled="updating === m.name" @click="toggleMirror(m)">
                {{ m.enabled ? '停用' : '启用' }}
              </button>
            </td>
            <td class="text-xs" :class="saved === m.name ? 'text-emerald-300' : 'text-slate-600'">
              {{ saved === m.name ? '已保存' : updating === m.name ? '保存中' : '-' }}
            </td>
          </tr>
        </tbody>
      </table>
      <EmptyState v-if="!mirrors.length" message="暂无镜像配置。" />
    </div>
  </div>
</template>
