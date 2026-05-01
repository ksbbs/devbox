<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getStatus } from '../api/client'
import StatusCard from '../components/StatusCard.vue'

const mirrors = ref<any[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    mirrors.value = await getStatus()
  } catch (e) {
    mirrors.value = []
  }
  loading.value = false
})
</script>

<template>
  <div>
    <h2 class="text-2xl font-semibold mb-6 text-white">Dashboard</h2>
    <div v-if="loading" class="flex items-center gap-2 text-slate-400">
      <span class="loading-dot"></span>
      <span class="loading-dot"></span>
      <span class="loading-dot"></span>
    </div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <TransitionGroup appear name="card">
        <StatusCard v-for="(m, i) in mirrors" :key="m.name" :mirror="m" :index="i" />
      </TransitionGroup>
    </div>
    <div v-if="!loading && mirrors.length === 0" class="text-slate-400 mt-4">
      No mirrors configured. Check your config file.
    </div>
  </div>
</template>

<style>
.loading-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #38bdf8;
  animation: pulse 1.2s ease-in-out infinite;
}
.loading-dot:nth-child(2) { animation-delay: 0.2s; }
.loading-dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes pulse {
  0%, 100% { opacity: 0.3; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1); }
}

.card-enter-from {
  opacity: 0;
  transform: translateY(16px) scale(0.95);
}
.card-enter-active {
  transition: all 0.35s ease-out;
}
.card-enter-to {
  opacity: 1;
  transform: translateY(0) scale(1);
}
</style>