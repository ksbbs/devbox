<script setup lang="ts">
import { useRouter } from 'vue-router'
import { isLoggedIn, logout } from './api/client'

const router = useRouter()

function handleLogout() {
  logout()
  router.push('/login')
}

const navItems = [
  { to: '/', label: 'Dashboard', code: 'dash' },
  { to: '/mirrors', label: 'Mirrors', code: 'mir' },
  { to: '/gitproxy', label: 'Git Proxy', code: 'git' },
  { to: '/search', label: 'Search', code: 'find' },
  { to: '/releases', label: 'Releases', code: 'rel' },
  { to: '/settings', label: 'Settings', code: 'cfg' },
]
</script>

<template>
  <div class="min-h-screen text-slate-200">
    <header class="sticky top-0 z-40 border-b border-slate-800 bg-slate-950/95 backdrop-blur-sm">
      <div class="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
        <div class="flex items-center justify-between gap-4">
          <router-link to="/" class="flex items-center gap-3 text-slate-100">
            <svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
              <defs>
                <linearGradient id="dbTop" x1="0" y1="0" x2="1" y2="1">
                  <stop offset="0%" stop-color="#f2e4ff"/>
                  <stop offset="100%" stop-color="#bd8bff"/>
                </linearGradient>
                <linearGradient id="dbLeft" x1="0" y1="0" x2="1" y2="0">
                  <stop offset="0%" stop-color="#7c3aed"/>
                  <stop offset="100%" stop-color="#9455f5"/>
                </linearGradient>
                <linearGradient id="dbRight" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="#6528d4"/>
                  <stop offset="100%" stop-color="#43158f"/>
                </linearGradient>
              </defs>
              <polygon points="12,2 19.75,6.75 12,11.5 4.25,6.75" fill="url(#dbTop)"/>
              <polygon points="12,11.5 19.75,6.75 19.75,16.25 12,21" fill="url(#dbRight)"/>
              <polygon points="12,11.5 12,21 4.25,16.25 4.25,6.75" fill="url(#dbLeft)"/>
              <polygon points="12,4 16.5,6.4 12,8.8 7.5,6.4" fill="#ffffff" opacity="0.16"/>
              <g stroke="#67e8f9" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="13.25,13.75 15.25,15.25 13.25,16.75"/>
                <line x1="16.25" y1="16.75" x2="17.75" y2="16.75"/>
              </g>
            </svg>
            <span class="text-sm font-semibold tracking-wide text-slate-100">devbox</span>
            <span class="hidden text-sm text-slate-500 sm:inline">mirror proxy console</span>
          </router-link>
          <button v-if="isLoggedIn()" @click="handleLogout" class="btn lg:hidden">logout</button>
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
          <button v-if="isLoggedIn()" @click="handleLogout" class="btn ml-2 hidden lg:inline-flex">logout</button>
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
