export const DEFAULT_FAVICON = '/favicon.svg'

export function updateFavicon(url: string) {
  let link = document.querySelector<HTMLLinkElement>('link[rel~="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.setAttribute('href', url || DEFAULT_FAVICON)
}
