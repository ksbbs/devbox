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
    <h2 class="text-2xl font-semibold mb-6">Dashboard</h2>
    <div v-if="loading" class="text-slate-400">Loading...</div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <StatusCard v-for="m in mirrors" :key="m.name" :mirror="m" />
    </div>
    <div v-if="!loading && mirrors.length === 0" class="text-slate-400 mt-4">
      No mirrors configured. Check your config file.
    </div>
  </div>
</template>