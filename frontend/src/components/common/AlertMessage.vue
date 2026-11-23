<script setup lang="ts">
import { AlertCircle, CheckCircle2, Info } from 'lucide-vue-next'

withDefaults(defineProps<{ message: string; type?: 'error' | 'success' | 'info' | 'warning' }>(), { type: 'info' })
</script>

<template>
  <div v-if="message" class="mb-4 flex items-start gap-2.5 rounded-xl border px-3.5 py-3 text-xs leading-5" :class="{
    'border-rose-300/15 bg-rose-400/[0.07] text-rose-200/85': type === 'error',
    'border-emerald-300/15 bg-emerald-400/[0.07] text-emerald-200/85': type === 'success',
    'border-indigo-300/15 bg-indigo-400/[0.07] text-indigo-200/80': type === 'info',
    'border-amber-300/15 bg-amber-400/[0.07] text-amber-100/85': type === 'warning',
  }" role="status">
    <AlertCircle v-if="type === 'error' || type === 'warning'" :size="15" class="mt-0.5 shrink-0" />
    <CheckCircle2 v-else-if="type === 'success'" :size="15" class="mt-0.5 shrink-0" />
    <Info v-else :size="15" class="mt-0.5 shrink-0" />
    <span>{{ message }}</span>
  </div>
</template>
