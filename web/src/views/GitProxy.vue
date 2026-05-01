<script setup lang="ts">
</script>

<template>
  <div>
    <h2 class="text-2xl font-semibold mb-6">Git Proxy</h2>
    <div class="bg-slate-800 border border-slate-700 rounded-lg p-6 space-y-4">
      <div>
        <h3 class="text-lg font-medium text-white">GitHub Clone</h3>
        <p class="text-slate-400 mt-1">Use the following URL pattern to clone GitHub repos:</p>
        <div class="mt-2 bg-slate-700 rounded px-4 py-3 text-emerald-400 font-mono text-sm">
          git clone http://your-vps:8080/gh/user/repo
        </div>
      </div>
      <div>
        <h3 class="text-lg font-medium text-white">GitLab Clone</h3>
        <p class="text-slate-400 mt-1">Use the following URL pattern to clone GitLab repos:</p>
        <div class="mt-2 bg-slate-700 rounded px-4 py-3 text-emerald-400 font-mono text-sm">
          git clone http://your-vps:8080/gl/user/repo
        </div>
      </div>
      <div>
        <h3 class="text-lg font-medium text-white">Archive Download</h3>
        <p class="text-slate-400 mt-1">Download GitHub archive files:</p>
        <div class="mt-2 bg-slate-700 rounded px-4 py-3 text-emerald-400 font-mono text-sm">
          http://your-vps:8080/gh/user/repo/archive/main.zip
        </div>
      </div>
      <div>
        <h3 class="text-lg font-medium text-white">Raw File</h3>
        <p class="text-slate-400 mt-1">Access raw content from GitHub:</p>
        <div class="mt-2 bg-slate-700 rounded px-4 py-3 text-emerald-400 font-mono text-sm">
          http://your-vps:8080/gh/user/repo/raw/branch/file.txt
        </div>
      </div>
    </div>
  </div>
</template>