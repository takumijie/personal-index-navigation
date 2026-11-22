<script setup lang="ts">
import { computed } from 'vue'
import { MonitorSmartphone } from 'lucide-vue-next'
import AppIcon from '@/components/common/AppIcon.vue'
import SectionHeader from '@/components/common/SectionHeader.vue'
import type { Device } from '@/types'

const props = defineProps<{ devices: Device[] }>()
const sortedDevices = computed(() => props.devices.filter((device) => device.enabled).sort((a, b) => a.sort - b.sort))
</script>

<template>
  <section class="glass-card animate-enter p-5 [animation-delay:160ms]" aria-label="我的设备">
    <SectionHeader title="我的设备" subtitle="SETUP" :icon="MonitorSmartphone" />
    <div v-if="sortedDevices.length" class="grid grid-cols-1 gap-2 min-[420px]:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
      <article v-for="device in sortedDevices" :key="device.id" class="glass-subtle group flex min-w-0 items-center gap-3 rounded-xl p-3 transition hover:bg-white/[0.075]">
        <div class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-indigo-300/[0.07] text-indigo-100/55 transition group-hover:text-indigo-100/85">
          <AppIcon :name="device.icon" :size="17" />
        </div>
        <div class="min-w-0">
          <p class="mb-0.5 truncate text-[10px] tracking-wide text-white/28">{{ device.category }}</p>
          <h3 class="truncate text-[12px] font-medium text-white/75" :title="device.name">{{ device.name }}</h3>
          <p class="mt-0.5 truncate text-[10px] text-white/35" :title="device.description">{{ device.description }}</p>
        </div>
      </article>
    </div>
    <p v-else class="py-5 text-center text-xs text-white/35">暂无设备信息</p>
  </section>
</template>
