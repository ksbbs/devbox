<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getPublicConfig, getRateLimitConfig, updateRateLimitConfig } from '../api/client'

const publicUrl = ref('')
const rlEnabled = ref(false)
const rlRate = ref(500)
const rlWhitelist = ref('')
const rlBlacklist = ref('')
const rlSaving = ref(false)
const rlMsg = ref('')

onMounted(async () => {
  try {
    const config = await getPublicConfig()
    publicUrl.value = config.publicUrl || ''
  } catch { /* ignore */ }
  try {
    const rl = await getRateLimitConfig()
    rlEnabled.value = rl.enabled
    rlRate.value = rl.rate
    rlWhitelist.value = (rl.whitelist || []).join(', ')
    rlBlacklist.value = (rl.blacklist || []).join(', ')
  } catch { /* ignore — no auth or not available */ }
})

async function saveRateLimit() {
  rlSaving.value = true
  rlMsg.value = ''
  try {
    const wl = rlWhitelist.value.split(',').map(s => s.trim()).filter(Boolean)
    const bl = rlBlacklist.value.split(',').map(s => s.trim()).filter(Boolean)
    const res = await updateRateLimitConfig({ enabled: rlEnabled.value, rate: rlRate.value, whitelist: wl, blacklist: bl })
    rlEnabled.value = res.enabled
    rlRate.value = res.rate
    rlMsg.value = '已保存'
    setTimeout(() => rlMsg.value = '', 2000)
  } catch (e: any) {
    rlMsg.value = '保存失败'
  }
  rlSaving.value = false
}

const features = [
  { name: 'Traffic Analytics', status: 'stable', icon: '◎' },
  { name: 'GitHub API Proxy', status: 'stable', icon: '◆' },
  { name: 'Web UI Auth', status: 'stable', icon: '▣' },
  { name: 'Log Auto Cleanup', status: 'stable', icon: '▦' },
  { name: 'Docker Registry Auth', status: 'stable', icon: '⬡' },
  { name: 'Mirror Search', status: 'stable', icon: '◇' },
  { name: 'IP Rate Limiting', status: 'stable', icon: '⊗' },
  { name: 'HuggingFace Proxy', status: 'stable', icon: '△' },
  { name: 'PWA Support', status: 'planned', icon: '◈' },
  { name: 'Auto Update Notify', status: 'planned', icon: '◉' },
  { name: 'Plugin System', status: 'planned', icon: '◐' }
]

const statusColors: Record<string, string> = {
  planned: 'bg-slate-600/30 text-slate-400 border-slate-500/30',
  beta: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
  stable: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30'
}

const statusText: Record<string, string> = {
  planned: '规划中',
  beta: '测试中',
  stable: '已上线'
}
</script>

<template>
  <div>
    <div class="mb-8">
      <h1 class="text-4xl font-bold mb-2">
        <span class="bg-gradient-to-r from-rose-400 via-pink-400 to-fuchsia-400 bg-clip-text text-transparent">
          Settings
        </span>
      </h1>
      <p class="text-slate-400">系统设置与功能规划</p>
    </div>

    <div class="grid gap-6">
      <!-- 版本信息 -->
      <div class="settings-card p-6 rounded-2xl">
        <h3 class="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-xl">◉</span>
          版本信息
        </h3>
        <div class="flex items-center gap-6">
          <div class="w-16 h-16 rounded-2xl bg-gradient-to-br from-sky-500 via-violet-500 to-fuchsia-500 flex items-center justify-center shadow-lg shadow-violet-500/20">
            <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"></path>
            </svg>
          </div>
          <div>
            <div class="text-2xl font-bold text-white">DevBox</div>
            <div class="text-slate-400">v1.1.0</div>
            <div class="text-xs text-emerald-400 mt-1">✓ 运行正常</div>
              <div v-if="publicUrl" class="text-xs text-sky-400 mt-1 font-mono">{{ publicUrl }}</div>
          </div>
        </div>
      </div>

      <!-- 限流设置 -->
      <div class="settings-card p-6 rounded-2xl">
        <h3 class="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-xl">⊗</span>
          IP 限流设置
        </h3>
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <span class="text-slate-300">启用限流</span>
            <button @click="rlEnabled = !rlEnabled"
              class="px-4 py-2 rounded-lg text-sm font-medium transition-all"
              :class="rlEnabled ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : 'bg-white/5 text-slate-500 border border-white/10'">
              {{ rlEnabled ? '已启用' : '已关闭' }}
            </button>
          </div>
          <div v-if="rlEnabled" class="space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-slate-300">每时段最大请求</span>
              <input type="number" v-model="rlRate" min="1" class="w-32 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-slate-200 text-sm focus:border-sky-500/50 focus:outline-none" />
            </div>
            <div>
              <span class="text-slate-300 text-sm">白名单 IP（免限速，逗号分隔）</span>
              <input type="text" v-model="rlWhitelist" placeholder="如 10.0.0.1, 192.168.1.0/24" class="w-full mt-1 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-slate-200 text-sm focus:border-sky-500/50 focus:outline-none" />
            </div>
            <div>
              <span class="text-slate-300 text-sm">黑名单 IP（永远禁止，逗号分隔）</span>
              <input type="text" v-model="rlBlacklist" placeholder="如 1.2.3.4" class="w-full mt-1 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-slate-200 text-sm focus:border-sky-500/50 focus:outline-none" />
            </div>
            <button @click="saveRateLimit" :disabled="rlSaving"
              class="w-full py-2 rounded-lg bg-sky-500/20 text-sky-400 border border-sky-500/30 hover:bg-sky-500/30 transition-all disabled:opacity-50 text-sm font-medium">
              {{ rlSaving ? '保存中...' : '保存限流设置' }}
            </button>
            <div v-if="rlMsg" class="text-center text-sm" :class="rlMsg === '已保存' ? 'text-emerald-400' : 'text-red-400'">{{ rlMsg }}</div>
          </div>
        </div>
      </div>

      <!-- 未来功能 -->
      <div class="settings-card p-6 rounded-2xl">
        <h3 class="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-xl">◈</span>
          路线图
        </h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div v-for="f in features" :key="f.name"
            class="feature-item group flex items-center gap-4 p-4 rounded-xl transition-all duration-300"
            :class="statusColors[f.status]">
            <span class="text-xl opacity-60 group-hover:opacity-100 group-hover:scale-110 transition-all">{{ f.icon }}</span>
            <div class="flex-1">
              <div class="font-medium">{{ f.name }}</div>
            </div>
            <span class="text-xs px-2 py-1 rounded-full bg-black/20">{{ statusText[f.status] }}</span>
          </div>
        </div>
      </div>

      <!-- 系统信息 -->
      <div class="settings-card p-6 rounded-2xl">
        <h3 class="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-xl">◐</span>
          系统信息
        </h3>
        <div class="space-y-3 text-sm">
          <div class="flex justify-between items-center py-2 border-b border-white/5">
            <span class="text-slate-400">后端</span>
            <span class="text-slate-200 font-mono">Go + SQLite</span>
          </div>
          <div class="flex justify-between items-center py-2 border-b border-white/5">
            <span class="text-slate-400">前端</span>
            <span class="text-slate-200 font-mono">Vue 3 + TailwindCSS</span>
          </div>
          <div class="flex justify-between items-center py-2 border-b border-white/5">
            <span class="text-slate-400">部署方式</span>
            <span class="text-slate-200 font-mono">Docker</span>
          </div>
          <div v-if="publicUrl" class="flex justify-between items-center py-2 border-b border-white/5">
            <span class="text-slate-400">访问地址</span>
            <span class="text-sky-400 font-mono">{{ publicUrl }}</span>
          </div>
          <div class="flex justify-between items-center py-2">
            <span class="text-slate-400">源码</span>
            <a href="https://github.com/ksbbs/devbox" target="_blank"
              class="text-sky-400 hover:text-sky-300 transition-colors">
              github.com/ksbbs/devbox
            </a>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style>
.settings-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.04) 0%, rgba(255, 255, 255, 0.01) 100%);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(10px);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.settings-card:hover {
  border-color: rgba(244, 114, 182, 0.3);
  box-shadow:
    0 20px 40px rgba(0, 0, 0, 0.3),
    0 0 60px rgba(244, 114, 182, 0.1);
}

.feature-item {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid transparent;
}

.feature-item:hover {
  transform: translateX(4px);
  background: rgba(255, 255, 255, 0.05);
}
</style>