<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { Music2, Pause, Play, Repeat, Repeat1, Shuffle, SkipBack, SkipForward, Volume1, Volume2, VolumeX } from 'lucide-vue-next'
import { publicApi } from '@/api'
import type { MusicSettings, Track } from '@/types'
import { findCurrentLrcLine, parseLrc, type LrcLine } from '@/utils/lrc'
import { playbackNeedsRefresh as playbackExpired } from '@/utils/playback'

type PlaybackMode = 'random' | 'sequential' | 'repeat-one'

const playbackModeKey = 'music-playback-mode'
const playbackModes: PlaybackMode[] = ['random', 'sequential', 'repeat-one']
const playbackModeLabels: Record<PlaybackMode, string> = {
  random: '随机播放',
  sequential: '顺序播放',
  'repeat-one': '单曲循环',
}

function readPlaybackMode(): PlaybackMode {
  try {
    const saved = localStorage.getItem(playbackModeKey) as PlaybackMode | null
    return saved && playbackModes.includes(saved) ? saved : 'random'
  } catch {
    return 'random'
  }
}

const props = defineProps<{ tracks: Track[]; settings: MusicSettings }>()
const audio = ref<HTMLAudioElement | null>(null)
const enabledTracks = ref<Track[]>([])
const currentIndex = ref(0)
const playing = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const volume = ref(0.65)
const previousVolume = ref(0.65)
const error = ref('')
const playbackUrl = ref('')
const coverFailed = ref(false)
const lyrics = ref<LrcLine[]>([])
const playbackMode = ref<PlaybackMode>(readPlaybackMode())
const currentTrack = computed(() => enabledTracks.value[currentIndex.value])
const playbackModeLabel = computed(() => playbackModeLabels[playbackMode.value])
const coverUrl = computed(() => currentTrack.value ? publicApi.coverUrl(currentTrack.value.id) : '')
const progress = computed(() => duration.value ? currentTime.value / duration.value * 100 : 0)
const currentLyricIndex = computed(() => findCurrentLrcLine(lyrics.value, currentTime.value))
const currentLyric = computed(() => currentLyricIndex.value >= 0 ? lyrics.value[currentLyricIndex.value].text : '')

let disposed = false
let trackSerial = 0
let playbackSerial = 0
let playbackExpiresAt = ''
let pendingPlaybackSerial = -1
let playRequested = false
let autoRetryUsed = false
let pendingSeek: { serial: number; time: number } | null = null

function sortedTracks(tracks: Track[]) {
  return tracks.filter((track) => track.enabled).sort((a, b) => a.sort - b.sort)
}

function isCurrentRequest(serial: number) {
  return !disposed && serial === trackSerial
}

function formatTime(value: number) {
  if (!Number.isFinite(value)) return '0:00'
  return `${Math.floor(value / 60)}:${Math.floor(value % 60).toString().padStart(2, '0')}`
}

async function loadLyrics(track: Track, serial: number) {
  if (!props.settings.lyrics_enabled) return
  try {
    const result = await publicApi.lyrics(track.id)
    if (!isCurrentRequest(serial)) return
    lyrics.value = result.format === 'lrc' ? parseLrc(result.text) : []
  } catch {
    if (isCurrentRequest(serial)) lyrics.value = []
  }
}

async function playLoadedAudio(serial: number) {
  if (!audio.value || !isCurrentRequest(serial) || !playRequested) return
  try {
    await audio.value.play()
  } catch {
    if (!isCurrentRequest(serial)) return
    playRequested = false
    playing.value = false
    error.value = '音频暂时无法播放'
  }
}

function clearPlayback() {
  playbackUrl.value = ''
  playbackSerial = 0
  playbackExpiresAt = ''
}

function playbackNeedsRefresh(serial: number): boolean {
  if (playbackSerial !== serial || !playbackUrl.value) return true
  return playbackExpired(playbackExpiresAt)
}

async function requestPlaybackAndPlay(track: Track, serial: number) {
  if (pendingPlaybackSerial === serial) return
  if (!playbackNeedsRefresh(serial)) {
    await playLoadedAudio(serial)
    return
  }

  if (currentTime.value > 0) pendingSeek = { serial, time: currentTime.value }
  clearPlayback()
  pendingPlaybackSerial = serial
  try {
    const playback = await publicApi.playback(track.id)
    if (!isCurrentRequest(serial)) return
    playbackUrl.value = playback.url
    playbackSerial = serial
    playbackExpiresAt = playback.expires_at || ''
    await nextTick()
    if (!audio.value || !isCurrentRequest(serial)) return
    audio.value.load()
    if (pendingSeek?.serial !== serial) await playLoadedAudio(serial)
  } catch {
    if (!isCurrentRequest(serial)) return
    clearPlayback()
    playRequested = false
    playing.value = false
    error.value = '音频暂时无法播放'
  } finally {
    if (pendingPlaybackSerial === serial) pendingPlaybackSerial = -1
  }
}

function togglePlay() {
  if (!audio.value || !currentTrack.value) return
  error.value = ''
  if (playing.value || playRequested) {
    playRequested = false
    audio.value.pause()
    return
  }
  playRequested = true
  autoRetryUsed = false
  void requestPlaybackAndPlay(currentTrack.value, trackSerial)
}

function changeTrack(index: number, continuePlaying = playRequested || playing.value) {
  const tracks = enabledTracks.value
  const serial = ++trackSerial
  playRequested = continuePlaying && tracks.length > 0
  autoRetryUsed = false
  pendingSeek = null
  clearPlayback()
  currentIndex.value = tracks.length ? (index + tracks.length) % tracks.length : 0
  currentTime.value = 0
  duration.value = 0
  error.value = ''
  lyrics.value = []
  coverFailed.value = false
  audio.value?.pause()

  const track = currentTrack.value
  void nextTick(() => {
    if (isCurrentRequest(serial)) audio.value?.load()
  })
  if (!track) {
    playRequested = false
    return
  }
  void loadLyrics(track, serial)
  if (playRequested) void requestPlaybackAndPlay(track, serial)
}

function cyclePlaybackMode() {
  const index = playbackModes.indexOf(playbackMode.value)
  playbackMode.value = playbackModes[(index + 1) % playbackModes.length]
  try {
    localStorage.setItem(playbackModeKey, playbackMode.value)
  } catch {
    // 无法持久化时仍允许当前页面切换播放模式。
  }
}

function randomTrackIndex() {
  const length = enabledTracks.value.length
  if (length <= 1) return 0
  const candidate = Math.floor(Math.random() * (length - 1))
  return candidate >= currentIndex.value ? candidate + 1 : candidate
}

function changeTrackBy(offset: number, continuePlaying = playRequested || playing.value) {
  const index = playbackMode.value === 'random' ? randomTrackIndex() : currentIndex.value + offset
  changeTrack(index, continuePlaying)
}

function handleEnded() {
  if (disposed) return
  if (playbackMode.value !== 'repeat-one') {
    changeTrackBy(1, true)
    return
  }
  if (!audio.value || !currentTrack.value) return
  error.value = ''
  playRequested = true
  pendingSeek = null
  audio.value.currentTime = 0
  currentTime.value = 0
  void requestPlaybackAndPlay(currentTrack.value, trackSerial)
}

function seek(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  if (audio.value && Number.isFinite(duration.value)) {
    const time = value / 100 * duration.value
    pendingSeek = { serial: trackSerial, time }
    audio.value.currentTime = time
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

function handleTimeUpdate() {
  if (!disposed) currentTime.value = audio.value?.currentTime || 0
}

function handleDurationChange() {
  if (!disposed) duration.value = audio.value?.duration || 0
}

function handleLoadedMetadata() {
  handleDurationChange()
  const target = pendingSeek
  if (disposed || !audio.value || !target || target.serial !== trackSerial || playbackSerial !== trackSerial) return
  pendingSeek = null
  audio.value.currentTime = target.time
  currentTime.value = audio.value.currentTime
  void playLoadedAudio(trackSerial)
}

function handleAudioError() {
  const track = currentTrack.value
  if (disposed || !track || !playbackUrl.value) return
  const serial = trackSerial
  clearPlayback()
  playing.value = false
  void nextTick(() => {
    if (isCurrentRequest(serial) && !playbackUrl.value) audio.value?.load()
  })
  if (autoRetryUsed) {
    playRequested = false
    error.value = '音频加载失败'
    return
  }
  autoRetryUsed = true
  error.value = ''
  playRequested = true
  void requestPlaybackAndPlay(track, serial)
}

onMounted(async () => {
  if (audio.value) audio.value.volume = volume.value
  try {
    const result = await publicApi.playlists()
    if (disposed) return
    const available = result
      .filter((playlist) => playlist.enabled)
      .sort((a, b) => a.sort - b.sort)
      .map((playlist) => ({ ...playlist, tracks: playlist.tracks.filter((track) => track.enabled) }))
    const selected = available.find((playlist) => playlist.id === props.settings.playlist_id) ?? available[0]
    enabledTracks.value = selected?.tracks ?? []
  } catch {
    if (disposed) return
    enabledTracks.value = sortedTracks(props.tracks)
  }
  changeTrack(playbackMode.value === 'random' ? randomTrackIndex() : 0, false)
})

onBeforeUnmount(() => {
  disposed = true
  trackSerial++
  playRequested = false
  audio.value?.pause()
})
</script>

<template>
  <div class="w-full">
    <audio
      ref="audio"
      :src="playbackUrl || undefined"
      preload="metadata"
      :volume="volume"
      @play="!disposed && (playing = true)"
      @pause="!disposed && (playing = false)"
      @timeupdate="handleTimeUpdate"
      @loadedmetadata="handleLoadedMetadata"
      @durationchange="handleDurationChange"
      @ended="handleEnded"
      @error="handleAudioError"
    />
    <div v-if="currentTrack" class="flex items-center gap-3.5">
      <div class="relative size-[62px] shrink-0 overflow-hidden rounded-xl border border-white/10 bg-gradient-to-br from-violet-500/25 to-indigo-950/50 shadow-lg">
        <img v-if="!coverFailed" :key="currentTrack.id" :src="coverUrl" :alt="`${currentTrack.title} 封面`" class="size-full object-cover" @error="coverFailed = true" />
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
            <p class="mt-0.5 h-4 truncate text-[10px] leading-4 text-violet-200/65" aria-live="polite">{{ currentLyric }}</p>
          </div>
          <span class="shrink-0 font-mono text-[9px] text-white/25">{{ currentIndex + 1 }} / {{ enabledTracks.length }}</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-7 font-mono text-[8px] text-white/28">{{ formatTime(currentTime) }}</span>
          <input class="music-range flex-1" type="range" min="0" max="100" step="0.1" :value="progress" aria-label="播放进度" @input="seek" />
          <span class="w-7 text-right font-mono text-[8px] text-white/28">{{ formatTime(duration) }}</span>
        </div>
        <div class="mt-1.5 flex items-center justify-between">
          <div class="flex items-center gap-0.5">
            <button class="icon-button size-7" type="button" :aria-label="`${playbackModeLabel}，点击切换播放模式`" :title="`${playbackModeLabel}，点击切换`" @click="cyclePlaybackMode">
              <Shuffle v-if="playbackMode === 'random'" :size="13" />
              <Repeat v-else-if="playbackMode === 'sequential'" :size="13" />
              <Repeat1 v-else :size="13" />
            </button>
            <button class="icon-button size-7" type="button" aria-label="上一曲" @click="changeTrackBy(-1)"><SkipBack :size="13" /></button>
            <button class="icon-button size-8 bg-white/10 text-white hover:bg-white/15" type="button" :aria-label="playing ? '暂停' : '播放'" @click="togglePlay">
              <Pause v-if="playing" :size="14" fill="currentColor" /><Play v-else :size="14" class="ml-0.5" fill="currentColor" />
            </button>
            <button class="icon-button size-7" type="button" aria-label="下一曲" @click="changeTrackBy(1)"><SkipForward :size="13" /></button>
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
