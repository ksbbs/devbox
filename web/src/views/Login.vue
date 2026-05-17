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
    error.value = 'auth rejected'
  }
  loading.value = false
}
</script>

<template>
  <div class="flex min-h-[70vh] items-center justify-center">
    <section class="panel-pad w-full max-w-md">
      <div class="mb-6 border-b border-slate-900 pb-4">
        <span class="page-kicker">auth</span>
        <h1 class="mt-2 text-2xl font-semibold text-slate-100">DevBox Console</h1>
        <p class="mt-1 text-sm text-slate-500">输入 AUTH_TOKEN 访问控制台。</p>
      </div>

      <div v-if="error" class="mb-4 border border-red-500/40 bg-red-950/30 px-3 py-2 text-sm text-red-300">
        {{ error }}
      </div>

      <label class="grid gap-2">
        <span class="text-xs uppercase tracking-[0.16em] text-slate-500">token</span>
        <input
          v-model="password"
          type="password"
          class="input w-full"
          placeholder="••••••••"
          :disabled="loading"
          autofocus
          @keydown.enter="submit"
        />
      </label>

      <button class="btn btn-primary mt-4 w-full" :disabled="loading || !password" @click="submit">
        {{ loading ? 'verifying' : 'login' }}
      </button>
    </section>
  </div>
</template>
