/**
 * Theme configuration - Design tokens for consistent styling
 */

export const theme = {
  colors: {
    primary: '#1a8fee',
    primaryDark: '#1472de',
    primaryLight: '#3898f8',
    primary50: '#f0f9ff',
    primary100: '#e0f2fe',
  },
  borderRadius: {
    sm: '0.375rem',    // 6px
    md: '0.5rem',      // 8px
    lg: '0.75rem',     // 12px
    xl: '1rem',        // 16px
    '2xl': '1.5rem',   // 24px
    full: '9999px',
  },
  spacing: {
    1: '0.25rem',     // 4px
    2: '0.5rem',      // 8px
    3: '0.75rem',     // 12px
    4: '1rem',        // 16px
    5: '1.25rem',     // 20px
    6: '1.5rem',      // 24px
    8: '2rem',        // 32px
    10: '2.5rem',     // 40px
    12: '3rem',       // 48px
    16: '4rem',       // 64px
  },
  transitions: {
    fast: '150ms ease',
    normal: '300ms ease',
    slow: '500ms ease',
  },
};

export const lightTheme = {
  background: '#ffffff',
  foreground: '#0f172a',
  cardBg: '#ffffff',
  cardBorder: '#e2e8f0',
  muted: '#f1f5f9',
  mutedForeground: '#64748b',
};

export const darkTheme = {
  background: '#0f172a',
  foreground: '#f8fafc',
  cardBg: '#1e293b',
  cardBorder: '#334155',
  muted: '#1e293b',
  mutedForeground: '#94a3b8',
};

export type Theme = typeof theme;