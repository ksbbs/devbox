<script setup lang="ts">
import { ref } from 'vue'
import { login } from '../api/client'

const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  if (!password.value) return
  loading.value = true
  error.value = ''
  try {
    await login(password.value)
    window.location.href = '/'
  } catch {
    error.value = '密码错误'
  }
  loading.value = false
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center">
    <div class="login-card p-8 rounded-2xl w-full max-w-sm">
      <div class="flex items-center justify-center mb-6">
        <div class="w-14 h-14 rounded-2xl bg-gradient-to-br from-sky-400 via-violet-500 to-fuchsia-500 flex items-center justify-center shadow-lg shadow-sky-500/20">
          <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
          </svg>
        </div>
      </div>
      <h2 class="text-2xl font-bold text-white text-center mb-2">DevBox</h2>
      <p class="text-slate-400 text-center text-sm mb-6">输入密码以访问 Dashboard</p>

      <div v-if="error" class="mb-4 px-4 py-2 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm text-center">
        {{ error }}
      </div>

      <input
        v-model="password"
        type="password"
        placeholder="密码"
        @keydown.enter="submit"
        class="w-full bg-slate-900/50 border border-slate-700 rounded-xl px-4 py-3 text-sm text-slate-200 focus:border-sky-500/50 focus:outline-none focus:ring-2 focus:ring-sky-500/20 transition-all mb-4"
        :disabled="loading"
      />

      <button
        @click="submit"
        :disabled="loading || !password"
        class="w-full py-3 rounded-xl font-medium transition-all"
        :class="loading || !password
          ? 'bg-slate-700 text-slate-500 cursor-not-allowed'
          : 'bg-gradient-to-r from-sky-500 via-violet-500 to-fuchsia-500 text-white hover:shadow-lg hover:shadow-violet-500/25'"
      >
        {{ loading ? '验证中...' : '登录' }}
      </button>
    </div>
  </div>
</template>

<style>
.login-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.05) 0%, rgba(255, 255, 255, 0.02) 100%);
  border: 1px solid rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(20px);
}
</style>