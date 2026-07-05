<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMirrorConfig, updateMirrorConfig } from '../api/client'

const mirrors = ref<any[]>([])
const loading = ref(true)
const errorMsg = ref('')
const updating = ref<string | null>(null)
const saved = ref<string | null>(null)

onMounted(async () => {
  try {
    mirrors.value = await getMirrorConfig()
  } catch (e: any) {
    mirrors.value = []
    errorMsg.value = e.response?.statusText || '加载镜像配置失败'
  }
  loading.value = false
})

async function toggleMirror(m: any) {
  updating.value = m.name
  errorMsg.value = ''
  try {
    await updateMirrorConfig(m.name, !m.enabled)
    m.enabled = !m.enabled
    flashSaved(m.name)
  } catch (e: any) {
    errorMsg.value = e.response?.statusText || '操作失败'
  } finally {
    updating.value = null
  }
}

async function updateUpstream(m: any) {
  updating.value = m.name
  errorMsg.value = ''
  try {
    await updateMirrorConfig(m.name, m.enabled, m.upstream)
    flashSaved(m.name)
  } catch (e: any) {
    errorMsg.value = e.response?.statusText || '保存失败'
  } finally {
    updating.value = null
  }
}

function flashSaved(name: string) {
  saved.value = name
  setTimeout(() => saved.value = null, 1500)
}

function ttlText(value: number | string) {
  const ttl = Number(value || 0)
  if (!ttl) return 'never'
  if (ttl >= 86400) return `${Math.round(ttl / 86400)}d`
  if (ttl >= 3600) return `${Math.round(ttl / 3600)}h`
  return `${ttl}s`
}
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">config</span>
      <h1 class="page-title">Mirrors</h1>
      <p class="page-subtitle">启停镜像服务并修改上游地址，保存后立即生效。</p>
    </section>

    <div v-if="loading" class="panel-pad text-sm text-cyan-400">loading mirrors...</div>

    <div v-if="errorMsg" class="mb-4 border border-red-500/40 bg-red-950/30 px-3 py-2 text-sm text-red-300">
      {{ errorMsg }}
    </div>

    <div v-else-if="!loading" class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>mirror</th>
            <th>state</th>
            <th>cache ttl</th>
            <th>upstream</th>
            <th>action</th>
            <th>save</th>
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
                {{ m.enabled ? 'enabled' : 'disabled' }}
              </span>
            </td>
            <td class="text-slate-400">{{ ttlText(m.cacheTTL) }}</td>
            <td class="min-w-[360px]">
              <input
                v-model="m.upstream"
                class="input w-full"
                :disabled="updating === m.name"
                placeholder="Upstream URL"
                @change="updateUpstream(m)"
                @keydown.enter="updateUpstream(m)"
              />
            </td>
            <td>
              <button class="btn" :class="m.enabled ? 'btn-danger' : 'btn-primary'" :disabled="updating === m.name" @click="toggleMirror(m)">
                {{ m.enabled ? 'disable' : 'enable' }}
              </button>
            </td>
            <td class="text-xs" :class="saved === m.name ? 'text-emerald-300' : 'text-slate-600'">
              {{ saved === m.name ? 'saved' : updating === m.name ? 'saving' : '-' }}
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!mirrors.length" class="p-6 text-center text-sm text-slate-500">暂无镜像配置。</div>
    </div>
  </div>
</template>
