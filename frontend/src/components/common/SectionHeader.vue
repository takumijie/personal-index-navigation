<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{ title: string; subtitle?: string; icon?: Component }>()
</script>

<template>
  <div class="mb-4 flex items-center justify-between gap-3">
    <div class="flex items-center gap-2.5">
      <component :is="icon" v-if="icon" :size="15" class="text-violet-300/80" />
      <h2 class="text-[11px] font-semibold tracking-[0.18em] text-white/55 uppercase">{{ title }}</h2>
    </div>
    <span v-if="subtitle" class="text-[10px] tracking-wide text-white/30">{{ subtitle }}</span>
    <slot />
  </div>
</template>
