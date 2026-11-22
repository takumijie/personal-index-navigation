import { onMounted, ref } from 'vue'
import { publicApi } from '@/api'
import { errorMessage } from '@/api/client'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import type { Device, GuestbookEntry, NavigationItem, Profile, Settings, Track, Visitor } from '@/types'

export function useHomeData() {
  const profile = ref<Profile | null>(null)
  const navigation = ref<NavigationItem[]>([])
  const devices = ref<Device[]>([])
  const settings = ref<Settings>(structuredClone(DEFAULT_SETTINGS))
  const guestbook = ref<GuestbookEntry[]>([])
  const tracks = ref<Track[]>([])
  const visitor = ref<Visitor | null>(null)
  const loading = ref(true)
  const error = ref('')
  const terminalError = ref('')
  let terminalDataTask: Promise<void> | null = null

  function loadTerminalData(): Promise<void> {
    if (terminalDataTask) return terminalDataTask
    terminalDataTask = (async () => {
      terminalError.value = ''
      try {
        const [guestbookData, visitorData] = await Promise.all([publicApi.guestbook(), publicApi.visitor()])
        guestbook.value = guestbookData
        visitor.value = visitorData
      } catch (reason) {
        if (import.meta.env.DEV) {
          const mock = await import('@/api/mock')
          guestbook.value = mock.mockGuestbook
          visitor.value = mock.mockVisitor
        } else {
          guestbook.value = []
          visitor.value = null
          terminalError.value = errorMessage(reason)
        }
      }
    })()
    return terminalDataTask
  }

  async function load() {
    loading.value = true
    error.value = ''
    terminalDataTask = null
    void loadTerminalData()
    try {
      const [profileData, navigationData, deviceData, settingsData, trackData] = await Promise.all([
        publicApi.profile(), publicApi.navigation(), publicApi.devices(), publicApi.settings(), publicApi.playlist(),
      ])
      profile.value = profileData
      navigation.value = navigationData
      devices.value = deviceData
      settings.value = settingsData
      tracks.value = trackData
    } catch (reason) {
      if (import.meta.env.DEV) {
        const mock = await import('@/api/mock')
        profile.value = mock.mockProfile
        navigation.value = mock.mockNavigation
        devices.value = mock.mockDevices
        settings.value = mock.mockSettings
        tracks.value = mock.mockTracks
        error.value = '当前为开发预览数据，后端连接不可用'
      } else {
        error.value = errorMessage(reason)
      }
    } finally {
      loading.value = false
    }
  }

  onMounted(load)
  return { profile, navigation, devices, settings, guestbook, tracks, visitor, loading, error, terminalError, loadTerminalData, reload: load }
}
