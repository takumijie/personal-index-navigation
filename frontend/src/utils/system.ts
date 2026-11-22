import type { SystemInfo } from '@/types'

const distributions: Array<{ key: string; name: string; pattern: RegExp }> = [
  { key: 'arch', name: 'Arch Linux', pattern: /arch linux/i },
  { key: 'ubuntu', name: 'Ubuntu', pattern: /ubuntu/i },
  { key: 'debian', name: 'Debian', pattern: /debian/i },
  { key: 'mint', name: 'Linux Mint', pattern: /linux mint|linuxmint/i },
  { key: 'fedora', name: 'Fedora', pattern: /fedora/i },
  { key: 'manjaro', name: 'Manjaro', pattern: /manjaro/i },
  { key: 'endeavouros', name: 'EndeavourOS', pattern: /endeavouros/i },
  { key: 'opensuse', name: 'openSUSE', pattern: /opensuse/i },
  { key: 'kali', name: 'Kali Linux', pattern: /kali/i },
  { key: 'gentoo', name: 'Gentoo', pattern: /gentoo/i },
  { key: 'alpine', name: 'Alpine Linux', pattern: /alpine/i },
  { key: 'nixos', name: 'NixOS', pattern: /nixos/i },
  { key: 'void', name: 'Void Linux', pattern: /void linux/i },
  { key: 'popos', name: 'Pop!_OS', pattern: /pop!?_os|pop os/i },
  { key: 'elementary', name: 'elementary OS', pattern: /elementary os|elementaryos/i },
  { key: 'zorin', name: 'Zorin OS', pattern: /zorin/i },
  { key: 'rocky', name: 'Rocky Linux', pattern: /rocky linux/i },
  { key: 'almalinux', name: 'AlmaLinux', pattern: /almalinux/i },
  { key: 'centos', name: 'CentOS', pattern: /centos/i },
  { key: 'rhel', name: 'Red Hat Enterprise Linux', pattern: /red hat enterprise|rhel/i },
  { key: 'deepin', name: 'deepin', pattern: /deepin/i },
]

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

  const distro = distributions.find((item) => item.pattern.test(source))
  if (distro) {
    os = distro.name
    distributionKey = distro.key
  } else if (/Android/i.test(source)) {
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
