<script setup lang="ts">
import { computed, ref } from 'vue'
import { ArrowUpRight, Compass } from 'lucide-vue-next'
import AppIcon from '@/components/common/AppIcon.vue'
import SectionHeader from '@/components/common/SectionHeader.vue'
import type { NavigationItem } from '@/types'

const props = defineProps<{ items: NavigationItem[] }>()
const failedIcons = ref(new Set<number>())
const sortedItems = computed(() => props.items.filter((item) => item.enabled).sort((a, b) => a.sort - b.sort))

function markFailed(id: number) {
  failedIcons.value = new Set(failedIcons.value).add(id)
}
</script>

<template>
  <section class="glass-card animate-enter p-5 [animation-delay:80ms]" aria-label="快速导航">
    <SectionHeader title="快速导航" :subtitle="`${sortedItems.length} LINKS`" :icon="Compass" />
    <div v-if="sortedItems.length" class="grid grid-cols-2 gap-2.5">
      <a
        v-for="item in sortedItems"
        :key="item.id"
        :href="item.url"
        :target="item.open_mode === 'blank' ? '_blank' : '_self'"
        :rel="item.open_mode === 'blank' ? 'noopener noreferrer' : undefined"
        class="group glass-subtle relative flex min-h-[76px] flex-col justify-between overflow-hidden rounded-xl p-3 transition duration-300 hover:-translate-y-0.5 hover:border-violet-200/20 hover:bg-violet-200/[0.085]"
      >
        <div class="flex items-start justify-between">
          <span class="flex size-8 items-center justify-center rounded-lg bg-gradient-to-br from-white/10 to-white/[0.035] text-violet-100/75 shadow-inner">
            <img v-if="item.icon_url && !failedIcons.has(item.id)" :src="item.icon_url" alt="" class="size-5 object-contain" @error="markFailed(item.id)" />
            <AppIcon v-else :name="item.icon" :size="16" />
          </span>
          <ArrowUpRight :size="12" class="text-white/20 transition group-hover:-translate-y-0.5 group-hover:translate-x-0.5 group-hover:text-white/55" />
        </div>
        <span class="mt-2 truncate text-[12px] font-medium text-white/78">{{ item.title }}</span>
      </a>
    </div>
    <p v-else class="py-6 text-center text-xs text-white/35">暂无可用导航</p>
  </section>
</template>
