module.exports = {
  content: ['./public/**/*.html', './src/**/*.vue'],
  theme: {
    extend: {
      gridTemplateColumns: {
        'labin': 'min-content auto' // label-input
      }
    },
  },
  variants: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
