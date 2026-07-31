<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 24 })
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
    <defs>
      <linearGradient id="dbTop" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stop-color="#f2e4ff"/>
        <stop offset="100%" stop-color="#bd8bff"/>
      </linearGradient>
      <linearGradient id="dbLeft" x1="0" y1="0" x2="1" y2="0">
        <stop offset="0%" stop-color="#7c3aed"/>
        <stop offset="100%" stop-color="#9455f5"/>
      </linearGradient>
      <linearGradient id="dbRight" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#6528d4"/>
        <stop offset="100%" stop-color="#43158f"/>
      </linearGradient>
    </defs>
    <polygon points="12,2 19.75,6.75 12,11.5 4.25,6.75" fill="url(#dbTop)"/>
    <polygon points="12,11.5 19.75,6.75 19.75,16.25 12,21" fill="url(#dbRight)"/>
    <polygon points="12,11.5 12,21 4.25,16.25 4.25,6.75" fill="url(#dbLeft)"/>
    <polygon points="12,4 16.5,6.4 12,8.8 7.5,6.4" fill="#ffffff" opacity="0.16"/>
    <g stroke="#67e8f9" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="13.25,13.75 15.25,15.25 13.25,16.75"/>
      <line x1="16.25" y1="16.75" x2="17.75" y2="16.75"/>
    </g>
  </svg>
</template>
