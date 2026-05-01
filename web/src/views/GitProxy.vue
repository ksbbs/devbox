<script setup lang="ts">
</script>

<template>
  <div>
    <h2 class="text-2xl font-semibold mb-6 text-white">Git Proxy</h2>
    <div class="space-y-3">
      <div class="card-hover bg-slate-800 border border-slate-700/60 rounded-xl p-6 transition-all duration-300 hover:border-sky-500/30">
        <h3 class="text-lg font-semibold text-white">GitHub Clone</h3>
        <p class="text-slate-400 mt-1">使用以下 URL 格式克隆 GitHub 仓库：</p>
        <div class="mt-2 bg-slate-900/60 rounded-lg px-4 py-3 text-emerald-400 font-mono text-sm border border-slate-700/30">
          git clone http://your-vps:8080/gh/user/repo
        </div>
      </div>
      <div class="card-hover bg-slate-800 border border-slate-700/60 rounded-xl p-6 transition-all duration-300 hover:border-sky-500/30">
        <h3 class="text-lg font-semibold text-white">GitLab Clone</h3>
        <p class="text-slate-400 mt-1">使用以下 URL 格式克隆 GitLab 仓库：</p>
        <div class="mt-2 bg-slate-900/60 rounded-lg px-4 py-3 text-emerald-400 font-mono text-sm border border-slate-700/30">
          git clone http://your-vps:8080/gl/user/repo
        </div>
      </div>
      <div class="card-hover bg-slate-800 border border-slate-700/60 rounded-xl p-6 transition-all duration-300 hover:border-sky-500/30">
        <h3 class="text-lg font-semibold text-white">Archive Download</h3>
        <p class="text-slate-400 mt-1">下载 GitHub 仓库的 archive 文件：</p>
        <div class="mt-2 bg-slate-900/60 rounded-lg px-4 py-3 text-emerald-400 font-mono text-sm border border-slate-700/30">
          http://your-vps:8080/gh/user/repo/archive/main.zip
        </div>
      </div>
      <div class="card-hover bg-slate-800 border border-slate-700/60 rounded-xl p-6 transition-all duration-300 hover:border-sky-500/30">
        <h3 class="text-lg font-semibold text-white">Raw File</h3>
        <p class="text-slate-400 mt-1">访问 GitHub 上的 raw 文件内容：</p>
        <div class="mt-2 bg-slate-900/60 rounded-lg px-4 py-3 text-emerald-400 font-mono text-sm border border-slate-700/30">
          http://your-vps:8080/gh/user/repo/raw/branch/file.txt
        </div>
      </div>
    </div>
  </div>
</template>