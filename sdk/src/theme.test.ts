import { beforeEach, describe, expect, it } from 'vitest'
import { loadThemePreference, resolveTheme, saveThemePreference } from './theme'

beforeEach(() => localStorage.clear())

describe('theme', () => {
  it('persists per site', () => {
    saveThemePreference('http://api.test', 'site-a', 'dark')
    expect(loadThemePreference('http://api.test', 'site-a')).toBe('dark')
    expect(loadThemePreference('http://api.test', 'site-b')).toBeNull()
  })

  it('resolves explicit modes', () => {
    expect(resolveTheme('light')).toBe('light')
    expect(resolveTheme('dark')).toBe('dark')
  })
})
