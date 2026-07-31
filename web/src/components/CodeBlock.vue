<script setup lang="ts">
import CopyButton from './CopyButton.vue'

withDefaults(defineProps<{
  code: string
  copyable?: boolean
}>(), {
  copyable: true,
})
</script>

<template>
  <div class="flex items-stretch overflow-hidden rounded-md border border-slate-800 bg-black/50 shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]">
    <div class="w-0.5 shrink-0 bg-gradient-to-b from-violet-500/70 via-cyan-400/50 to-transparent" aria-hidden="true" />
    <code class="min-w-0 flex-1 whitespace-pre-wrap break-all px-3 py-2 font-mono text-xs text-emerald-300 md:overflow-x-auto md:whitespace-nowrap">{{ code }}</code>
    <div v-if="copyable" class="flex shrink-0 items-center border-l border-slate-800/80 px-1.5">
      <CopyButton :text="code" />
    </div>
  </div>
</template>
