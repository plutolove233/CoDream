import type { Config } from 'tailwindcss'

export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'deep-blue': {
          50: '#f0f4ff',
          100: '#e0e9ff',
          200: '#c7d7fe',
          300: '#a5bbfc',
          400: '#8199f8',
          500: '#6477f0',
          600: '#4f5ae4',
          700: '#4148c9',
          800: '#373da3',
          900: '#1e2875',
          950: '#0f172a',
        },
        'ocean-blue': {
          50: '#f0f9ff',
          100: '#e0f2fe',
          200: '#b9e5fe',
          300: '#7dd0fc',
          400: '#38b9f8',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
          800: '#075985',
          900: '#0c4a6e',
          950: '#082f49',
        },
      },
      fontFamily: {
        'display': ['Satoshi', 'system-ui', 'sans-serif'],
        'sans': ['Geist', 'Inter', 'system-ui', 'sans-serif'],
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-mesh': 'radial-gradient(at 40% 20%, hsla(220, 70%, 50%, 0.15) 0px, transparent 50%), radial-gradient(at 80% 0%, hsla(210, 80%, 60%, 0.15) 0px, transparent 50%), radial-gradient(at 0% 50%, hsla(230, 60%, 40%, 0.15) 0px, transparent 50%)',
      },
      boxShadow: {
        'soft': '0 2px 15px -3px rgba(0, 0, 0, 0.07), 0 10px 20px -2px rgba(0, 0, 0, 0.04)',
        'glow': '0 0 20px rgba(100, 119, 240, 0.3)',
      },
    },
  },
  plugins: [],
} satisfies Config
