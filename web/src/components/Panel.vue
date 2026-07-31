<script setup lang="ts">
withDefaults(defineProps<{
  pad?: boolean
  hover?: boolean
}>(), {
  pad: true,
  hover: false,
})
</script>

<template>
  <section class="glass" :class="[pad ? 'p-4' : '', hover ? 'glass-hover' : '']">
    <header v-if="$slots.title" class="mb-4 flex items-start justify-between gap-3 border-b border-slate-800/70 pb-3">
      <div class="min-w-0">
        <h2 class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-400">
          <slot name="title" />
        </h2>
        <p v-if="$slots.desc" class="mt-1 text-xs text-slate-600">
          <slot name="desc" />
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <slot name="action" />
      </div>
    </header>
    <slot />
  </section>
</template>
