<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { isVideoUrl } from '@/utils/media'

const props = withDefaults(defineProps<{
  dayUrl: string
  nightUrl: string
  mode: 'day' | 'night'
  loaded: ReadonlySet<string>
  duration: number
  reducedMotion?: boolean
}>(), { reducedMotion: false })

const layers = computed(() => [
  { key: 'day' as const, url: props.dayUrl, video: isVideoUrl(props.dayUrl) },
  { key: 'night' as const, url: props.nightUrl, video: isVideoUrl(props.nightUrl) },
])

const videoElements: Record<string, HTMLVideoElement | null> = {}

// 仅激活层播放视频，减少解码开销；减少动态偏好时停留在首帧
function syncPlayback(element: HTMLVideoElement, key: string) {
  if (props.mode === key && !props.reducedMotion) {
    element.play().catch(() => {})
  } else {
    element.pause()
  }
}

function setVideoRef(key: string, element: Element | null) {
  const video = element as HTMLVideoElement | null
  videoElements[key] = video
  // ref 回调不是响应式依赖，挂载时立即同步一次播放状态
  if (video) syncPlayback(video, key)
}

watchEffect(() => {
  for (const layer of layers.value) {
    const element = videoElements[layer.key]
    if (element) syncPlayback(element, layer.key)
  }
})
</script>

<template>
  <div class="fixed inset-0 -z-20 overflow-hidden bg-[#17162e]" aria-hidden="true">
    <template v-for="layer in layers" :key="layer.key">
      <video
        v-if="layer.video"
        :ref="(element) => setVideoRef(layer.key, element as Element | null)"
        :src="layer.url"
        muted
        loop
        playsinline
        autoplay
        preload="auto"
        class="absolute inset-0 size-full scale-[1.025] object-cover will-change-[opacity]"
        :class="loaded.has(layer.url) ? '' : 'opacity-0'"
        :style="{
          opacity: loaded.has(layer.url) && mode === layer.key ? 1 : 0,
          transition: `opacity ${duration}ms ease-in-out`,
        }"
      />
      <div
        v-else
        class="absolute inset-0 scale-[1.025] bg-cover bg-center will-change-[opacity]"
        :class="loaded.has(layer.url) ? '' : 'opacity-0'"
        :style="{
          backgroundImage: `url('${layer.url}')`,
          opacity: loaded.has(layer.url) && mode === layer.key ? 1 : 0,
          transition: `opacity ${duration}ms ease-in-out`,
        }"
      />
    </template>
    <div class="absolute inset-0 bg-[radial-gradient(circle_at_72%_10%,transparent_0%,rgba(10,10,28,0.1)_45%,rgba(6,7,21,0.48)_100%)]" />
    <div class="absolute inset-0 bg-gradient-to-b from-[#111127]/15 via-[#111127]/5 to-[#090b1c]/55" />
    <div class="absolute inset-0 opacity-[0.035] [background-image:url('data:image/svg+xml,%3Csvg_viewBox=%220_0_180_180%22_xmlns=%22http://www.w3.org/2000/svg%22%3E%3Cfilter_id=%22n%22%3E%3CfeTurbulence_type=%22fractalNoise%22_baseFrequency=%22.8%22_numOctaves=%223%22_stitchTiles=%22stitch%22/%3E%3C/filter%3E%3Crect_width=%22100%25%22_height=%22100%25%22_filter=%22url(%23n)%22/%3E%3C/svg%3E')]" />
  </div>
</template>
