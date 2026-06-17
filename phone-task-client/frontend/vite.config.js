import { defineConfig } from 'vite';

export default defineConfig({
  base: './',
  plugins: [{
    name: 'trim-html-trailing-whitespace',
    transformIndexHtml(html) {
      return html.replace(/[ \t]+$/gm, '');
    },
  }],
  build: {
    rollupOptions: {
      output: {
        entryFileNames: 'assets/index.js',
        assetFileNames: 'assets/[name][extname]',
      },
    },
  },
});
