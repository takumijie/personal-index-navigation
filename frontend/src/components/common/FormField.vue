<script setup lang="ts">
defineProps<{ label: string; hint?: string; error?: string }>()
</script>

<template>
  <label class="block">
    <span class="mb-1.5 flex items-center justify-between gap-2 text-xs font-medium text-white/58">
      <span>{{ label }}</span>
      <span v-if="hint" class="font-normal text-white/30">{{ hint }}</span>
    </span>
    <slot />
    <span v-if="error" class="mt-1 block text-xs text-rose-300">{{ error }}</span>
  </label>
</template>
