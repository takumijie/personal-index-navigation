<script setup lang="ts">
import { computed, ref } from 'vue'
import { AtSign, Download, Sparkles } from 'lucide-vue-next'
import type { Profile } from '@/types'
import { DEFAULT_DAY_BACKGROUND } from '@/data/defaults'
import { isVideoUrl } from '@/utils/media'

const props = defineProps<{ profile: Profile; activeBackground: string; dayBackground: string; nightBackground: string }>()
const avatarFailed = ref(false)
// 封面展示与下载始终是图片：背景为视频时回退到默认静态图
function staticCover(url: string) {
  return isVideoUrl(url) ? DEFAULT_DAY_BACKGROUND : url
}
const cover = computed(() => {
  if (props.profile.cover_mode === 'custom') return props.profile.cover_url || staticCover(props.activeBackground)
  if (props.profile.cover_mode === 'day') return staticCover(props.dayBackground)
  if (props.profile.cover_mode === 'night') return staticCover(props.nightBackground)
  return staticCover(props.activeBackground)
})

function downloadCover() {
  if (!cover.value) return
  const anchor = document.createElement('a')
  anchor.href = cover.value
  anchor.download = `${props.profile.username.replace('@', '') || 'profile'}-cover.jpg`
  anchor.target = '_blank'
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}
</script>

<template>
  <section class="glass-card animate-enter overflow-hidden" aria-label="个人资料">
    <div class="group relative h-[104px] overflow-hidden bg-gradient-to-br from-indigo-500/40 to-purple-900/60">
      <img v-if="cover" :src="cover" alt="个人资料封面" class="size-full object-cover transition duration-700 group-hover:scale-105" />
      <div class="absolute inset-0 bg-gradient-to-t from-[#17182d]/85 via-transparent to-black/5" />
      <button class="icon-button absolute right-3 top-3 size-8 border border-white/10 bg-black/20 backdrop-blur-md" type="button" title="下载当前封面" aria-label="下载当前封面" @click="downloadCover">
        <Download :size="14" />
      </button>
      <div class="absolute bottom-3 right-4 flex items-center gap-1.5 text-[9px] tracking-[0.16em] text-white/55">
        <Sparkles :size="11" /> STAY CURIOUS
      </div>
    </div>

    <div class="relative px-5 pb-5">
      <div class="absolute -top-10 left-5 size-[74px] rounded-2xl border-[3px] border-[#22233a] bg-gradient-to-br from-violet-400 to-indigo-700 shadow-xl shadow-black/25">
        <img v-if="profile.avatar && !avatarFailed" :src="profile.avatar" :alt="`${profile.name} 的头像`" class="size-full rounded-[13px] object-cover" @error="avatarFailed = true" />
        <span v-else class="flex size-full items-center justify-center text-2xl font-semibold text-white/80">{{ profile.name.slice(0, 1) }}</span>
        <span class="absolute -bottom-0.5 -right-0.5 size-3.5 rounded-full border-[3px] border-[#22233a] bg-emerald-400" title="在线" />
      </div>

      <div class="pt-11">
        <div class="flex items-baseline justify-between gap-3">
          <h1 class="truncate text-[22px] font-semibold tracking-tight text-white">{{ profile.name }}</h1>
          <span class="flex shrink-0 items-center text-[11px] text-violet-200/45"><AtSign :size="11" class="mr-0.5" />{{ profile.username.replace('@', '') }}</span>
        </div>
        <p class="mt-2 min-h-10 text-[12px] leading-[1.75] text-white/52">{{ profile.description }}</p>
        <div v-if="profile.tags.length" class="mt-3.5 flex flex-wrap gap-1.5">
          <span v-for="tag in profile.tags" :key="tag" class="rounded-md border border-violet-200/10 bg-violet-200/[0.055] px-2 py-1 text-[10px] text-violet-100/60">{{ tag }}</span>
        </div>
      </div>
    </div>
  </section>
</template>
