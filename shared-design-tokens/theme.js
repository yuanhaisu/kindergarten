/**
 * theme.js — Runtime theme switching via CSS Custom Properties
 *
 * Usage:
 *   import { applyTheme, resetTheme } from 'kindergarten-design-tokens/theme'
 *
 *   // Switch to a kindergarten-specific brand color
 *   applyTheme({
 *     '--color-brand-primary': '#FF6B35',
 *     '--color-brand-primary-light': '#FF8C5A',
 *     '--color-brand-primary-dark': '#E55A2B',
 *     '--color-brand-primary-bg': '#FFF5F0',
 *     '--color-bg-page': '#FFFBF5',
 *     '--color-bg-card': '#FFFFFF',
 *   })
 *
 *   // Reset to defaults
 *   resetTheme()
 */

const ROOT_SELECTOR = 'page'

/**
 * Apply a theme by setting CSS custom properties on the root element.
 * Only the provided properties are changed — all others keep their defaults.
 *
 * @param {Record<string, string>} vars — CSS custom properties to set, e.g. { '--color-brand-primary': '#ff0000' }
 */
export function applyTheme(vars) {
  const root = document.querySelector(ROOT_SELECTOR) || document.documentElement
  for (const [prop, value] of Object.entries(vars)) {
    root.style.setProperty(prop, value)
  }
}

/**
 * Reset all theme custom properties to their default (inherited from :root).
 * @param {string[]} props — specific properties to reset, or omit to reset nothing (removes inline styles)
 */
export function resetTheme(props) {
  const root = document.querySelector(ROOT_SELECTOR) || document.documentElement
  if (props && props.length) {
    for (const prop of props) {
      root.style.removeProperty(prop)
    }
  } else {
    // Remove all inline style overrides
    root.removeAttribute('style')
  }
}

/**
 * Build a theme object from a kindergarten brand config.
 * This is a convenience helper — you can also use applyTheme() directly.
 *
 * @param {Object} brand
 * @param {string} brand.primary — primary brand color (hex)
 * @param {string} brand.primaryLight — lighter brand color
 * @param {string} brand.primaryDark — darker brand color
 * @param {string} brand.primaryBg — brand background color
 * @param {string} [brand.bgPage] — page background
 * @param {string} [brand.bgCard] — card background
 * @returns {Record<string, string>}
 */
export function brandToTheme(brand) {
  return {
    '--color-brand-primary': brand.primary,
    '--color-brand-primary-light': brand.primaryLight,
    '--color-brand-primary-dark': brand.primaryDark,
    '--color-brand-primary-bg': brand.primaryBg,
    ...(brand.bgPage && { '--color-bg-page': brand.bgPage }),
    ...(brand.bgCard && { '--color-bg-card': brand.bgCard }),
  }
}

/**
 * Example kindergarten brand presets — extend this as you add more kindergartens.
 */
export const BRANDS = {
  default: {
    primary: '#ff8a3d',
    primaryLight: '#ffa866',
    primaryDark: '#e6762a',
    primaryBg: '#fff7f0',
  },
  sunshine: {
    primary: '#FFB800',
    primaryLight: '#FFD54A',
    primaryDark: '#E6A600',
    primaryBg: '#FFF8E1',
  },
  forest: {
    primary: '#4CAF50',
    primaryLight: '#81C784',
    primaryDark: '#388E3C',
    primaryBg: '#E8F5E9',
  },
  ocean: {
    primary: '#2196F3',
    primaryLight: '#64B5F6',
    primaryDark: '#1976D2',
    primaryBg: '#E3F2FD',
  },
}
