<script setup lang="ts">
withDefaults(defineProps<{ label?: string }>(), { label: '加载中' })
</script>

<template>
  <span class="inline-flex items-center gap-2" role="status">
    <span class="size-3.5 animate-spin rounded-full border border-current border-t-transparent opacity-70" aria-hidden="true" />
    <span>{{ label }}</span>
  </span>
</template>
