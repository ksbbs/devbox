<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getStatus, getTraffic, getPublicConfig } from '../api/client'

const mirrors = ref<any[]>([])
const loading = ref(true)
const traffic = ref<any[]>([])
const publicUrl = ref('')
const copiedUsage = ref<string | null>(null)

const stats = computed(() => {
  const total = mirrors.value.length
  const healthy = mirrors.value.filter(m => m.status === 'healthy' && m.enabled).length
  const enabled = mirrors.value.filter(m => m.enabled).length
  return { total, healthy, enabled }
})

onMounted(async () => {
  try {
    mirrors.value = await getStatus()
    traffic.value = await getTraffic()
  } catch (e) {
    mirrors.value = []
  }
  try {
    const config = await getPublicConfig()
    publicUrl.value = config.publicUrl || ''
  } catch { /* ignore */ }
  loading.value = false
})

function copyUsage(m: any) {
  navigator.clipboard.writeText(m.usage)
  copiedUsage.value = m.name
  setTimeout(() => copiedUsage.value = null, 2000)
}
</script>

<template>
  <div>
    <!-- 标题区域 -->
    <div class="mb-8">
      <h1 class="text-4xl font-bold mb-2">
        <span class="bg-gradient-to-r from-sky-400 via-violet-400 to-fuchsia-400 bg-clip-text text-transparent">
          Dashboard
        </span>
      </h1>
      <p class="text-slate-400">镜像加速服务状态实时监控</p>
      <p v-if="publicUrl" class="text-sm text-sky-400/80 mt-1 font-mono">{{ publicUrl }}</p>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
      <div class="glass-card p-6 rounded-2xl relative overflow-hidden group">
        <div class="absolute inset-0 bg-gradient-to-br from-sky-500/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
        <div class="relative">
          <div class="text-slate-400 text-sm mb-1">总镜像数</div>
          <div class="text-4xl font-bold text-white">{{ stats.total }}</div>
          <div class="mt-2 text-sky-400 text-sm">Registry Mirrors</div>
        </div>
        <div class="absolute top-4 right-4 w-12 h-12 rounded-xl bg-sky-500/10 flex items-center justify-center">
          <span class="text-2xl">🌐</span>
        </div>
      </div>

      <div class="glass-card p-6 rounded-2xl relative overflow-hidden group">
        <div class="absolute inset-0 bg-gradient-to-br from-emerald-500/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
        <div class="relative">
          <div class="text-slate-400 text-sm mb-1">健康状态</div>
          <div class="text-4xl font-bold text-emerald-400">{{ stats.healthy }}</div>
          <div class="mt-2 text-emerald-400/70 text-sm">Healthy</div>
        </div>
        <div class="absolute top-4 right-4 w-12 h-12 rounded-xl bg-emerald-500/10 flex items-center justify-center">
          <span class="text-2xl">✓</span>
        </div>
      </div>

      <div class="glass-card p-6 rounded-2xl relative overflow-hidden group">
        <div class="absolute inset-0 bg-gradient-to-br from-violet-500/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
        <div class="relative">
          <div class="text-slate-400 text-sm mb-1">已启用</div>
          <div class="text-4xl font-bold text-violet-400">{{ stats.enabled }}</div>
          <div class="mt-2 text-violet-400/70 text-sm">Enabled</div>
        </div>
        <div class="absolute top-4 right-4 w-12 h-12 rounded-xl bg-violet-500/10 flex items-center justify-center">
          <span class="text-2xl">⚡</span>
        </div>
      </div>
    </div>

    <!-- Mirror 状态卡片 -->
    <h2 class="text-xl font-semibold mb-4 text-slate-300">Mirror 状态</h2>
    <div v-if="loading" class="flex items-center gap-3 text-slate-400 py-12">
      <div class="w-8 h-8 border-2 border-sky-500/30 border-t-sky-400 rounded-full animate-spin"></div>
      <span>加载中...</span>
    </div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <TransitionGroup appear name="card">
        <div v-for="m in mirrors" :key="m.name"
          class="mirror-card group relative p-5 rounded-2xl overflow-hidden">
          <!-- 背景发光 -->
          <div class="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500"
            :class="m.status === 'healthy' && m.enabled ? 'glow-healthy' : m.enabled ? 'glow-warning' : 'glow-disabled'"></div>

          <!-- 内容 -->
          <div class="relative z-10">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-xl flex items-center justify-center text-lg font-bold"
                  :class="m.status === 'healthy' && m.enabled ? 'bg-emerald-500/20 text-emerald-400' : m.enabled ? 'bg-amber-500/20 text-amber-400' : 'bg-slate-600/30 text-slate-500'">
                  {{ m.name[0].toUpperCase() }}
                </div>
                <div>
                  <h3 class="font-semibold text-white">{{ m.name }}</h3>
                  <div class="text-xs text-slate-400">{{ m.pattern }}</div>
                </div>
              </div>
              <div class="status-dot"
                :class="m.status === 'healthy' && m.enabled ? 'status-healthy' : m.enabled ? 'status-unhealthy' : 'status-disabled'">
              </div>
            </div>

            <div class="space-y-2 text-sm">
              <div class="flex items-center gap-2">
                <span class="text-slate-500">Upstream:</span>
                <span class="text-slate-300 truncate">{{ m.upstream }}</span>
              </div>
              <div v-if="m.usage" class="flex items-center gap-2">
                <span class="text-slate-500 text-xs">Usage:</span>
                <code class="text-xs text-emerald-400 font-mono truncate">{{ m.usage }}</code>
                <button @click="copyUsage(m)"
                  class="shrink-0 px-2 py-1 rounded-md text-xs font-medium transition-all"
                  :class="copiedUsage === m.name
                    ? 'bg-emerald-500/20 text-emerald-400'
                    : 'bg-white/5 text-slate-400 hover:bg-white/10 hover:text-white'">
                  {{ copiedUsage === m.name ? '已复制!' : '复制' }}
                </button>
              </div>
              <div v-if="m.error" class="text-red-400 text-xs bg-red-500/10 rounded-lg px-3 py-2">
                {{ m.error }}
              </div>
            </div>

            <!-- 底部装饰线 -->
            <div class="mt-4 h-0.5 rounded-full overflow-hidden">
              <div class="h-full transition-all duration-1000"
                :class="m.status === 'healthy' && m.enabled ? 'w-full bg-gradient-to-r from-emerald-500 to-emerald-400' : m.enabled ? 'w-full bg-gradient-to-r from-amber-500 to-amber-400' : 'w-full bg-slate-600'">
              </div>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>

<style>
/* 玻璃卡片 */
.glass-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(10px);
  transition: all 0.3s ease;
}

.glass-card:hover {
  border-color: rgba(56, 189, 248, 0.2);
  transform: translateY(-2px);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
}

/* Mirror 卡片 */
.mirror-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.05) 0%, rgba(255, 255, 255, 0.02) 100%);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(10px);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.mirror-card:hover {
  transform: translateY(-4px) scale(1.01);
  border-color: rgba(56, 189, 248, 0.3);
}

/* 发光效果 */
.glow-healthy {
  background: radial-gradient(circle at 50% 0%, rgba(16, 185, 129, 0.15) 0%, transparent 70%);
}

.glow-warning {
  background: radial-gradient(circle at 50% 0%, rgba(245, 158, 11, 0.15) 0%, transparent 70%);
}

.glow-disabled {
  background: radial-gradient(circle at 50% 0%, rgba(100, 116, 139, 0.1) 0%, transparent 70%);
}

/* 状态指示点 */
.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  position: relative;
}

.status-dot::after {
  content: '';
  position: absolute;
  inset: -4px;
  border-radius: 50%;
  opacity: 0.5;
  animation: pulse 2s ease-in-out infinite;
}

.status-healthy {
  background: #10b981;
  box-shadow: 0 0 10px #10b981;
}

.status-healthy::after {
  background: #10b981;
}

.status-unhealthy {
  background: #f59e0b;
  box-shadow: 0 0 10px #f59e0b;
}

.status-unhealthy::after {
  background: #f59e0b;
}

.status-disabled {
  background: #64748b;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: 0.5; }
  50% { transform: scale(1.5); opacity: 0; }
}

/* 进入动画 */
.card-enter-from {
  opacity: 0;
  transform: translateY(30px) scale(0.95);
}

.card-enter-active {
  transition: all 0.5s cubic-bezier(0.4, 0, 0.2, 1);
}

.card-enter-to {
  opacity: 1;
  transform: translateY(0) scale(1);
}
</style>