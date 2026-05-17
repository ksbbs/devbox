<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getPublicConfig } from '../api/client'

const publicUrl = ref('')
const copied = ref<number | null>(null)

onMounted(async () => {
  try {
    const config = await getPublicConfig()
    publicUrl.value = config.publicUrl || window.location.origin
  } catch {
    publicUrl.value = window.location.origin
  }
})

const commands = computed(() => [
  {
    title: 'GitHub Clone',
    path: '/gh/user/repo',
    desc: '克隆 GitHub 仓库',
    cmd: `git clone ${publicUrl.value}/gh/user/repo`,
  },
  {
    title: 'GitLab Clone',
    path: '/gl/user/repo',
    desc: '克隆 GitLab 仓库',
    cmd: `git clone ${publicUrl.value}/gl/user/repo`,
  },
  {
    title: 'Archive',
    path: '/gh/user/repo/archive/main.zip',
    desc: '下载仓库压缩包',
    cmd: `curl ${publicUrl.value}/gh/user/repo/archive/main.zip -o main.zip`,
  },
  {
    title: 'Raw File',
    path: '/gh/user/repo/raw/branch/file.txt',
    desc: '获取原始文件内容',
    cmd: `curl ${publicUrl.value}/gh/user/repo/raw/branch/file.txt`,
  },
])

function copyCmd(index: number, cmd: string) {
  navigator.clipboard.writeText(cmd)
  copied.value = index
  setTimeout(() => copied.value = null, 1500)
}
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">proxy</span>
      <div class="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="page-title">Git Proxy</h1>
          <p class="page-subtitle">GitHub / GitLab clone、archive、raw 请求加速命令。</p>
        </div>
        <code class="code-line w-fit max-w-full truncate">{{ publicUrl || 'resolving origin...' }}</code>
      </div>
    </section>

    <div class="grid gap-3">
      <article v-for="(item, i) in commands" :key="item.title" class="panel-pad">
        <div class="grid gap-3 lg:grid-cols-[180px_minmax(0,1fr)_auto] lg:items-center">
          <div>
            <h2 class="text-sm font-semibold text-slate-100">{{ item.title }}</h2>
            <p class="mt-1 text-xs text-slate-500">{{ item.desc }}</p>
            <code class="mt-2 block text-xs text-slate-600">{{ item.path }}</code>
          </div>
          <code class="code-line block overflow-x-auto whitespace-nowrap">{{ item.cmd }}</code>
          <button class="btn btn-primary" @click="copyCmd(i, item.cmd)">
            {{ copied === i ? 'copied' : 'copy' }}
          </button>
        </div>
      </article>
    </div>
  </div>
</template>
