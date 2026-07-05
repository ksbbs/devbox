<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getStatus, getTraffic, getPublicConfig, getRecentLogs } from '../api/client'
import StatusCard from '../components/StatusCard.vue'

const mirrors = ref<any[]>([])
const loading = ref(true)
const errorMsg = ref('')
const traffic = ref<any[]>([])
const hourlyTraffic = ref<any[]>([])
const logs = ref<any[]>([])
const publicUrl = ref('')
const copiedGuide = ref<string | null>(null)
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
    title: 'GitHub clone',
    desc: '把 github.com/owner/repo 替换为 /gh/owner/repo。',
    cmd: `git clone ${usageBaseUrl.value}/gh/user/repo`,
  },
  {
    title: 'GitLab clone',
    desc: '把 gitlab.com/group/repo 替换为 /gl/group/repo。',
    cmd: `git clone ${usageBaseUrl.value}/gl/group/repo`,
  },
  {
    title: 'Archive',
    desc: '下载仓库压缩包。',
    cmd: `curl ${usageBaseUrl.value}/gh/user/repo/archive/main.zip -o main.zip`,
  },
  {
    title: 'Raw file',
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

const trendRows = computed(() => {
  const metric = (item: any) => chartMode.value === 'bandwidth'
    ? Number(item.bytes_out || item.bytesOut || 0)
    : Number(item.requests || 0)

  if (!hourlyTraffic.value.length) {
    return traffic.value
      .map(item => ({
        mirror: item.mirror || 'unknown',
        total: metric(item),
        requests: Number(item.requests || 0),
        bytesOut: Number(item.bytes_out || item.bytesOut || 0),
        values: [metric(item)],
      }))
      .sort((a, b) => b.total - a.total)
      .slice(0, 8)
  }

  const groups = new Map<string, { mirror: string; byHour: Map<string, number>; requests: number; bytesOut: number }>()
  for (const item of hourlyTraffic.value) {
    const mirror = item.mirror || 'unknown'
    const group = groups.get(mirror) || { mirror, byHour: new Map(), requests: 0, bytesOut: 0 }
    group.byHour.set(item.hour, (group.byHour.get(item.hour) || 0) + metric(item))
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

onMounted(async () => {
  const [statusRes, trafficRes, hourlyRes, logsRes, configRes] = await Promise.allSettled([
    getStatus(),
    getTraffic(),
    getTraffic(undefined, undefined, 'hourly'),
    getRecentLogs(50),
    getPublicConfig(),
  ])

  if (statusRes.status === 'fulfilled') mirrors.value = Array.isArray(statusRes.value) ? statusRes.value : []
  else errorMsg.value = 'Failed to load mirror status'
  if (trafficRes.status === 'fulfilled') traffic.value = Array.isArray(trafficRes.value) ? trafficRes.value : []
  if (hourlyRes.status === 'fulfilled') hourlyTraffic.value = Array.isArray(hourlyRes.value) ? hourlyRes.value : []
  if (logsRes.status === 'fulfilled') logs.value = Array.isArray(logsRes.value) ? logsRes.value : []
  if (configRes.status === 'fulfilled') publicUrl.value = configRes.value.publicUrl || ''
  loading.value = false
})

async function switchChartMode() {
  chartMode.value = chartMode.value === 'requests' ? 'bandwidth' : 'requests'
}

async function switchGranularity(level: 'hourly' | 'daily' | 'weekly') {
  chartGranularity.value = level
  const data = await getTraffic(undefined, undefined, level)
  hourlyTraffic.value = Array.isArray(data) ? data : []
}

async function refreshLogs() {
  logs.value = await getRecentLogs(50)
}

function copyGuide(id: string, cmd: string) {
  navigator.clipboard.writeText(cmd)
  copiedGuide.value = id
  setTimeout(() => copiedGuide.value = null, 1500)
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

</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">runtime</span>
      <div class="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="page-title">Dashboard</h1>
          <p class="page-subtitle">镜像加速服务状态、流量趋势与访问日志。</p>
        </div>
        <code v-if="publicUrl" class="code-line w-fit max-w-full truncate">{{ publicUrl }}</code>
      </div>
    </section>

    <div v-if="errorMsg" class="mb-4 border border-red-500/40 bg-red-950/30 px-3 py-2 text-sm text-red-300">
      {{ errorMsg }}
    </div>

    <section class="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
      <div class="panel-pad">
        <div class="text-xs uppercase tracking-[0.18em] text-slate-500">mirrors</div>
        <div class="mt-2 text-3xl font-semibold text-slate-100">{{ stats.total }}</div>
      </div>
      <div class="panel-pad">
        <div class="text-xs uppercase tracking-[0.18em] text-slate-500">healthy</div>
        <div class="mt-2 text-3xl font-semibold text-emerald-300">{{ stats.healthy }}</div>
      </div>
      <div class="panel-pad">
        <div class="text-xs uppercase tracking-[0.18em] text-slate-500">enabled</div>
        <div class="mt-2 text-3xl font-semibold text-cyan-300">{{ stats.enabled }}</div>
      </div>
      <div class="panel-pad">
        <div class="text-xs uppercase tracking-[0.18em] text-slate-500">requests</div>
        <div class="mt-2 text-3xl font-semibold text-slate-100">{{ stats.requests.toLocaleString() }}</div>
      </div>
    </section>

    <section class="mb-5 grid gap-5 xl:grid-cols-2">
      <div class="panel-pad">
        <div class="mb-4 flex items-center justify-between gap-3 border-b border-slate-900 pb-3">
          <div>
            <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">mirror acceleration</h2>
            <p class="mt-1 text-xs text-slate-600">复制命令后把示例包名或镜像名替换成你的目标。</p>
          </div>
          <span class="tag tag-ok">packages</span>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <article v-for="item in mirrorUsage" :key="item.title" class="border border-slate-900 bg-black/20 p-3">
            <div class="mb-2 flex items-start justify-between gap-2">
              <div>
                <h3 class="text-sm font-semibold text-slate-100">{{ item.title }}</h3>
                <p class="mt-1 text-xs text-slate-600">{{ item.desc }}</p>
              </div>
              <button class="btn" @click="copyGuide(`mirror-${item.title}`, item.cmd)">
                {{ copiedGuide === `mirror-${item.title}` ? 'copied' : 'copy' }}
              </button>
            </div>
            <code class="code-line block overflow-x-auto whitespace-nowrap">{{ item.cmd }}</code>
          </article>
        </div>
      </div>

      <div class="panel-pad">
        <div class="mb-4 flex items-center justify-between gap-3 border-b border-slate-900 pb-3">
          <div>
            <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">git acceleration</h2>
            <p class="mt-1 text-xs text-slate-600">GitHub 走 /gh/，GitLab 走 /gl/，支持 clone、archive 和 raw。</p>
          </div>
          <span class="tag tag-ok">git</span>
        </div>
        <div class="grid gap-3">
          <article v-for="item in gitUsage" :key="item.title" class="border border-slate-900 bg-black/20 p-3">
            <div class="mb-2 flex items-start justify-between gap-2">
              <div>
                <h3 class="text-sm font-semibold text-slate-100">{{ item.title }}</h3>
                <p class="mt-1 text-xs text-slate-600">{{ item.desc }}</p>
              </div>
              <button class="btn" @click="copyGuide(`git-${item.title}`, item.cmd)">
                {{ copiedGuide === `git-${item.title}` ? 'copied' : 'copy' }}
              </button>
            </div>
            <code class="code-line block overflow-x-auto whitespace-nowrap">{{ item.cmd }}</code>
          </article>
        </div>
      </div>
    </section>

    <section class="mb-5">
      <div class="mb-3 flex items-center justify-between gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">mirror status</h2>
        <span v-if="loading" class="text-xs text-cyan-400">loading...</span>
      </div>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <StatusCard v-for="m in mirrors" :key="m.name" :mirror="m" />
      </div>
      <div v-if="!loading && !mirrors.length" class="p-6 text-center text-sm text-slate-500">暂无镜像状态。</div>
    </section>

    <section class="mb-5 grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div>
        <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
          <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">traffic trend</h2>
          <div class="flex gap-2">
            <select class="select w-auto text-xs" :value="chartGranularity" @change="switchGranularity(($event.target as HTMLSelectElement).value as any)">
              <option value="hourly">hourly</option>
              <option value="daily">daily</option>
              <option value="weekly">weekly</option>
            </select>
            <button class="btn" @click="switchChartMode">
              {{ chartMode === 'requests' ? 'metric: requests' : 'metric: bandwidth' }}
            </button>
          </div>
        </div>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>mirror</th>
                <th>{{ chartMode === 'requests' ? 'requests' : 'bytes out' }}</th>
                <th>sparkline</th>
                <th>total requests</th>
                <th>total out</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in trendRows" :key="row.mirror">
                <td class="font-semibold text-slate-100">{{ row.mirror }}</td>
                <td class="text-cyan-300">{{ formatMetric(row.total) }}</td>
                <td class="w-44">
                  <svg viewBox="0 0 100 32" class="h-8 w-36 text-cyan-400" preserveAspectRatio="none">
                    <polyline :points="sparklinePoints(row.values)" fill="none" stroke="currentColor" stroke-width="2" vector-effect="non-scaling-stroke" />
                  </svg>
                </td>
                <td class="text-slate-400">{{ row.requests.toLocaleString() }}</td>
                <td class="text-slate-400">{{ formatBytes(row.bytesOut) }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="!trendRows.length" class="p-6 text-center text-sm text-slate-500">暂无流量数据。</div>
        </div>
      </div>

      <div class="panel-pad">
        <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">health summary</h2>
        <div class="mt-4 space-y-3 text-sm">
          <div class="flex justify-between border-b border-slate-900 pb-2">
            <span class="text-slate-500">healthy ratio</span>
            <span class="text-emerald-300">{{ stats.total ? Math.round(stats.healthy / stats.total * 100) : 0 }}%</span>
          </div>
          <div class="flex justify-between border-b border-slate-900 pb-2">
            <span class="text-slate-500">enabled ratio</span>
            <span class="text-cyan-300">{{ stats.total ? Math.round(stats.enabled / stats.total * 100) : 0 }}%</span>
          </div>
          <div class="flex justify-between border-b border-slate-900 pb-2">
            <span class="text-slate-500">hour buckets</span>
            <span class="text-slate-300">{{ hours.length || '-' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-slate-500">log rows</span>
            <span class="text-slate-300">{{ logs.length }}</span>
          </div>
        </div>
      </div>
    </section>

    <section>
      <div class="mb-2 flex items-center justify-between gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">access logs</h2>
        <button class="btn" @click="refreshLogs">refresh</button>
      </div>
      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>time</th>
              <th>mirror</th>
              <th>method</th>
              <th>path</th>
              <th>status</th>
              <th class="text-right">size</th>
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
        <div v-if="!logs.length" class="p-6 text-center text-sm text-slate-500">暂无访问记录。</div>
      </div>
    </section>
  </div>
</template>
