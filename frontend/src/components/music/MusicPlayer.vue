<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ListMusic, Music2, Pause, Play, SkipBack, SkipForward, Volume1, Volume2, VolumeX } from 'lucide-vue-next'
import type { Track } from '@/types'

const props = defineProps<{ tracks: Track[] }>()
const audio = ref<HTMLAudioElement | null>(null)
const currentIndex = ref(0)
const playing = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const volume = ref(0.65)
const previousVolume = ref(0.65)
const error = ref('')
const enabledTracks = computed(() => props.tracks.filter((track) => track.enabled).sort((a, b) => a.sort - b.sort))
const currentTrack = computed(() => enabledTracks.value[currentIndex.value])
const progress = computed(() => duration.value ? currentTime.value / duration.value * 100 : 0)

function formatTime(value: number) {
  if (!Number.isFinite(value)) return '0:00'
  return `${Math.floor(value / 60)}:${Math.floor(value % 60).toString().padStart(2, '0')}`
}

async function togglePlay() {
  if (!audio.value || !currentTrack.value) return
  error.value = ''
  if (playing.value) {
    audio.value.pause()
  } else {
    try { await audio.value.play() } catch { error.value = '音频暂时无法播放' }
  }
}

async function changeTrack(index: number, continuePlaying = playing.value) {
  if (!enabledTracks.value.length) return
  currentIndex.value = (index + enabledTracks.value.length) % enabledTracks.value.length
  currentTime.value = 0
  duration.value = 0
  await nextTick()
  audio.value?.load()
  if (continuePlaying && audio.value) {
    try { await audio.value.play() } catch { error.value = '音频暂时无法播放' }
  }
}

function seek(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  if (audio.value && Number.isFinite(duration.value)) {
    audio.value.currentTime = value / 100 * duration.value
    currentTime.value = audio.value.currentTime
  }
}

function setVolume(event: Event) {
  volume.value = Number((event.target as HTMLInputElement).value)
  if (audio.value) audio.value.volume = volume.value
  if (volume.value > 0) previousVolume.value = volume.value
}

function toggleMute() {
  volume.value = volume.value > 0 ? 0 : previousVolume.value || 0.65
  if (audio.value) audio.value.volume = volume.value
}

watch(enabledTracks, (tracks) => {
  if (currentIndex.value >= tracks.length) currentIndex.value = 0
})
onMounted(() => {
  if (audio.value) audio.value.volume = volume.value
})
onBeforeUnmount(() => audio.value?.pause())
</script>

<template>
  <div class="w-full">
    <audio
      ref="audio"
      :src="currentTrack?.url"
      preload="metadata"
      :volume="volume"
      @play="playing = true"
      @pause="playing = false"
      @timeupdate="currentTime = audio?.currentTime || 0"
      @loadedmetadata="duration = audio?.duration || 0"
      @durationchange="duration = audio?.duration || 0"
      @ended="changeTrack(currentIndex + 1, true)"
      @error="error = currentTrack ? '音频加载失败' : ''"
    />
    <div v-if="currentTrack" class="flex items-center gap-3.5">
      <div class="relative size-[62px] shrink-0 overflow-hidden rounded-xl border border-white/10 bg-gradient-to-br from-violet-500/25 to-indigo-950/50 shadow-lg">
        <img v-if="currentTrack.cover" :src="currentTrack.cover" :alt="`${currentTrack.title} 封面`" class="size-full object-cover" />
        <Music2 v-else :size="24" class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 text-white/45" />
        <div v-if="playing" class="absolute inset-x-0 bottom-0 flex h-5 items-end justify-center gap-[2px] bg-black/35 pb-1 backdrop-blur-sm">
          <i v-for="bar in 4" :key="bar" class="h-3 w-0.5 origin-bottom animate-[equalize_0.8s_ease-in-out_infinite] rounded-full bg-white/75" :style="{ animationDelay: `${bar * -0.16}s` }" />
        </div>
      </div>

      <div class="min-w-0 flex-1">
        <div class="mb-2 flex items-center justify-between gap-3">
          <div class="min-w-0">
            <h3 class="truncate text-[12px] font-medium text-white/80">{{ currentTrack.title }}</h3>
            <p class="mt-0.5 truncate text-[10px] text-white/35">{{ currentTrack.artist }}<span v-if="currentTrack.album"> · {{ currentTrack.album }}</span></p>
          </div>
          <ListMusic :size="13" class="shrink-0 text-white/25" :title="`${currentIndex + 1} / ${enabledTracks.length}`" />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-7 font-mono text-[8px] text-white/28">{{ formatTime(currentTime) }}</span>
          <input class="music-range flex-1" type="range" min="0" max="100" step="0.1" :value="progress" aria-label="播放进度" @input="seek" />
          <span class="w-7 text-right font-mono text-[8px] text-white/28">{{ formatTime(duration) }}</span>
        </div>
        <div class="mt-1.5 flex items-center justify-between">
          <div class="flex items-center gap-0.5">
            <button class="icon-button size-7" type="button" aria-label="上一曲" @click="changeTrack(currentIndex - 1)"><SkipBack :size="13" /></button>
            <button class="icon-button size-8 bg-white/10 text-white hover:bg-white/15" type="button" :aria-label="playing ? '暂停' : '播放'" @click="togglePlay">
              <Pause v-if="playing" :size="14" fill="currentColor" /><Play v-else :size="14" class="ml-0.5" fill="currentColor" />
            </button>
            <button class="icon-button size-7" type="button" aria-label="下一曲" @click="changeTrack(currentIndex + 1)"><SkipForward :size="13" /></button>
          </div>
          <div class="flex items-center gap-1">
            <button class="icon-button size-7" type="button" aria-label="静音切换" @click="toggleMute">
              <VolumeX v-if="volume === 0" :size="13" /><Volume1 v-else-if="volume < 0.5" :size="13" /><Volume2 v-else :size="13" />
            </button>
            <input class="music-range hidden w-14 min-[420px]:block" type="range" min="0" max="1" step="0.01" :value="volume" aria-label="音量" @input="setVolume" />
          </div>
        </div>
      </div>
    </div>
    <div v-else class="flex min-h-[70px] items-center justify-center gap-2 text-xs text-white/35"><Music2 :size="17" />暂无可用曲目</div>
    <p v-if="error" class="mt-1 text-right text-[10px] text-rose-300/75">{{ error }}</p>
  </div>
</template>

<style scoped>
.music-range { height: 3px; appearance: none; border-radius: 99px; outline: none; background: rgba(255,255,255,.12); accent-color: #c4b5fd; cursor: pointer; }
.music-range::-webkit-slider-thumb { width: 8px; height: 8px; appearance: none; border-radius: 50%; background: #ddd6fe; box-shadow: 0 0 0 3px rgba(196,181,253,.08); }
.music-range::-moz-range-thumb { width: 8px; height: 8px; border: 0; border-radius: 50%; background: #ddd6fe; }
</style>
