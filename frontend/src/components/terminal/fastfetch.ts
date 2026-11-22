import androidLogo from '@/assets/fastfetch/android.png'
import chromeosLogo from '@/assets/fastfetch/chromeos.png'
import freebsdLogo from '@/assets/fastfetch/freebsd.png'
import iosLogo from '@/assets/fastfetch/ios.png'
import linuxLogo from '@/assets/fastfetch/linux.png'
import macosLogo from '@/assets/fastfetch/macos.png'
import windowsLogo from '@/assets/fastfetch/windows.png'
import type { SystemInfo, Visitor } from '@/types'

export interface FastfetchDetail {
  label: string
  value: string
}

export interface FastfetchData {
  logo: string | null
  identity: string
  details: FastfetchDetail[]
}

const logos: Record<string, string> = {
  android: androidLogo,
  chromeos: chromeosLogo,
  freebsd: freebsdLogo,
  ios: iosLogo,
  ipados: iosLogo,
  linux: linuxLogo,
  macos: macosLogo,
  windows: windowsLogo,
}

export function createFastfetchData(system: SystemInfo, visitor?: Visitor): FastfetchData {
  const identity = `${system.os}@${visitor?.ip_address || 'unknown'}`
  return {
    logo: logos[system.distributionKey] || null,
    identity,
    details: [
      { label: 'OS', value: system.os },
      { label: 'Browser', value: `${system.browserName} ${system.browserVersion}`.trim() },
      { label: 'Device', value: /Mobile|Android|iPhone|iPad/i.test(navigator.userAgent) ? 'Mobile' : 'Desktop' },
      { label: 'IP Address', value: visitor?.ip_address || 'unknown' },
      ...(visitor ? [{ label: 'Visit Time', value: visitor.visit_time }] : []),
      { label: 'Local Time', value: new Date().toLocaleString() },
      { label: 'Language', value: navigator.language || 'unknown' },
      { label: 'Resolution', value: `${window.screen.width}x${window.screen.height}` },
      { label: 'Viewport', value: `${window.innerWidth}x${window.innerHeight}` },
      { label: 'Timezone', value: Intl.DateTimeFormat().resolvedOptions().timeZone || 'unknown' },
      { label: 'Shell', value: 'guestbook (no shell access)' },
      { label: 'Theme', value: 'Twilight Glass' },
    ],
  }
}
