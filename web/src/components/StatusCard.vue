<script setup lang="ts">
import StatusDot from './StatusDot.vue'

defineProps<{ mirror: any }>()
</script>

<template>
  <div class="glass glass-hover p-4">
    <div class="mb-3 flex items-center justify-between gap-2">
      <h3 class="truncate text-[15px] font-semibold text-slate-100">{{ mirror.name }}</h3>
      <span v-if="mirror.status === 'healthy' && mirror.enabled" class="tag tag-ok">
        <StatusDot tone="ok" pulse />
        健康
      </span>
      <span v-else-if="mirror.status !== 'healthy' && mirror.enabled" class="tag" :class="'tag-warn'">
        <StatusDot tone="danger" />
        异常
      </span>
      <span v-else class="tag tag-off">
        <StatusDot tone="off" />
        已停用
      </span>
    </div>
    <div class="space-y-1.5 text-xs text-slate-500">
      <div class="flex min-w-0 items-center gap-2">
        <span class="shrink-0 uppercase tracking-[0.14em]">路径</span>
        <span class="truncate font-mono text-slate-300">{{ mirror.pattern }}</span>
      </div>
      <div class="flex min-w-0 items-center gap-2">
        <span class="shrink-0 uppercase tracking-[0.14em]">上游</span>
        <span class="truncate text-slate-400">{{ mirror.upstream }}</span>
      </div>
      <p v-if="mirror.error" class="mt-2 text-xs text-red-400">{{ mirror.error }}</p>
    </div>
  </div>
</template>
