import { getAsciiArt } from '@/data/ascii'
import type { SystemInfo, Visitor } from '@/types'

export function createFastfetchLines(system: SystemInfo, visitor?: Visitor): string[] {
  const art = getAsciiArt(system.distributionKey).trim().split('\n')
  const details = [
    'visitor@personal-space',
    '---------------------',
    `OS: ${system.os}`,
    `Distribution: ${system.distribution}`,
    `Browser: ${system.browserName} ${system.browserVersion}`.trim(),
    `Device: ${/Mobile|Android|iPhone|iPad/i.test(navigator.userAgent) ? 'Mobile' : 'Desktop'}`,
    ...(visitor ? [`IP Address: ${visitor.ip}`, `Visit Time: ${visitor.visit_time}`] : []),
    `Local Time: ${new Date().toLocaleString()}`, 
    `Language: ${navigator.language || 'unknown'}`,
    `Resolution: ${window.screen.width}x${window.screen.height}`,
    `Viewport: ${window.innerWidth}x${window.innerHeight}`,
    `Timezone: ${Intl.DateTimeFormat().resolvedOptions().timeZone || 'unknown'}`,
    'Shell: guestbook (no shell access)',
    'Theme: Twilight Glass',
  ]
  const lines: string[] = []
  const count = Math.max(art.length, details.length)
  for (let index = 0; index < count; index += 1) {
    lines.push(`${(art[index] || '').padEnd(32)}${details[index] || ''}`.trimEnd())
  }
  return lines
}
