<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getStatus, getTraffic, getPublicConfig, getRecentLogs } from '../api/client'
import StatusCard from '../components/StatusCard.vue'
import Panel from '../components/Panel.vue'
import CodeBlock from '../components/CodeBlock.vue'
import EmptyState from '../components/EmptyState.vue'
import Banner from '../components/Banner.vue'
import StatusDot from '../components/StatusDot.vue'

const mirrors = ref<any[]>([])
const loading = ref(true)
const errorMsg = ref('')
const traffic = ref<any[]>([])
const hourlyTraffic = ref<any[]>([])
const logs = ref<any[]>([])
const publicUrl = ref('')
const chartMode = ref<'requests' | 'bandwidth'>('requests')
const chartGranularity = ref<'hourly' | 'daily' | 'weekly'>('hourly')
const usageBaseUrl = computed(() => (publicUrl.value || window.location.origin).replace(/\/$/, ''))
const dockerHost = computed(() => usageBaseUrl.value.replace(/^https?:\/\//, ''))

const mirrorUsage = computed(() => [
  {
    title: 'npm',
    desc: '设置默认 registry 后直接 npm install。',
    cmd: `npm config set registry ${usageBaseUrl.value}/npm`,
  },
  {
    title: 'PyPI',
    desc: '设置 pip 全局 index-url。',
    cmd: `pip config set global.index-url ${usageBaseUrl.value}/pypi`,
  },
  {
    title: 'Docker Hub',
    desc: '写入 /etc/docker/daemon.json 的 registry-mirrors。',
    cmd: `{"registry-mirrors":["${usageBaseUrl.value}/docker"]}`,
  },
  {
    title: 'Go modules',
    desc: '设置 GOPROXY，失败时回落 direct。',
    cmd: `go env -w GOPROXY=${usageBaseUrl.value}/golang,direct`,
  },
  {
    title: 'GHCR',
    desc: '拉取 GitHub Container Registry 镜像。',
    cmd: `docker pull ${dockerHost.value}/ghcr/owner/image:tag`,
  },
  {
    title: 'HuggingFace',
    desc: '设置模型下载 endpoint。',
    cmd: `export HF_ENDPOINT=${usageBaseUrl.value}/hf`,
  },
])

const gitUsage = computed(() => [
  {
    title: 'GitHub 克隆',
    desc: '把 github.com/owner/repo 替换为 /gh/owner/repo。',
    cmd: `git clone ${usageBaseUrl.value}/gh/user/repo`,
  },
  {
    title: 'GitLab 克隆',
    desc: '把 gitlab.com/group/repo 替换为 /gl/group/repo。',
    cmd: `git clone ${usageBaseUrl.value}/gl/group/repo`,
  },
  {
    title: '压缩包下载',
    desc: '下载仓库压缩包。',
    cmd: `curl ${usageBaseUrl.value}/gh/user/repo/archive/main.zip -o main.zip`,
  },
  {
    title: '原始文件',
    desc: '读取仓库原始文件内容。',
    cmd: `curl ${usageBaseUrl.value}/gh/user/repo/raw/branch/file.txt`,
  },
])

const stats = computed(() => {
  const total = mirrors.value.length
  const healthy = mirrors.value.filter(m => m.status === 'healthy' && m.enabled).length
  const enabled = mirrors.value.filter(m => m.enabled).length
  const requests = traffic.value.reduce((sum, item) => sum + Number(item.requests || 0), 0)
  return { total, healthy, enabled, requests }
})

const hours = computed(() => [...new Set(hourlyTraffic.value.map((item: any) => item.hour))].sort().slice(-24))

const metricOf = (item: any) => chartMode.value === 'bandwidth'
  ? Number(item.bytes_out || item.bytesOut || 0)
  : Number(item.requests || 0)

const trendRows = computed(() => {
  if (!hourlyTraffic.value.length) {
    return traffic.value
      .map(item => ({
        mirror: item.mirror || 'unknown',
        total: metricOf(item),
        requests: Number(item.requests || 0),
        bytesOut: Number(item.bytes_out || item.bytesOut || 0),
        values: [metricOf(item)],
      }))
      .sort((a, b) => b.total - a.total)
      .slice(0, 8)
  }

  const groups = new Map<string, { mirror: string; byHour: Map<string, number>; requests: number; bytesOut: number }>()
  for (const item of hourlyTraffic.value) {
    const mirror = item.mirror || 'unknown'
    const group = groups.get(mirror) || { mirror, byHour: new Map(), requests: 0, bytesOut: 0 }
    group.byHour.set(item.hour, (group.byHour.get(item.hour) || 0) + metricOf(item))
    group.requests += Number(item.requests || 0)
    group.bytesOut += Number(item.bytes_out || item.bytesOut || 0)
    groups.set(mirror, group)
  }

  return [...groups.values()]
    .map(group => {
      const values = hours.value.map(hour => group.byHour.get(hour) || 0)
      return {
        mirror: group.mirror,
        total: values.reduce((sum, value) => sum + value, 0),
        requests: group.requests,
        bytesOut: group.bytesOut,
        values,
      }
    })
    .sort((a, b) => b.total - a.total)
    .slice(0, 8)
})

const totalTrend = computed(() => {
  if (!hours.value.length) return []
  const sums = new Map(hours.value.map(h => [h, 0]))
  for (const item of hourlyTraffic.value) {
    const h = item.hour
    if (sums.has(h)) sums.set(h, (sums.get(h) ?? 0) + metricOf(item))
  }
  return hours.value.map(h => sums.get(h) || 0)
})

const activeMirrors = computed(() => {
  if (!hourlyTraffic.value.length) {
    return traffic.value.filter(item => metricOf(item) > 0).length
  }
  const names = new Set(hourlyTraffic.value.filter(item => metricOf(item) > 0).map(item => item.mirror || 'unknown'))
  return names.size
})

const trendStats = computed(() => {
  const values = totalTrend.value
  if (!values.length) return { total: 0, peak: 0, avg: 0, active: 0 }
  const total = values.reduce((s, v) => s + v, 0)
  const peak = Math.max(...values)
  return {
    total,
    peak,
    avg: total / values.length,
    active: activeMirrors.value,
  }
})

const peakPos = computed(() => {
  const values = totalTrend.value
  if (!values.length) return null
  const idx = peakIndex(values)
  const x = values.length === 1 ? 100 : (idx / (values.length - 1)) * 100
  const max = Math.max(...values, 1)
  const y = 30 - (values[idx] / max) * 26
  return { x, y }
})

onMounted(async () => {
  const [statusRes, trafficRes, hourlyRes, logsRes, configRes] = await Promise.allSettled([
    getStatus(),
    getTraffic(),
    getTraffic(undefined, undefined, 'hourly'),
    getRecentLogs(50),
    getPublicConfig(),
  ])

  if (statusRes.status === 'fulfilled') mirrors.value = Array.isArray(statusRes.value) ? statusRes.value : []
  else errorMsg.value = '加载镜像状态失败'
  if (trafficRes.status === 'fulfilled') traffic.value = Array.isArray(trafficRes.value) ? trafficRes.value : []
  if (hourlyRes.status === 'fulfilled') hourlyTraffic.value = Array.isArray(hourlyRes.value) ? hourlyRes.value : []
  if (logsRes.status === 'fulfilled') logs.value = Array.isArray(logsRes.value) ? logsRes.value : []
  if (configRes.status === 'fulfilled') publicUrl.value = configRes.value.publicUrl || ''
  loading.value = false
})

async function switchChartMode() {
  chartMode.value = chartMode.value === 'requests' ? 'bandwidth' : 'requests'
}

let granularityRequestId = 0

async function switchGranularity(level: 'hourly' | 'daily' | 'weekly') {
  const requestId = ++granularityRequestId
  try {
    const data = await getTraffic(undefined, undefined, level)
    if (requestId !== granularityRequestId) return
    hourlyTraffic.value = Array.isArray(data) ? data : []
    chartGranularity.value = level
  } catch (e: any) {
    if (requestId === granularityRequestId) {
      errorMsg.value = e.response?.statusText || '流量数据加载失败'
    }
  }
}

async function refreshLogs() {
  logs.value = await getRecentLogs(50)
}

function formatBytes(b: number) {
  const value = Number(b || 0)
  if (value < 1024) return value + ' B'
  if (value < 1024 * 1024) return (value / 1024).toFixed(1) + ' KB'
  if (value < 1024 * 1024 * 1024) return (value / 1024 / 1024).toFixed(1) + ' MB'
  return (value / 1024 / 1024 / 1024).toFixed(1) + ' GB'
}

function formatMetric(value: number) {
  return chartMode.value === 'bandwidth' ? formatBytes(value) : Math.round(value).toLocaleString()
}

function sparklinePoints(values: number[]) {
  const source = values.length > 1 ? values : [0, values[0] || 0]
  const max = Math.max(...source, 1)
  return source
    .map((value, index) => {
      const x = source.length === 1 ? 100 : (index / (source.length - 1)) * 100
      const y = 28 - (value / max) * 24
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')
}

function sparkAreaPoints(values: number[]) {
  const points = sparklinePoints(values)
  return `${points} 100,32 0,32`
}

function bigChartPoints(values: number[]) {
  const source = values.length > 1 ? values : [0, values[0] || 0]
  const max = Math.max(...source, 1)
  return source
    .map((value, index) => {
      const x = (index / (source.length - 1)) * 100
      const y = 30 - (value / max) * 26
      return `${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(' ')
}

function bigAreaPoints(values: number[]) {
  return `${bigChartPoints(values)} 100,32 0,32`
}

function peakIndex(values: number[]) {
  if (!values.length) return -1
  let idx = 0
  values.forEach((v, i) => { if (v > values[idx]) idx = i })
  return idx
}
</script>

<template>
  <div>
    <svg width="0" height="0" class="absolute" aria-hidden="true">
      <defs>
        <linearGradient id="sparkStroke" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stop-color="#7c3aed"/>
          <stop offset="100%" stop-color="#67e8f9"/>
        </linearGradient>
        <linearGradient id="sparkFill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="rgba(103,232,249,0.22)"/>
          <stop offset="100%" stop-color="rgba(103,232,249,0)"/>
        </linearGradient>
      </defs>
    </svg>

    <section class="page-header">
      <span class="page-kicker">运行时</span>
      <div class="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="page-title">仪表盘</h1>
          <p class="page-subtitle">镜像加速服务状态、流量趋势与访问日志。</p>
        </div>
        <code v-if="publicUrl" class="code-line w-fit max-w-full truncate">{{ publicUrl }}</code>
      </div>
    </section>

    <Banner v-if="errorMsg" :message="errorMsg" />

    <section class="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
      <div class="glass glass-hover min-w-0 p-4">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="text-[10px] uppercase tracking-[0.2em] text-slate-500">镜像源</div>
            <div class="mt-2 text-3xl font-semibold text-slate-100">{{ stats.total }}</div>
          </div>
          <svg class="mt-0.5 h-6 w-6 shrink-0 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" aria-hidden="true">
            <path d="M12 3 19.5 7.5 12 12 4.5 7.5 12 3Z" stroke-linejoin="round"/>
            <path d="M12 12v9M19.5 7.5V16.5M4.5 7.5V16.5" stroke-linecap="round"/>
          </svg>
        </div>
      </div>
      <div class="glass glass-hover min-w-0 p-4">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="text-[10px] uppercase tracking-[0.2em] text-slate-500">健康</div>
            <div class="mt-2 text-3xl font-semibold text-emerald-300">{{ stats.healthy }}</div>
          </div>
          <svg class="mt-0.5 h-6 w-6 shrink-0 text-emerald-400/70" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M3 12h4l2.5-6 4 12 2.5-6h5" />
          </svg>
        </div>
      </div>
      <div class="glass glass-hover min-w-0 p-4">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="text-[10px] uppercase tracking-[0.2em] text-slate-500">已启用</div>
            <div class="mt-2 text-3xl font-semibold text-cyan-300">{{ stats.enabled }}</div>
          </div>
          <svg class="mt-0.5 h-6 w-6 shrink-0 text-cyan-400/70" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M12 4v5l-4 3-1 6h10l-1-6-4-3V4" />
            <path d="M9 2h6" />
          </svg>
        </div>
      </div>
      <div class="glass glass-hover min-w-0 p-4">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="text-[10px] uppercase tracking-[0.2em] text-slate-500">请求数</div>
            <div class="mt-2 text-3xl font-semibold text-slate-100">{{ stats.requests.toLocaleString() }}</div>
          </div>
          <svg class="mt-0.5 h-6 w-6 shrink-0 text-violet-400/80" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" aria-hidden="true">
            <path d="M4 17v-4M9.5 17V9M15 17v-6M20.5 17V5" />
            <path d="M3 20h18" />
          </svg>
        </div>
      </div>
    </section>

    <section class="mb-5 grid grid-cols-1 gap-5 xl:grid-cols-2">
      <Panel>
        <template #title>镜像加速</template>
        <template #desc>复制命令后把示例包名或镜像名替换成你的目标。</template>
        <template #action><span class="tag tag-ok">软件包</span></template>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <article v-for="item in mirrorUsage" :key="item.title" class="glass-inset min-w-0 p-3">
            <div class="mb-2">
              <h3 class="text-sm font-semibold text-slate-100">{{ item.title }}</h3>
              <p class="mt-1 text-xs text-slate-600">{{ item.desc }}</p>
            </div>
            <CodeBlock :code="item.cmd" />
          </article>
        </div>
      </Panel>

      <Panel>
        <template #title>Git 加速</template>
        <template #desc>GitHub 走 /gh/，GitLab 走 /gl/，支持克隆、压缩包和原始文件。</template>
        <template #action><span class="tag tag-ok">git</span></template>
        <div class="grid grid-cols-1 gap-3">
          <article v-for="item in gitUsage" :key="item.title" class="glass-inset min-w-0 p-3">
            <div class="mb-2">
              <h3 class="text-sm font-semibold text-slate-100">{{ item.title }}</h3>
              <p class="mt-1 text-xs text-slate-600">{{ item.desc }}</p>
            </div>
            <CodeBlock :code="item.cmd" />
          </article>
        </div>
      </Panel>
    </section>

    <section class="mb-5">
      <div class="mb-3 flex items-center justify-between gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">镜像状态</h2>
        <span v-if="loading" class="flex items-center gap-2 text-xs text-cyan-400">
          <StatusDot tone="accent" pulse /> 加载中...
        </span>
      </div>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <StatusCard v-for="m in mirrors" :key="m.name" :mirror="m" />
      </div>
      <div v-if="!loading && !mirrors.length">
        <EmptyState message="暂无镜像状态。" />
      </div>
    </section>

    <section class="mb-5 grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div class="min-w-0">
        <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
          <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">流量趋势</h2>
          <div class="flex gap-2">
            <select class="select w-auto text-xs" :value="chartGranularity" @change="switchGranularity(($event.target as HTMLSelectElement).value as any)">
              <option value="hourly">小时</option>
              <option value="daily">天</option>
              <option value="weekly">周</option>
            </select>
            <button class="btn" @click="switchChartMode">
              {{ chartMode === 'requests' ? '指标：请求数' : '指标：带宽' }}
            </button>
          </div>
        </div>

        <div class="glass mb-4 p-4">
          <div class="mb-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
            <div>
              <div class="text-[10px] uppercase tracking-[0.18em] text-slate-500">总{{ chartMode === 'requests' ? '请求' : '流量' }}</div>
              <div class="mt-1 text-lg font-semibold text-slate-100">{{ formatMetric(trendStats.total) }}</div>
            </div>
            <div>
              <div class="text-[10px] uppercase tracking-[0.18em] text-slate-500">峰值</div>
              <div class="mt-1 text-lg font-semibold text-cyan-300">{{ formatMetric(trendStats.peak) }}</div>
            </div>
            <div>
              <div class="text-[10px] uppercase tracking-[0.18em] text-slate-500">均值</div>
              <div class="mt-1 text-lg font-semibold text-violet-300">{{ formatMetric(trendStats.avg) }}</div>
            </div>
            <div>
              <div class="text-[10px] uppercase tracking-[0.18em] text-slate-500">活跃镜像</div>
              <div class="mt-1 text-lg font-semibold text-emerald-300">{{ trendStats.active }}</div>
            </div>
          </div>
          <div v-if="totalTrend.length" class="relative">
            <svg viewBox="0 0 100 32" class="h-32 w-full" preserveAspectRatio="none">
              <line v-for="gy in [0, 8, 16, 24, 32]" :key="gy" x1="0" :y1="gy" x2="100" :y2="gy" stroke="rgba(148,163,184,0.09)" stroke-width="0.2" vector-effect="non-scaling-stroke" />
              <polygon :points="bigAreaPoints(totalTrend)" fill="url(#sparkFill)" />
              <polyline :points="bigChartPoints(totalTrend)" fill="none" stroke="url(#sparkStroke)" stroke-width="0.45" vector-effect="non-scaling-stroke" />
              <circle v-if="peakPos" :cx="peakPos.x" :cy="peakPos.y" r="0.9" fill="#67e8f9" />
            </svg>
            <div class="pointer-events-none absolute left-0 top-0 h-full w-full" />
          </div>
          <EmptyState v-else message="暂无流量数据，产生访问后这里会显示趋势图。" />
        </div>

        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>镜像</th>
                <th>{{ chartMode === 'requests' ? '请求数' : '输出流量' }}</th>
                <th>趋势图</th>
                <th>总请求数</th>
                <th>总输出</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in trendRows" :key="row.mirror">
                <td class="font-semibold text-slate-100">{{ row.mirror }}</td>
                <td class="text-cyan-300">{{ formatMetric(row.total) }}</td>
                <td class="w-44">
                  <svg viewBox="0 0 100 32" class="h-8 w-36" preserveAspectRatio="none">
                    <polygon :points="sparkAreaPoints(row.values)" fill="url(#sparkFill)" />
                    <polyline :points="sparklinePoints(row.values)" fill="none" stroke="url(#sparkStroke)" stroke-width="2" vector-effect="non-scaling-stroke" />
                  </svg>
                </td>
                <td class="text-slate-400">{{ row.requests.toLocaleString() }}</td>
                <td class="text-slate-400">{{ formatBytes(row.bytesOut) }}</td>
              </tr>
            </tbody>
          </table>
          <EmptyState v-if="!trendRows.length" message="暂无流量数据。" />
        </div>
      </div>

      <Panel>
        <template #title>健康概览</template>
        <div class="space-y-3 text-sm">
          <div class="flex justify-between border-b border-slate-800/70 pb-2">
            <span class="text-slate-500">健康占比</span>
            <span class="text-emerald-300">{{ stats.total ? Math.round(stats.healthy / stats.total * 100) : 0 }}%</span>
          </div>
          <div class="flex justify-between border-b border-slate-800/70 pb-2">
            <span class="text-slate-500">启用占比</span>
            <span class="text-cyan-300">{{ stats.total ? Math.round(stats.enabled / stats.total * 100) : 0 }}%</span>
          </div>
          <div class="flex justify-between border-b border-slate-800/70 pb-2">
            <span class="text-slate-500">时间桶数</span>
            <span class="text-slate-300">{{ hours.length || '-' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-slate-500">日志条数</span>
            <span class="text-slate-300">{{ logs.length }}</span>
          </div>
        </div>
      </Panel>
    </section>

    <section>
      <div class="mb-2 flex items-center justify-between gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">访问日志</h2>
        <button class="btn" @click="refreshLogs">刷新</button>
      </div>
      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>镜像</th>
              <th>方法</th>
              <th>路径</th>
              <th>状态</th>
              <th class="text-right">大小</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.id">
              <td class="font-mono text-xs text-slate-500">{{ log.created_at?.slice(11, 19) }}</td>
              <td><span class="tag">{{ log.mirror }}</span></td>
              <td class="font-mono text-xs text-slate-400">{{ log.method }}</td>
              <td class="max-w-[420px] truncate font-mono text-xs text-slate-300">{{ log.path }}</td>
              <td :class="log.status >= 200 && log.status < 300 ? 'text-emerald-300' : 'text-red-300'">{{ log.status }}</td>
              <td class="text-right text-xs text-slate-400">{{ formatBytes(log.bytes_out) }}</td>
            </tr>
          </tbody>
        </table>
        <EmptyState v-if="!logs.length" message="暂无访问记录。" />
      </div>
    </section>
  </div>
</template>
