<script setup lang="ts">
defineProps<{ hands: { hour: number; minute: number; second: number } }>()
const markers = Array.from({ length: 60 }, (_, index) => index)
</script>

<template>
  <div class="relative size-[108px] shrink-0 rounded-full border border-white/10 bg-[radial-gradient(circle_at_45%_35%,rgba(255,255,255,0.055),rgba(5,7,20,0.32))] shadow-[inset_0_0_28px_rgba(0,0,0,0.18),0_8px_25px_rgba(0,0,0,0.15)]" role="img" aria-label="模拟时钟">
    <span
      v-for="marker in markers"
      :key="marker"
      class="absolute left-1/2 top-1/2 h-[46px] w-px origin-top -translate-x-1/2"
      :style="{ transform: `translateX(-50%) rotate(${marker * 6}deg)` }"
    >
      <span class="block w-px bg-white/25" :class="marker % 5 === 0 ? 'h-[5px] bg-white/45' : 'h-[2px] opacity-35'" />
    </span>
    <span class="absolute left-1/2 top-1/2 h-[27px] w-[3px] origin-top -translate-x-1/2 rounded-full bg-white/80 shadow" :style="{ transform: `translateX(-50%) rotate(${hands.hour + 180}deg)` }" />
    <span class="absolute left-1/2 top-1/2 h-[37px] w-[2px] origin-top -translate-x-1/2 rounded-full bg-violet-100/85" :style="{ transform: `translateX(-50%) rotate(${hands.minute + 180}deg)` }" />
    <span class="absolute left-1/2 top-1/2 h-[41px] w-px origin-top -translate-x-1/2 bg-fuchsia-300/85" :style="{ transform: `translateX(-50%) rotate(${hands.second + 180}deg)` }" />
    <span class="absolute left-1/2 top-1/2 size-2 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-[#292640] bg-fuchsia-200 shadow" />
  </div>
</template>
