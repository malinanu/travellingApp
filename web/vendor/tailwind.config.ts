import type { Config } from 'tailwindcss';

const config: Config = {
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    '../packages/charts/src/**/*.tsx',
  ],
  theme: {
    extend: {
      colors: {
        cream: 'var(--cream)',
        sand: 'var(--sand)',
        ink: 'var(--ink)',
        muted: 'var(--muted)',
        line: 'var(--line)',
        accent: 'var(--accent)',
        'accent-tint': 'var(--accent-tint)',
        sun: 'var(--sun)',
        coral: 'var(--coral)',
      },
      fontFamily: {
        sans: ['var(--font-nunito)', 'ui-rounded', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        xl2: '18px',
      },
    },
  },
  plugins: [],
};
export default config;
