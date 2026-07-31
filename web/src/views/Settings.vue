<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getPublicConfig, getRateLimitConfig, updateRateLimitConfig } from '../api/client'
import Panel from '../components/Panel.vue'
import Banner from '../components/Banner.vue'
import StatusDot from '../components/StatusDot.vue'

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
  // 与后端校验保持一致：rate 必须为正、interval 必须可解析
  if (!Number.isInteger(rlRate.value) || rlRate.value <= 0) {
    rlMsg.value = '最大请求数必须为正整数'
    rlSaving.value = false
    return
  }
  if (!/^\d+(ms|s|m|h|d)$/.test(rlInterval.value.trim())) {
    rlMsg.value = '时间窗口格式无效（如 3h、30m、1d）'
    rlSaving.value = false
    return
  }
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
    rlMsg.value = e.response?.statusText || '保存失败'
  }
  rlSaving.value = false
}

const capabilities = [
  '流量分析',
  'GitHub API 代理',
  'Web 界面鉴权',
  '日志自动清理',
  'Docker 仓库鉴权',
  '镜像搜索',
  'IP 限流',
  'HuggingFace 代理',
  '发行版下载',
]
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">系统</span>
      <h1 class="page-title">设置</h1>
      <p class="page-subtitle">运行信息与访问治理配置。</p>
    </section>

    <div v-if="loading" class="glass flex items-center gap-2 p-4 text-sm text-cyan-400">
      <StatusDot tone="accent" pulse /> 正在加载设置...
    </div>

    <Banner v-if="rlLoadError" :message="rlLoadError" />

    <div v-if="!loading" class="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <Panel>
        <template #title>限流配置</template>
        <template #desc>滚动时间窗口限流，白名单绕过，黑名单直接拒绝。</template>
        <template #action>
          <button class="btn" :class="rlEnabled ? 'btn-primary' : ''" @click="rlEnabled = !rlEnabled">
            <StatusDot :tone="rlEnabled ? 'ok' : 'off'" :pulse="rlEnabled" />
            {{ rlEnabled ? '已启用' : '已停用' }}
          </button>
        </template>

        <div class="grid gap-4">
          <label class="grid gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-slate-500">每个时间窗口最大请求数</span>
            <input v-model.number="rlRate" type="number" min="1" class="input w-full" />
          </label>
          <label class="grid gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-slate-500">时间窗口</span>
            <input v-model="rlInterval" type="text" class="input w-full" placeholder="3h, 30m, 1d" />
          </label>
          <label class="grid gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-slate-500">白名单</span>
            <input v-model="rlWhitelist" type="text" class="input w-full" placeholder="10.0.0.1, 192.168.1.0/24" />
          </label>
          <label class="grid gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-slate-500">黑名单</span>
            <input v-model="rlBlacklist" type="text" class="input w-full" placeholder="1.2.3.4" />
          </label>
          <div class="flex items-center gap-3">
            <button class="btn btn-primary" :disabled="rlSaving" @click="saveRateLimit">
              {{ rlSaving ? '保存中' : '保存配置' }}
            </button>
            <span v-if="rlMsg" class="text-xs" :class="rlMsg === 'saved' ? 'text-emerald-300' : 'text-red-300'">{{ rlMsg === 'saved' ? '已保存' : rlMsg }}</span>
          </div>
        </div>
      </Panel>

      <aside class="grid gap-5">
        <Panel>
          <template #title>运行信息</template>
          <dl class="space-y-3 text-sm">
            <div class="flex justify-between gap-4 border-b border-slate-800/70 pb-2">
              <dt class="text-slate-500">应用</dt>
              <dd class="text-slate-100">DevBox</dd>
            </div>
            <div class="flex justify-between gap-4 border-b border-slate-800/70 pb-2">
              <dt class="text-slate-500">版本</dt>
              <dd class="text-slate-100">v1.1.0</dd>
            </div>
            <div class="flex justify-between gap-4 border-b border-slate-800/70 pb-2">
              <dt class="text-slate-500">后端</dt>
              <dd class="text-slate-300">Go + SQLite</dd>
            </div>
            <div class="flex justify-between gap-4 border-b border-slate-800/70 pb-2">
              <dt class="text-slate-500">前端</dt>
              <dd class="text-slate-300">Vue 3 + TailwindCSS</dd>
            </div>
            <div v-if="publicUrl" class="grid gap-1 border-b border-slate-800/70 pb-2">
              <dt class="text-slate-500">公网地址</dt>
              <dd class="truncate text-cyan-300">{{ publicUrl }}</dd>
            </div>
            <div class="flex justify-between gap-4">
              <dt class="text-slate-500">源码</dt>
              <dd><a href="https://github.com/wha7ev9r/devbox" target="_blank" class="muted-link">github</a></dd>
            </div>
          </dl>
        </Panel>

        <Panel>
          <template #title>功能特性</template>
          <div class="flex flex-wrap gap-2">
            <span v-for="item in capabilities" :key="item" class="tag tag-ok">
              <svg class="h-3 w-3" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <polyline points="2 6.2 4.5 8.5 10 3.5" />
              </svg>
              {{ item }}
            </span>
          </div>
        </Panel>
      </aside>
    </div>
  </div>
</template>
