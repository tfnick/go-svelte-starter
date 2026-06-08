module.exports = {
  content: ['./index.html', './src/**/*.{svelte,js}'],
  theme: {
    extend: {}
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: ['light']
  }
};
