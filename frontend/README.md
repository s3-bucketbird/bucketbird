# BucketBird Frontend

BucketBird's frontend is a React 19 + TypeScript single-page application scaffolded with Vite and styled with Tailwind CSS. It provides a modern S3-compatible storage console interface that integrates with the Go backend.

## Stack

- React 19 with Vite for fast local development
- TypeScript with strict settings and React Router for routing
- Tailwind CSS (v3) with custom design tokens
- @tanstack/react-query for data fetching and caching
- Integrated with Go backend API for real-time bucket and object management
- Dark/light theme support with a custom ThemeProvider and Material Symbols icons

## Getting Started

```bash
cp .env.example .env # adjust VITE_API_URL if your backend runs elsewhere
npm install
npm run dev        # start Vite dev server at http://localhost:5173
npm run build      # type-check and create a production bundle
```

Linting uses ESLint (`npm run lint`).

## Project Layout

```
frontend/
├── src/
│   ├── app/               # Router and global providers
│   ├── api/               # Mock API client + types
│   ├── components/        # Layout, theme, and UI primitives
│   ├── hooks/             # React Query hooks around the API client
│   ├── pages/             # Route-level screens matching the design templates
│   ├── lib/               # Shared utilities (e.g., class name helper)
│   └── index.css          # Tailwind base styles
└── index.html             # Font/material icon setup and theme bootstrapping
```

## Theming

The UI supports dark and light themes, respecting the user's OS preference by default. The `ThemeProvider` handles theme toggling and state management. Users can switch themes via the settings icon in the top bar on desktop or the menu on mobile.

## Features

- **Bucket Management** – Create, view, and manage S3-compatible buckets across multiple storage providers
- **File Operations** – Upload, download, rename, delete, and preview files with drag-and-drop support
- **Folder Navigation** – Browse folder hierarchies with breadcrumb navigation
- **Multi-Provider Support** – Connect to AWS S3, MinIO, and other S3-compatible storage services
- **Credential Management** – Securely store and manage multiple storage credentials
- **User Profile** – Manage account settings, preferences, and password
- **Theme Support** – Switch between light and dark themes
- **Search** – Search for files and folders within buckets
- **File Preview** – Preview images, videos, audio, PDFs, and text files directly in the browser
- **Bulk Operations** – Select and delete multiple files at once
- **Responsive Design** – Works on desktop and mobile devices

## Development

The frontend communicates with the Go backend API via REST endpoints. All API calls are made through the `src/api/client.ts` module, with React Query handling caching and state management through hooks in `src/hooks`.
