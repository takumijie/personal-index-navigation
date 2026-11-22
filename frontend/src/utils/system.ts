import type { SystemInfo } from '@/types'

function browserInfo(ua: string): { name: string; version: string } {
  const matchers: Array<[string, RegExp]> = [
    ['Edge', /Edg(?:e|A|iOS)?\/([\d.]+)/],
    ['Opera', /(?:OPR|Opera)\/([\d.]+)/],
    ['Firefox', /(?:Firefox|FxiOS)\/([\d.]+)/],
    ['Chrome', /(?:Chrome|CriOS)\/([\d.]+)/],
    ['Safari', /Version\/([\d.]+).*Safari/],
  ]
  for (const [name, pattern] of matchers) {
    const match = ua.match(pattern)
    if (match) return { name, version: match[1] || '' }
  }
  return { name: 'Unknown Browser', version: '' }
}

export async function detectSystem(): Promise<SystemInfo> {
  const ua = navigator.userAgent
  const browser = browserInfo(ua)
  let platform = ''
  try {
    const hints = await navigator.userAgentData?.getHighEntropyValues?.(['platform'])
    platform = hints?.platform || navigator.userAgentData?.platform || ''
  } catch {
    platform = navigator.userAgentData?.platform || ''
  }
  const source = `${ua} ${platform}`
  let os = 'Unknown OS'
  let distributionKey = 'unknown'

  if (/Android/i.test(source)) {
    const version = ua.match(/Android\s([\d.]+)/i)?.[1] || ''
    os = `Android ${version}`.trim()
    distributionKey = 'android'
  } else if (/iPad/i.test(ua)) {
    const version = ua.match(/OS\s([\d_]+)/i)?.[1]?.replaceAll('_', '.') || ''
    os = `iPadOS ${version}`.trim()
    distributionKey = 'ipados'
  } else if (/iPhone|iPod/i.test(ua)) {
    const version = ua.match(/OS\s([\d_]+)/i)?.[1]?.replaceAll('_', '.') || ''
    os = `iOS ${version}`.trim()
    distributionKey = 'ios'
  } else if (/CrOS/i.test(source)) {
    os = 'ChromeOS'
    distributionKey = 'chromeos'
  } else if (/Windows/i.test(source)) {
    os = 'Windows'
    distributionKey = 'windows'
  } else if (/Mac OS X|macOS/i.test(source)) {
    const version = ua.match(/Mac OS X\s([\d_]+)/i)?.[1]?.replaceAll('_', '.') || ''
    os = `macOS ${version}`.trim()
    distributionKey = 'macos'
  } else if (/FreeBSD/i.test(source)) {
    os = 'FreeBSD'
    distributionKey = 'freebsd'
  } else if (/Linux/i.test(source)) {
    os = 'Linux'
    distributionKey = 'linux'
  }

  return {
    os,
    distribution: distributionKey,
    distributionKey,
    browserName: browser.name,
    browserVersion: browser.version,
  }
}
