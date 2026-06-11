# DaisyUI 5.5.23 Upgrade Research

## Current Project State

* `frontend/package.json` currently uses:
  * `daisyui: ^4.12.24`
  * `tailwindcss: ^3.4.17`
  * `postcss: ^8.5.1`
  * `autoprefixer: ^10.4.20`
* `frontend/package-lock.json` currently resolves:
  * `daisyui: 4.12.24`
  * `tailwindcss: 3.4.19`
* Current integration:
  * `frontend/tailwind.config.cjs` uses `plugins: [require('daisyui')]`.
  * `frontend/tailwind.config.cjs` sets `daisyui.themes = ['light']`.
  * `frontend/postcss.config.cjs` uses `tailwindcss` and `autoprefixer`.
  * `frontend/src/styles.css` uses Tailwind 3 directives: `@tailwind base;`, `@tailwind components;`, `@tailwind utilities;`.

## Target Version Availability

`npm view daisyui@5.5.23 version dist-tags --json` confirms:

```json
{
  "version": "5.5.23",
  "dist-tags": {
    "latest": "5.5.23"
  }
}
```

## Migration Implication

daisyUI 5 is designed for Tailwind CSS 4 style integration. The project should not only bump `daisyui`; it should also move frontend styling integration to the Tailwind CSS 4 / daisyUI 5 pattern:

* Add Tailwind CSS 4 and the Vite plugin package.
* Use the Vite Tailwind plugin instead of the Tailwind 3 PostCSS plugin path.
* Move daisyUI plugin registration to CSS using `@plugin "daisyui"` and keep the `light` theme setting there.
* Remove or retire obsolete Tailwind 3 config/PostCSS wiring if no longer used.

## Relevant Files

* `frontend/package.json`
* `frontend/package-lock.json`
* `frontend/vite.config.js`
* `frontend/src/styles.css`
* `frontend/tailwind.config.cjs`
* `frontend/postcss.config.cjs`

## Validation Needed

* `cd frontend && npm install` or equivalent lockfile update.
* `cd frontend && npm test`.
* `cd frontend && npm run build`.
* Visual/browser smoke check is recommended because many pages rely on daisyUI classes such as `btn`, `card`, `drawer`, `tabs`, `input`, `select`, `toggle`, and `alert`.
