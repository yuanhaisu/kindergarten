import type { Config } from 'tailwindcss'
import tokens from '../shared-design-tokens/build/ts/tokens-admin'

export default {
  content: [
    './components/**/*.{vue,js,ts}',
    './layouts/**/*.vue',
    './pages/**/*.vue',
    './plugins/**/*.{js,ts}',
    './app.vue',
  ],
  theme: {
    extend: {
      colors: {
        // Brand & semantic colors from shared tokens
        ...tokens.colors,
        // Preserve sidebar shorthand (token has separate sidebar/sidebarHover keys)
        sidebar: {
          DEFAULT: tokens.colors.brand.sidebar,
          hover: tokens.colors.brand.sidebarHover,
        },
        // Preserve primary palette shorthand for backward compat (bg-primary-50 etc.)
        primary: {
          50: tokens.colors.brand.primaryBg,
          100: tokens.colors.brand.primaryBg,
          500: tokens.colors.brand.primary,
          600: tokens.colors.brand.primaryDark,
          700: tokens.colors.brand.primaryDark,
        },
      },
      borderRadius: tokens.borderRadius,
      boxShadow: tokens.boxShadow,
      fontFamily: {
        sans: tokens.fontFamily.sans,
      },
      fontSize: tokens.fontSize,
      fontWeight: tokens.fontWeight,
      spacing: tokens.spacing,
    },
  },
  plugins: [],
} satisfies Config
