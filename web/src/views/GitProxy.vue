<script setup lang="ts">
import { ref } from 'vue'

const commands = [
  {
    title: 'GitHub Clone',
    desc: '克隆 GitHub 仓库',
    cmd: 'git clone http://your-vps:8080/gh/user/repo',
    icon: '◆'
  },
  {
    title: 'GitLab Clone',
    desc: '克隆 GitLab 仓库',
    cmd: 'git clone http://your-vps:8080/gl/user/repo',
    icon: '◇'
  },
  {
    title: 'Archive 下载',
    desc: '下载仓库压缩包',
    cmd: 'curl http://your-vps:8080/gh/user/repo/archive/main.zip -o main.zip',
    icon: '▣'
  },
  {
    title: 'Raw 文件',
    desc: '获取原始文件内容',
    cmd: 'curl http://your-vps:8080/gh/user/repo/raw/branch/file.txt',
    icon: '▤'
  }
]

const copied = ref<number | null>(null)

function copyCmd(index: number, cmd: string) {
  navigator.clipboard.writeText(cmd)
  copied.value = index
  setTimeout(() => copied.value = null, 2000)
}
</script>

<template>
  <div>
    <div class="mb-8">
      <h1 class="text-4xl font-bold mb-2">
        <span class="bg-gradient-to-r from-emerald-400 via-teal-400 to-cyan-400 bg-clip-text text-transparent">
          Git Proxy
        </span>
      </h1>
      <p class="text-slate-400">加速 GitHub / GitLab 代码克隆和下载</p>
    </div>

    <div class="grid gap-4">
      <div v-for="(item, i) in commands" :key="i"
        class="git-card group relative p-6 rounded-2xl overflow-hidden">

        <!-- 背景装饰 -->
        <div class="absolute -right-10 -top-10 w-40 h-40 rounded-full opacity-0 group-hover:opacity-20 transition-opacity duration-500"
          :class="[
            'bg-gradient-to-br',
            i === 0 ? 'from-sky-400 to-violet-500' :
            i === 1 ? 'from-violet-400 to-fuchsia-500' :
            i === 2 ? 'from-fuchsia-400 to-pink-500' :
            'from-pink-400 to-rose-500'
          ]"></div>

        <div class="relative flex items-start gap-5">
          <!-- 图标 -->
          <div class="w-14 h-14 rounded-2xl flex items-center justify-center text-2xl shrink-0 transition-all duration-300"
            :class="[
              'bg-gradient-to-br shadow-lg',
              i === 0 ? 'from-sky-500 to-violet-500 shadow-sky-500/20' :
              i === 1 ? 'from-violet-500 to-fuchsia-500 shadow-violet-500/20' :
              i === 2 ? 'from-fuchsia-500 to-pink-500 shadow-fuchsia-500/20' :
              'from-pink-500 to-rose-500 shadow-pink-500/20',
              'text-white'
            ]">
            {{ item.icon }}
          </div>

          <!-- 内容 -->
          <div class="flex-1 min-w-0">
            <h3 class="text-lg font-semibold text-white mb-1">{{ item.title }}</h3>
            <p class="text-slate-400 text-sm mb-3">{{ item.desc }}</p>

            <!-- 命令框 -->
            <div class="relative group/cmd">
              <div class="absolute -inset-0.5 rounded-xl opacity-30 blur transition-opacity"
                :class="[
                  'bg-gradient-to-r',
                  i === 0 ? 'from-sky-400 to-violet-500' :
                  i === 1 ? 'from-violet-400 to-fuchsia-500' :
                  i === 2 ? 'from-fuchsia-400 to-pink-500' :
                  'from-pink-400 to-rose-500'
                ]"></div>
              <div class="relative flex items-center gap-3 bg-slate-900/90 rounded-xl px-4 py-3 border border-white/5">
                <code class="flex-1 font-mono text-sm text-emerald-400 truncate">{{ item.cmd }}</code>
                <button @click="copyCmd(i, item.cmd)"
                  class="shrink-0 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
                  :class="copied === i
                    ? 'bg-emerald-500/20 text-emerald-400'
                    : 'bg-white/5 text-slate-400 hover:bg-white/10 hover:text-white'">
                  {{ copied === i ? '已复制!' : '复制' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style>
.git-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.04) 0%, rgba(255, 255, 255, 0.01) 100%);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(10px);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.git-card:hover {
  border-color: rgba(16, 185, 129, 0.3);
  transform: translateY(-2px);
  box-shadow:
    0 20px 40px rgba(0, 0, 0, 0.3),
    0 0 60px rgba(16, 185, 129, 0.1);
}
</style>