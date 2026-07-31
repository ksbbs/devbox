<script setup lang="ts">
import { ref } from 'vue'
import { login } from '../api/client'
import Logo from '../components/Logo.vue'
import Banner from '../components/Banner.vue'
import StatusDot from '../components/StatusDot.vue'

const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  if (!password.value) return
  loading.value = true
  error.value = ''
  try {
    await login(password.value)
    // Only allow relative-path redirects to avoid open-redirect abuse.
    // Reject protocol-relative URLs too: `//evil.com` and `/\evil.com`
    // start with "/" but navigate to an external host.
    const redirect = new URLSearchParams(window.location.search).get('redirect')
    const safe =
      redirect &&
      redirect.startsWith('/') &&
      !redirect.startsWith('//') &&
      !redirect.startsWith('/\\')
    window.location.href = safe ? redirect : '/'
  } catch {
    error.value = '认证被拒绝，请检查 AUTH_TOKEN'
  }
  loading.value = false
}
</script>

<template>
  <div class="flex min-h-[75vh] items-center justify-center">
    <section class="glass w-full max-w-md p-6">
      <div class="mb-6 flex flex-col items-center border-b border-slate-800/80 pb-5 text-center">
        <div class="mb-3 flex items-center gap-3">
          <Logo :size="30" />
          <span class="text-base font-semibold tracking-wide text-slate-100">devbox</span>
        </div>
        <span class="page-kicker">认证</span>
        <h1 class="mt-2 text-xl font-semibold text-slate-100">DevBox 控制台</h1>
        <p class="mt-1 text-sm text-slate-500">输入 AUTH_TOKEN 访问控制台。</p>
      </div>

      <Banner v-if="error" :message="error" />

      <label class="grid gap-2">
        <span class="text-xs uppercase tracking-[0.16em] text-slate-500">令牌</span>
        <div class="relative">
          <input
            v-model="password"
            type="password"
            class="input w-full pr-9"
            placeholder="••••••••"
            :disabled="loading"
            autofocus
            @keydown.enter="submit"
          />
          <svg v-if="!loading" class="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-600" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" aria-hidden="true">
            <rect x="3" y="7" width="10" height="7" rx="1.5" />
            <path d="M5.5 7V4.5a2.5 2.5 0 0 1 5 0V7" />
          </svg>
        </div>
      </label>

      <button class="btn btn-primary mt-4 w-full" :disabled="loading || !password" @click="submit">
        <StatusDot v-if="loading" tone="accent" pulse />
        {{ loading ? '验证中' : '登录' }}
      </button>
    </section>
  </div>
</template>
