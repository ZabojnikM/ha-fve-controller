<script setup lang="ts">
defineProps<{ name: 'sun' | 'battery' | 'water' | 'car' }>()
</script>
<template>
  <svg class="energy-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <g v-if="name === 'sun'"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5"/></g>
    <g v-else-if="name === 'battery'"><rect x="5" y="4" width="14" height="18" rx="3"/><path d="M9 4V2h6v2M8 16h8m-8 3h8M12 7l-2 4h4l-2 3"/></g>
    <g v-else-if="name === 'water'"><path d="M12 2C9 6 5 10 5 14a7 7 0 0014 0c0-4-4-8-7-12Z M8 15a4 4 0 004 4"/></g>
    <g v-else><path d="M4 10l2-6h12l2 6M3 10h18v8H3zM5 18v3m14-3v3M6 14h2m8 0h2M8 7h8"/></g>
  </svg>
</template>
