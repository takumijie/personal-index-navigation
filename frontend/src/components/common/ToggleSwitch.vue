<script setup lang="ts">
const model = defineModel<boolean>({ required: true })
defineProps<{ label: string; disabled?: boolean }>()
</script>

<template>
  <label class="flex cursor-pointer items-center justify-between gap-4 text-sm text-white/75">
    <span>{{ label }}</span>
    <input v-model="model" type="checkbox" class="peer sr-only" :disabled="disabled" />
    <span class="relative h-5 w-9 rounded-full bg-white/12 transition peer-checked:bg-violet-500 peer-disabled:opacity-40 after:absolute after:left-0.5 after:top-0.5 after:size-4 after:rounded-full after:bg-white after:shadow after:transition-transform peer-checked:after:translate-x-4" aria-hidden="true" />
  </label>
</template>
