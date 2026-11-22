<script setup lang="ts">
defineProps<{
  dayUrl: string
  nightUrl: string
  mode: 'day' | 'night'
  loaded: ReadonlySet<string>
  duration: number
}>()
</script>

<template>
  <div class="fixed inset-0 -z-20 overflow-hidden bg-[#17162e]" aria-hidden="true">
    <div
      v-for="layer in [{ key: 'day', url: dayUrl }, { key: 'night', url: nightUrl }]"
      :key="layer.key"
      class="absolute inset-0 scale-[1.025] bg-cover bg-center will-change-[opacity]"
      :class="loaded.has(layer.url) ? '' : 'opacity-0'"
      :style="{
        backgroundImage: `url('${layer.url}')`,
        opacity: loaded.has(layer.url) && mode === layer.key ? 1 : 0,
        transition: `opacity ${duration}ms ease-in-out`,
      }"
    />
    <div class="absolute inset-0 bg-[radial-gradient(circle_at_72%_10%,transparent_0%,rgba(10,10,28,0.1)_45%,rgba(6,7,21,0.48)_100%)]" />
    <div class="absolute inset-0 bg-gradient-to-b from-[#111127]/15 via-[#111127]/5 to-[#090b1c]/55" />
    <div class="absolute inset-0 opacity-[0.035] [background-image:url('data:image/svg+xml,%3Csvg_viewBox=%220_0_180_180%22_xmlns=%22http://www.w3.org/2000/svg%22%3E%3Cfilter_id=%22n%22%3E%3CfeTurbulence_type=%22fractalNoise%22_baseFrequency=%22.8%22_numOctaves=%223%22_stitchTiles=%22stitch%22/%3E%3C/filter%3E%3Crect_width=%22100%25%22_height=%22100%25%22_filter=%22url(%23n)%22/%3E%3C/svg%3E')]" />
  </div>
</template>
