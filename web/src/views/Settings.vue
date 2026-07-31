<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getPublicConfig, getRateLimitConfig, updateRateLimitConfig } from '../api/client'

const publicUrl = ref('')
const rlEnabled = ref(false)
const rlRate = ref(500)
const rlInterval = ref('3h')
const rlWhitelist = ref('')
const rlBlacklist = ref('')
const rlSaving = ref(false)
const rlMsg = ref('')
const rlLoadError = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    const config = await getPublicConfig()
    publicUrl.value = config.publicUrl || ''
  } catch { /* ignore - non-critical */ }
  try {
    const rl = await getRateLimitConfig()
    rlEnabled.value = rl.enabled
    rlRate.value = rl.rate
    rlInterval.value = rl.interval || '3h'
    rlWhitelist.value = (rl.whitelist || []).join(', ')
    rlBlacklist.value = (rl.blacklist || []).join(', ')
  } catch (e: any) {
    rlLoadError.value = e.response?.statusText || '加载限流配置失败'
  }
  loading.value = false
})

async function saveRateLimit() {
  rlSaving.value = true
  rlMsg.value = ''
  try {
    const wl = rlWhitelist.value.split(',').map(s => s.trim()).filter(Boolean)
    const bl = rlBlacklist.value.split(',').map(s => s.trim()).filter(Boolean)
    const res = await updateRateLimitConfig({ enabled: rlEnabled.value, rate: rlRate.value, interval: rlInterval.value, whitelist: wl, blacklist: bl })
    rlEnabled.value = res.enabled
    rlRate.value = res.rate
    rlInterval.value = res.interval || rlInterval.value
    rlMsg.value = 'saved'
    setTimeout(() => rlMsg.value = '', 1500)
  } catch (e: any) {
    rlMsg.value = e.response?.statusText || 'save failed'
  }
  rlSaving.value = false
}

const capabilities = [
  'Traffic Analytics',
  'GitHub API Proxy',
  'Web UI Auth',
  'Log Auto Cleanup',
  'Docker Registry Auth',
  'Mirror Search',
  'IP Rate Limiting',
  'HuggingFace Proxy',
  'Release Downloads',
]
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">system</span>
      <h1 class="page-title">Settings</h1>
      <p class="page-subtitle">运行信息与访问治理配置。</p>
    </section>

    <div v-if="loading" class="panel-pad text-sm text-cyan-400">loading settings...</div>

    <div v-if="rlLoadError" class="mb-4 border border-red-500/40 bg-red-950/30 px-3 py-2 text-sm text-red-300">
      {{ rlLoadError }}
    </div>

    <div v-if="!loading" class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <section class="panel-pad">
        <div class="mb-4 flex items-center justify-between gap-3 border-b border-slate-900 pb-3">
          <div>
            <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">rate limit</h2>
            <p class="mt-1 text-xs text-slate-600">滚动时间窗口限流，白名单绕过，黑名单直接拒绝。</p>
          </div>
          <button class="btn" :class="rlEnabled ? 'btn-primary' : ''" @click="rlEnabled = !rlEnabled">
            {{ rlEnabled ? 'enabled' : 'disabled' }}
          </button>
        </div>

        <div class="grid gap-4">
          <label class="grid gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-slate-500">max requests per rolling window</span>
            <input v-model.number="rlRate" type="number" min="1" class="input w-full" />
          </label>
          <label class="grid gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-slate-500">rolling window</span>
            <input v-model="rlInterval" type="text" class="input w-full" placeholder="3h, 30m, 1d" />
          </label>
          <label class="grid gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-slate-500">whitelist</span>
            <input v-model="rlWhitelist" type="text" class="input w-full" placeholder="10.0.0.1, 192.168.1.0/24" />
          </label>
          <label class="grid gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-slate-500">blacklist</span>
            <input v-model="rlBlacklist" type="text" class="input w-full" placeholder="1.2.3.4" />
          </label>
          <div class="flex items-center gap-3">
            <button class="btn btn-primary" :disabled="rlSaving" @click="saveRateLimit">
              {{ rlSaving ? 'saving' : 'save config' }}
            </button>
            <span v-if="rlMsg" class="text-xs" :class="rlMsg === 'saved' ? 'text-emerald-300' : 'text-red-300'">{{ rlMsg }}</span>
          </div>
        </div>
      </section>

      <aside class="grid gap-5">
        <section class="panel-pad">
          <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">runtime</h2>
          <dl class="mt-4 space-y-3 text-sm">
            <div class="flex justify-between gap-4 border-b border-slate-900 pb-2">
              <dt class="text-slate-500">app</dt>
              <dd class="text-slate-100">DevBox</dd>
            </div>
            <div class="flex justify-between gap-4 border-b border-slate-900 pb-2">
              <dt class="text-slate-500">version</dt>
              <dd class="text-slate-100">v1.1.0</dd>
            </div>
            <div class="flex justify-between gap-4 border-b border-slate-900 pb-2">
              <dt class="text-slate-500">backend</dt>
              <dd class="text-slate-300">Go + SQLite</dd>
            </div>
            <div class="flex justify-between gap-4 border-b border-slate-900 pb-2">
              <dt class="text-slate-500">frontend</dt>
              <dd class="text-slate-300">Vue 3 + TailwindCSS</dd>
            </div>
            <div v-if="publicUrl" class="grid gap-1 border-b border-slate-900 pb-2">
              <dt class="text-slate-500">public url</dt>
              <dd class="truncate text-cyan-300">{{ publicUrl }}</dd>
            </div>
            <div class="flex justify-between gap-4">
              <dt class="text-slate-500">source</dt>
              <dd><a href="https://github.com/wha7ev9r/devbox" target="_blank" class="muted-link">github</a></dd>
            </div>
          </dl>
        </section>

        <section class="panel-pad">
          <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">capabilities</h2>
          <div class="mt-4 flex flex-wrap gap-2">
            <span v-for="item in capabilities" :key="item" class="tag tag-ok">{{ item }}</span>
          </div>
        </section>
      </aside>
    </div>
  </div>
</template>
