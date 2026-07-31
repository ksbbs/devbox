<script setup lang="ts">
import { useRouter } from 'vue-router'
import { isLoggedIn, logout } from './api/client'
import Logo from './components/Logo.vue'

const router = useRouter()

function handleLogout() {
  logout()
  router.push('/login')
}

const navItems = [
  { to: '/', label: '仪表盘', code: 'dash' },
  { to: '/mirrors', label: '镜像源', code: 'mir' },
  { to: '/gitproxy', label: 'Git 代理', code: 'git' },
  { to: '/search', label: '搜索', code: 'find' },
  { to: '/releases', label: '发行版', code: 'rel' },
  { to: '/settings', label: '设置', code: 'cfg' },
]
</script>

<template>
  <div class="min-h-screen text-slate-200">
    <header class="sticky top-0 z-40 border-b border-slate-800/80 bg-slate-950/70 backdrop-blur-md">
      <div class="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
        <div class="flex items-center justify-between gap-4">
          <router-link to="/" class="group flex items-center gap-3">
            <Logo class="transition-[filter] duration-300 group-hover:drop-shadow-[0_0_10px_rgba(124,58,237,0.7)]" />
            <span class="text-sm font-semibold tracking-wide text-slate-100">devbox</span>
            <span class="hidden text-sm text-slate-500 sm:inline">镜像代理控制台</span>
          </router-link>
          <button v-if="isLoggedIn()" @click="handleLogout" class="btn lg:hidden">退出</button>
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
          <button v-if="isLoggedIn()" @click="handleLogout" class="btn ml-2 hidden lg:inline-flex">退出</button>
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
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border: 1px solid transparent;
  border-radius: 0.375rem;
  padding: 0.45rem 0.75rem;
  color: #94a3b8;
  font-size: 0.78rem;
  line-height: 1rem;
  text-decoration: none;
  transition: border-color 150ms ease, color 150ms ease, background-color 150ms ease, box-shadow 150ms ease;
  white-space: nowrap;
}

.nav-link:hover {
  border-color: rgba(103, 232, 249, 0.28);
  color: #cbd5e1;
  background: linear-gradient(180deg, rgba(148, 163, 184, 0.09), rgba(2, 6, 23, 0.35));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

.nav-link.active {
  border-color: rgba(189, 139, 255, 0.3);
  color: var(--brand-light);
  background: linear-gradient(180deg, color-mix(in srgb, var(--brand-1) 14%, transparent), rgba(2, 6, 23, 0.3));
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--brand-3) 12%, transparent), inset 0 -2px 0 0 color-mix(in srgb, var(--brand-1) 35%, transparent);
}

.nav-link.active::after {
  content: "";
  position: absolute;
  left: 0.5rem;
  right: 0.5rem;
  bottom: 0.2rem;
  height: 2px;
  border-radius: 1px;
  background: linear-gradient(90deg, var(--brand-1), var(--accent));
  box-shadow: 0 0 10px color-mix(in srgb, var(--brand-1) 80%, transparent);
}

.nav-link.active .text-\[10px\] {
  color: var(--brand-3);
}

.page-enter-active,
.page-leave-active {
  transition: opacity 140ms ease, transform 140ms ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(4px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-2px);
}
</style>
