module.exports = {
  content: ['./public/**/*.html', './src/**/*.vue'],
  theme: {
    extend: {},
  },
  variants: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
