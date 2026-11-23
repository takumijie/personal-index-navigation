const refreshSafetyMS = 60_000

export function playbackNeedsRefresh(expiresAt: string, now = Date.now()): boolean {
  if (!expiresAt) return false
  const timestamp = Date.parse(expiresAt)
  return !Number.isFinite(timestamp) || timestamp - now <= refreshSafetyMS
}
