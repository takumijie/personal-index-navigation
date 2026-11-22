<script setup lang="ts">
import { toRef } from 'vue'
import { Clock3 } from 'lucide-vue-next'
import AnalogClock from './AnalogClock.vue'
import MusicPlayer from '@/components/music/MusicPlayer.vue'
import { useClock } from '@/composables/useClock'
import type { Settings, Track } from '@/types'

const props = defineProps<{ settings: Settings; tracks: Track[] }>()
const { zonedTime, zonedDate, hands } = useClock(toRef(props, 'settings'))
</script>

<template>
  <section class="glass-card animate-enter flex flex-col gap-5 p-5 [animation-delay:240ms] md:flex-row md:items-center" aria-label="时钟与音乐">
    <div class="flex shrink-0 items-center gap-5">
      <AnalogClock :hands="hands" />
      <div class="min-w-[128px]">
        <div class="mb-1.5 flex items-center gap-1.5 text-[9px] tracking-[0.15em] text-white/35"><Clock3 :size="11" />{{ settings.clock_label }}</div>
        <time class="block whitespace-nowrap font-mono text-[26px] font-light tracking-[-0.055em] text-white/90">{{ zonedTime }}</time>
        <p class="mt-1 text-[10px] text-white/38">{{ zonedDate }}</p>
      </div>
    </div>
    <div class="hidden h-[82px] w-px bg-white/[0.08] md:block" />
    <MusicPlayer :tracks="tracks" :settings="settings.music" class="min-w-0 flex-1" />
  </section>
</template>
