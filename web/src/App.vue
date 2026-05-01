<script setup lang="ts">
import { ref } from 'vue'
const currentPath = ref(window.location.pathname)
window.addEventListener('popstate', () => {
  currentPath.value = window.location.pathname
})
</script>

<template>
  <div class="min-h-screen bg-slate-900 text-slate-200">
    <nav class="bg-slate-800 border-b border-slate-700 px-6 py-3 flex items-center gap-6">
      <h1 class="text-xl font-bold text-white">DevBox</h1>
      <a href="/" class="nav-link" :class="{ active: currentPath === '/' }">Dashboard</a>
      <a href="/mirrors" class="nav-link" :class="{ active: currentPath === '/mirrors' }">Mirrors</a>
      <a href="/gitproxy" class="nav-link" :class="{ active: currentPath === '/gitproxy' }">Git Proxy</a>
      <a href="/settings" class="nav-link" :class="{ active: currentPath === '/settings' }">Settings</a>
    </nav>
    <main class="p-6">
      <router-view />
    </main>
  </div>
</template>

<style>
.nav-link {
  color: #94a3b8;
  padding: 0.25rem 0.75rem;
  border-radius: 0.375rem;
  transition: color 0.2s;
}
.nav-link:hover { color: #e2e8f0; }
.nav-link.active { color: #38bdf8; background: rgba(56,189,248,0.1); }
</style>