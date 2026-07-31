<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMirrorConfig, updateMirrorConfig } from '../api/client'
import StatusDot from '../components/StatusDot.vue'
import EmptyState from '../components/EmptyState.vue'
import Banner from '../components/Banner.vue'

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
  if (!ttl) return '永不过期'
  if (ttl >= 86400) return `${Math.round(ttl / 86400)} 天`
  if (ttl >= 3600) return `${Math.round(ttl / 3600)} 小时`
  return `${ttl} 秒`
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
            <td class="text-slate-400">{{ ttlText(m.cacheTTL) }}</td>
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
