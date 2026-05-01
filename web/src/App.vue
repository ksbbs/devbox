<script setup lang="ts">
import { isLoggedIn, logout } from './api/client'
</script>

<template>
  <div class="min-h-screen bg-[#0a0a0f] text-slate-200 overflow-x-hidden">
    <!-- 动态背景 -->
    <div class="fixed inset-0 pointer-events-none">
      <!-- 渐变网格 -->
      <div class="absolute inset-0 bg-[linear-gradient(rgba(56,189,248,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(56,189,248,0.03)_1px,transparent_1px)] bg-[size:60px_60px]"></div>
      <!-- 动态光晕 -->
      <div class="aurora aurora-1"></div>
      <div class="aurora aurora-2"></div>
      <div class="aurora aurora-3"></div>
      <!-- 粒子效果 -->
      <div class="particles">
        <span v-for="n in 20" :key="n" :style="{ '--i': n }"></span>
      </div>
    </div>

    <!-- 导航栏 -->
    <nav class="fixed top-0 left-0 right-0 z-50">
      <div class="mx-4 mt-4">
        <div class="glass-nav px-6 py-3 flex items-center gap-2 rounded-2xl border border-white/10">
          <!-- Logo -->
          <div class="flex items-center gap-3 mr-6">
            <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-sky-400 via-violet-500 to-fuchsia-500 flex items-center justify-center shadow-lg shadow-sky-500/20">
              <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"></path>
              </svg>
            </div>
            <span class="text-xl font-bold bg-gradient-to-r from-sky-400 via-violet-400 to-fuchsia-400 bg-clip-text text-transparent">DevBox</span>
          </div>

          <router-link to="/" class="nav-link" active-class="active">
            <span class="nav-icon">◈</span>
            Dashboard
          </router-link>
          <router-link to="/mirrors" class="nav-link" active-class="active">
            <span class="nav-icon">◎</span>
            Mirrors
          </router-link>
          <router-link to="/gitproxy" class="nav-link" active-class="active">
            <span class="nav-icon">◉</span>
            Git Proxy
          </router-link>
          <router-link to="/settings" class="nav-link" active-class="active">
            <span class="nav-icon">◐</span>
            Settings
          </router-link>
          <button v-if="isLoggedIn()" @click="logout"
            class="ml-auto px-3 py-1.5 rounded-xl text-xs font-medium text-slate-400 hover:text-slate-200 hover:bg-white/10 transition-all">
            登出
          </button>
        </div>
      </div>
    </nav>

    <!-- 主内容 -->
    <main class="relative z-10 pt-28 pb-12 px-6 max-w-6xl mx-auto">
      <router-view v-slot="{ Component }">
        <transition name="page" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
  </div>
</template>

<style>
/* 玻璃拟态导航 */
.glass-nav {
  background: rgba(15, 15, 25, 0.7);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  box-shadow:
    0 4px 30px rgba(0, 0, 0, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
}

/* 极光背景 */
.aurora {
  position: absolute;
  width: 600px;
  height: 600px;
  border-radius: 50%;
  filter: blur(100px);
  opacity: 0.4;
  animation: float 20s ease-in-out infinite;
}

.aurora-1 {
  background: radial-gradient(circle, rgba(56, 189, 248, 0.4) 0%, transparent 70%);
  top: -200px;
  right: -100px;
  animation-delay: 0s;
}

.aurora-2 {
  background: radial-gradient(circle, rgba(139, 92, 246, 0.4) 0%, transparent 70%);
  bottom: -200px;
  left: -100px;
  animation-delay: -7s;
}

.aurora-3 {
  background: radial-gradient(circle, rgba(236, 72, 153, 0.3) 0%, transparent 70%);
  top: 50%;
  left: 50%;
  animation-delay: -14s;
}

@keyframes float {
  0%, 100% { transform: translate(0, 0) scale(1); }
  25% { transform: translate(50px, -30px) scale(1.1); }
  50% { transform: translate(-30px, 50px) scale(0.9); }
  75% { transform: translate(-50px, -20px) scale(1.05); }
}

/* 粒子效果 */
.particles span {
  position: absolute;
  width: 2px;
  height: 2px;
  background: rgba(56, 189, 248, 0.6);
  border-radius: 50%;
  left: calc(var(--i) * 5%);
  top: 100%;
  animation: rise calc(10s + var(--i) * 1s) linear infinite;
  animation-delay: calc(var(--i) * 0.5s);
}

@keyframes rise {
  0% {
    top: 100%;
    opacity: 0;
    transform: translateX(0);
  }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% {
    top: -10%;
    opacity: 0;
    transform: translateX(calc(sin(var(--i)) * 100px));
  }
}

/* 导航链接 */
.nav-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 0.75rem;
  color: #94a3b8;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.nav-link::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(56, 189, 248, 0.1), rgba(139, 92, 246, 0.1));
  opacity: 0;
  transition: opacity 0.3s;
  border-radius: inherit;
}

.nav-link:hover {
  color: #e2e8f0;
  transform: translateY(-1px);
}

.nav-link:hover::before {
  opacity: 1;
}

.nav-link.active {
  color: #38bdf8;
  background: linear-gradient(135deg, rgba(56, 189, 248, 0.15), rgba(139, 92, 246, 0.15));
  box-shadow:
    0 0 20px rgba(56, 189, 248, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
}

.nav-icon {
  font-size: 1.1em;
  opacity: 0.7;
  transition: all 0.3s;
}

.nav-link:hover .nav-icon,
.nav-link.active .nav-icon {
  opacity: 1;
  transform: scale(1.1);
  filter: drop-shadow(0 0 8px currentColor);
}

/* 页面过渡 */
.page-enter-active {
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.page-leave-active {
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.page-enter-from {
  opacity: 0;
  transform: translateY(20px) scale(0.98);
  filter: blur(4px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-10px) scale(1.01);
  filter: blur(2px);
}
</style>