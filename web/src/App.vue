<script setup lang="ts">
import { isLoggedIn, logout } from './api/client'

const navItems = [
  { to: '/', label: 'Dashboard', code: 'dash' },
  { to: '/mirrors', label: 'Mirrors', code: 'mir' },
  { to: '/gitproxy', label: 'Git Proxy', code: 'git' },
  { to: '/search', label: 'Search', code: 'find' },
  { to: '/settings', label: 'Settings', code: 'cfg' },
]
</script>

<template>
  <div class="min-h-screen text-slate-200">
    <header class="sticky top-0 z-40 border-b border-slate-800 bg-slate-950/95 backdrop-blur-sm">
      <div class="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
        <div class="flex items-center justify-between gap-4">
          <router-link to="/" class="flex items-center gap-3 text-slate-100">
            <span class="border border-cyan-500/50 bg-cyan-950/30 px-2 py-1 text-xs font-semibold text-cyan-300">devbox</span>
            <span class="text-sm text-slate-500">mirror proxy console</span>
          </router-link>
          <button v-if="isLoggedIn()" @click="logout" class="btn lg:hidden">logout</button>
        </div>

        <nav class="flex items-center gap-1 overflow-x-auto pb-1 lg:pb-0">
          <router-link
            v-for="item in navItems"
            :key="item.to"
            :to="item.to"
            class="nav-link"
            active-class="active"
          >
            <span class="text-[10px] uppercase tracking-widest text-slate-600">{{ item.code }}</span>
            <span>{{ item.label }}</span>
          </router-link>
          <button v-if="isLoggedIn()" @click="logout" class="btn ml-2 hidden lg:inline-flex">logout</button>
        </nav>
      </div>
    </header>

    <main class="mx-auto max-w-7xl px-4 py-6">
      <router-view v-slot="{ Component }">
        <transition name="page" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
  </div>
</template>

<style>
.nav-link {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border: 1px solid transparent;
  padding: 0.45rem 0.75rem;
  color: #94a3b8;
  font-size: 0.78rem;
  line-height: 1rem;
  text-decoration: none;
  transition: border-color 150ms ease, color 150ms ease, background-color 150ms ease;
  white-space: nowrap;
}

.nav-link:hover {
  border-color: rgba(34, 211, 238, 0.35);
  color: #cbd5e1;
  background: rgba(15, 23, 42, 0.8);
}

.nav-link.active {
  border-color: rgba(34, 211, 238, 0.65);
  color: #67e8f9;
  background: rgba(8, 47, 73, 0.35);
}

.page-enter-active,
.page-leave-active {
  transition: opacity 120ms ease;
}

.page-enter-from,
.page-leave-to {
  opacity: 0;
}
</style>
