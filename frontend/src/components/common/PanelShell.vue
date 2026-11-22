<script setup lang="ts">
defineProps<{ title: string; description: string }>()
</script>

<template>
  <section class="glass-card overflow-hidden rounded-2xl">
    <header class="flex flex-col gap-4 border-b border-white/[0.07] px-5 py-5 min-[640px]:flex-row min-[640px]:items-center min-[640px]:justify-between min-[640px]:px-7">
      <div>
        <h2 class="text-lg font-medium tracking-tight text-white/90">{{ title }}</h2>
        <p class="mt-1 text-xs leading-5 text-white/38">{{ description }}</p>
      </div>
      <slot name="actions" />
    </header>
    <div class="p-5 min-[640px]:p-7"><slot /></div>
  </section>
</template>
