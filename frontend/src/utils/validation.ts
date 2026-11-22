export function validWebUrl(value: string, allowEmpty = false): boolean {
  if (!value && allowEmpty) return true
  if (value.startsWith('/')) return true
  try {
    const url = new URL(value)
    return ['http:', 'https:'].includes(url.protocol)
  } catch {
    return false
  }
}

export function validNavigationUrl(value: string): boolean {
  if (value.startsWith('mailto:')) return /^mailto:[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
  try {
    const url = new URL(value)
    return ['http:', 'https:'].includes(url.protocol)
  } catch {
    return false
  }
}

export function validTimezone(value: string): boolean {
  try {
    new Intl.DateTimeFormat('zh-CN', { timeZone: value }).format()
    return true
  } catch {
    return false
  }
}

export function validClockTime(value: string): boolean {
  return /^([01]\d|2[0-3]):[0-5]\d$/.test(value)
}
