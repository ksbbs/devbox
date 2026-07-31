<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getPublicConfig } from '../api/client'
import CodeBlock from '../components/CodeBlock.vue'

const publicUrl = ref('')

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
    title: 'GitHub 克隆',
    path: '/gh/user/repo',
    desc: '克隆 GitHub 仓库',
    cmd: `git clone ${publicUrl.value}/gh/user/repo`,
  },
  {
    title: 'GitLab 克隆',
    path: '/gl/user/repo',
    desc: '克隆 GitLab 仓库',
    cmd: `git clone ${publicUrl.value}/gl/user/repo`,
  },
  {
    title: '压缩包下载',
    path: '/gh/user/repo/archive/main.zip',
    desc: '下载仓库压缩包',
    cmd: `curl ${publicUrl.value}/gh/user/repo/archive/main.zip -o main.zip`,
  },
  {
    title: '原始文件',
    path: '/gh/user/repo/raw/branch/file.txt',
    desc: '获取原始文件内容',
    cmd: `curl ${publicUrl.value}/gh/user/repo/raw/branch/file.txt`,
  },
])
</script>

<template>
  <div>
    <section class="page-header">
      <span class="page-kicker">代理</span>
      <div class="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="page-title">Git 代理</h1>
          <p class="page-subtitle">GitHub / GitLab 克隆、压缩包、原始文件请求加速命令。</p>
        </div>
        <code class="code-line w-fit max-w-full truncate">{{ publicUrl || '解析服务地址中...' }}</code>
      </div>
    </section>

    <div class="grid grid-cols-1 gap-3">
      <article v-for="item in commands" :key="item.title" class="glass glass-hover min-w-0 p-4">
        <div class="grid grid-cols-1 gap-3 lg:grid-cols-[180px_minmax(0,1fr)] lg:items-center">
          <div class="min-w-0">
            <h2 class="text-sm font-semibold text-slate-100">{{ item.title }}</h2>
            <p class="mt-1 text-xs text-slate-500">{{ item.desc }}</p>
            <code class="mt-2 block text-xs text-slate-600">{{ item.path }}</code>
          </div>
          <CodeBlock :code="item.cmd" />
        </div>
      </article>
    </div>
  </div>
</template>
