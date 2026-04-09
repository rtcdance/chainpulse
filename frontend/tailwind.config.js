/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        glow: '#f4a261',
        sand: '#f1e8d8',
        mist: '#a89f91',
        ink: '#07111f',
      },
      backgroundImage: {
        grid: 'linear-gradient(rgba(255,255,255,0.06) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.06) 1px, transparent 1px)',
      },
      backgroundSize: {
        grid: '42px 42px',
      },
      fontFamily: {
        sans: ['"Avenir Next"', '"Segoe UI"', '"Helvetica Neue"', 'sans-serif'],
        display: ['"Avenir Next Condensed"', '"Trebuchet MS"', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
