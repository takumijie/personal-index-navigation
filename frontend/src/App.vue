<script setup lang="ts">
import { onMounted } from 'vue'

onMounted(() => {
  document.documentElement.style.removeProperty('background')
})
</script>

<template>
  <RouterView />
</template>
