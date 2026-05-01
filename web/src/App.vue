<script setup lang="ts">
</script>

<template>
  <div class="min-h-screen bg-slate-900 text-slate-200">
    <nav class="bg-slate-800/80 border-b border-slate-700/50 px-6 py-3 flex items-center gap-2 backdrop-blur-sm sticky top-0 z-50">
      <h1 class="text-xl font-bold text-white mr-4">DevBox</h1>
      <router-link to="/" class="nav-link" active-class="active">Dashboard</router-link>
      <router-link to="/mirrors" class="nav-link" active-class="active">Mirrors</router-link>
      <router-link to="/gitproxy" class="nav-link" active-class="active">Git Proxy</router-link>
      <router-link to="/settings" class="nav-link" active-class="active">Settings</router-link>
    </nav>
    <main class="p-6 max-w-5xl mx-auto">
      <router-view v-slot="{ Component }">
        <transition name="fade-slide" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
  </div>
</template>

<style>
.nav-link {
  color: #94a3b8;
  padding: 0.375rem 0.875rem;
  border-radius: 0.5rem;
  transition: all 0.2s ease;
  font-weight: 500;
  text-decoration: none;
}
.nav-link:hover {
  color: #e2e8f0;
  background: rgba(148,163,184,0.1);
}
.nav-link.active {
  color: #38bdf8;
  background: rgba(56,189,248,0.12);
  box-shadow: 0 0 12px rgba(56,189,248,0.15);
}

.fade-slide-enter-active {
  transition: all 0.25s ease-out;
}
.fade-slide-leave-active {
  transition: all 0.2s ease-in;
}
.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>