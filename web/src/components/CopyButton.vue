<script setup lang="ts">
import { ref } from 'vue'

const props = withDefaults(defineProps<{
  text: string
  label?: string
  copiedLabel?: string
  variant?: 'default' | 'primary' | 'brand'
}>(), {
  label: '复制',
  copiedLabel: '已复制',
  variant: 'default',
})

const copied = ref(false)
let timer: ReturnType<typeof setTimeout> | null = null

async function copy() {
  let ok = true
  try {
    await navigator.clipboard.writeText(props.text)
  } catch {
    const el = document.createElement('textarea')
    el.value = props.text
    document.body.appendChild(el)
    el.select()
    ok = document.execCommand('copy')
    document.body.removeChild(el)
  }
  if (!ok) return
  copied.value = true
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => (copied.value = false), 1500)
}
</script>

<template>
  <button
    type="button"
    class="btn"
    :class="[variant === 'primary' ? 'btn-primary' : '', variant === 'brand' ? 'btn-brand' : '', copied ? 'border-emerald-400/50 text-emerald-300' : '']"
    @click="copy"
  >
    <svg v-if="copied" class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <polyline points="2.5 8.5 6 12 13.5 4.5" />
    </svg>
    <svg v-else class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" aria-hidden="true">
      <rect x="5.5" y="5.5" width="8" height="8" rx="1.5" />
      <path d="M10.5 5.5v-2a1.5 1.5 0 0 0-1.5-1.5h-5a1.5 1.5 0 0 0-1.5 1.5v5a1.5 1.5 0 0 0 1.5 1.5h2" />
    </svg>
    {{ copied ? copiedLabel : label }}
  </button>
</template>
