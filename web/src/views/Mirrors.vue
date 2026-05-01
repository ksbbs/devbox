<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMirrorConfig, updateMirrorConfig } from '../api/client'

const mirrors = ref<any[]>([])
const loading = ref(true)
const updating = ref<string | null>(null)

onMounted(async () => {
  try {
    mirrors.value = await getMirrorConfig()
  } catch { mirrors.value = [] }
  loading.value = false
})

async function toggleMirror(m: any) {
  updating.value = m.name
  try {
    await updateMirrorConfig(m.name, !m.enabled)
    m.enabled = !m.enabled
  } finally {
    updating.value = null
  }
}

async function updateUpstream(m: any) {
  updating.value = m.name
  try {
    await updateMirrorConfig(m.name, m.enabled, m.upstream)
  } finally {
    updating.value = null
  }
}
</script>

<template>
  <div>
    <div class="mb-8">
      <h1 class="text-4xl font-bold mb-2">
        <span class="bg-gradient-to-r from-violet-400 via-fuchsia-400 to-pink-400 bg-clip-text text-transparent">
          Mirrors
        </span>
      </h1>
      <p class="text-slate-400">配置和管理镜像加速服务</p>
    </div>

    <div v-if="loading" class="flex items-center gap-3 text-slate-400 py-12">
      <div class="w-8 h-8 border-2 border-violet-500/30 border-t-violet-400 rounded-full animate-spin"></div>
      <span>加载中...</span>
    </div>

    <TransitionGroup v-else appear name="card" tag="div" class="space-y-4">
      <div v-for="m in mirrors" :key="m.name"
        class="config-card group relative p-6 rounded-2xl overflow-hidden"
        :class="{ 'opacity-75': updating === m.name }">

        <!-- 侧边发光条 -->
        <div class="absolute left-0 top-0 bottom-0 w-1 transition-all duration-300"
          :class="m.enabled ? 'bg-gradient-to-b from-emerald-400 to-sky-400' : 'bg-slate-600'"></div>

        <div class="flex items-center gap-6 pl-4">
          <!-- 图标 -->
          <div class="w-14 h-14 rounded-2xl flex items-center justify-center text-2xl font-bold shrink-0 transition-all duration-300"
            :class="m.enabled
              ? 'bg-gradient-to-br from-sky-500 to-violet-500 text-white shadow-lg shadow-sky-500/25'
              : 'bg-slate-700/50 text-slate-500'">
            {{ m.name[0].toUpperCase() }}
          </div>

          <!-- 信息 -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-3 mb-2">
              <h3 class="text-xl font-semibold text-white">{{ m.name }}</h3>
              <span class="px-2 py-0.5 rounded-full text-xs font-medium"
                :class="m.enabled ? 'bg-emerald-500/20 text-emerald-400' : 'bg-slate-600/30 text-slate-500'">
                {{ m.enabled ? '已启用' : '已禁用' }}
              </span>
            </div>
            <div class="flex items-center gap-4 text-sm text-slate-400">
              <span>Cache TTL: <span class="text-slate-300">{{ m.cacheTTL }}s</span></span>
            </div>
            <div class="mt-3 flex items-center gap-3">
              <input v-model="m.upstream" @change="updateUpstream(m)"
                class="flex-1 bg-slate-900/50 border border-slate-700 rounded-xl px-4 py-2.5 text-sm text-slate-200 focus:border-sky-500/50 focus:outline-none focus:ring-2 focus:ring-sky-500/20 transition-all"
                :disabled="updating === m.name"
                placeholder="Upstream URL" />
            </div>
          </div>

          <!-- 开关 -->
          <button @click="toggleMirror(m)"
            class="relative w-16 h-8 rounded-full transition-colors duration-300 focus:outline-none"
            :class="m.enabled ? 'bg-emerald-500/30' : 'bg-slate-700'"
            :disabled="updating === m.name">
            <span class="absolute top-1 left-1 w-6 h-6 rounded-full transition-all duration-300 flex items-center justify-center text-xs"
              :class="m.enabled
                ? 'translate-x-8 bg-gradient-to-br from-emerald-400 to-emerald-500 text-white shadow-lg'
                : 'translate-x-0 bg-slate-500 text-slate-300'">
              {{ m.enabled ? '✓' : '✕' }}
            </span>
          </button>
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<style>
.config-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.04) 0%, rgba(255, 255, 255, 0.01) 100%);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(10px);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.config-card:hover {
  border-color: rgba(139, 92, 246, 0.3);
  transform: translateX(4px);
  box-shadow:
    0 20px 40px rgba(0, 0, 0, 0.3),
    0 0 40px rgba(139, 92, 246, 0.1);
}

/* 进入动画 */
.card-enter-from {
  opacity: 0;
  transform: translateX(-30px);
}

.card-enter-active {
  transition: all 0.5s cubic-bezier(0.4, 0, 0.2, 1);
}

.card-enter-to {
  opacity: 1;
  transform: translateX(0);
}
</style>