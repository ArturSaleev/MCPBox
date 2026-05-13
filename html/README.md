# MCPBox Admin UI

This directory contains the React + Vite source for the embedded MCPBox admin UI.

## Development

```bash
npm install
npm run dev
```

The Vite dev server runs the UI separately for frontend work.

## Production Build

```bash
npm run build
```

The production build is written to `../internal/httpapi/ui/dist` and embedded into the Go binary.
