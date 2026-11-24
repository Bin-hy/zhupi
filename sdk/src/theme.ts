import type { ThemeMode } from './state'

const prefKey = (apiBase: string, siteKey: string) => `annotation-sdk-theme:${apiBase}:${siteKey}`

export function loadThemePreference(apiBase: string, siteKey: string): ThemeMode | null {
  try {
    const raw = localStorage.getItem(prefKey(apiBase, siteKey))
    if (raw === 'auto' || raw === 'light' || raw === 'dark') return raw
  } catch {
    /* ignore */
  }
  return null
}

export function saveThemePreference(apiBase: string, siteKey: string, theme: ThemeMode): void {
  try {
    localStorage.setItem(prefKey(apiBase, siteKey), theme)
  } catch {
    /* ignore */
  }
}

export function systemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined' || !window.matchMedia) return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function resolveTheme(mode: ThemeMode): 'light' | 'dark' {
  return mode === 'auto' ? systemTheme() : mode
}

export function watchSystemTheme(callback: (theme: 'light' | 'dark') => void): () => void {
  if (typeof window === 'undefined' || !window.matchMedia) return () => {}
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const handler = () => callback(media.matches ? 'dark' : 'light')
  if (media.addEventListener) {
    media.addEventListener('change', handler)
    return () => media.removeEventListener('change', handler)
  }
  media.addListener(handler)
  return () => media.removeListener(handler)
}
