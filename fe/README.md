# Jobber Frontend

Job Application Tracking Platform - Frontend Application

## Tech Stack

- **React 18** - UI framework
- **TypeScript** - Type safety (strict mode)
- **Vite** - Build tool
- **Tailwind CSS** - Styling
- **shadcn/ui** - UI components
- **React Router v6** - Routing
- **TanStack Query** - Server state management
- **Zustand** - Client state management
- **i18next** - Internationalization
- **ky** - HTTP client

## Project Structure

```
src/
├── app/            # App bootstrap & routing
│   ├── layouts/    # Layout components
│   ├── providers.tsx
│   └── router.tsx
├── pages/          # Route-level pages
├── features/       # Business features
│   ├── applications/
│   └── resumes/
├── entities/       # Domain entities (types & helpers)
├── widgets/        # Reusable composed UI blocks
│   ├── Sidebar.tsx
│   └── Header.tsx
├── shared/         # Shared utilities & components
│   ├── ui/         # UI components (shadcn/ui)
│   ├── lib/        # Utilities
│   ├── locales/    # Translation files
│   ├── types/      # TypeScript types
│   └── constants/
├── services/       # API clients
└── stores/         # Zustand stores
```

## Setup

1. Install dependencies:
```bash
npm install
```

2. Create `.env` file:
```bash
cp .env.example .env
```

3. Update the API URL in `.env`:
```
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

4. Start the development server:
```bash
npm run dev
```

The app will be available at `http://localhost:3000`

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint

## Features

### Authentication
- Login / Register
- Access token in memory
- Refresh token in httpOnly cookie
- Silent refresh flow
- Protected routes

### Core Features
- **Applications** - Track job applications with stages
- **Resumes** - Manage multiple resume versions
- **Companies** - Store company information
- **Jobs** - Track job postings
- **Settings** - Theme, language, account settings

### UI/UX
- Mobile-first responsive design
- Light / Dark theme
- i18n support (English)
- Modal-first interactions
- Sidebar navigation (collapsible)
- Skeleton loading states
- Empty states
- Error handling

## Architecture Principles

1. **Backend as Source of Truth**
   - Frontend reflects backend state
   - No business logic in frontend
   - No state inference

2. **Server State Management**
   - All API data via React Query
   - No manual refetch
   - Proper cache invalidation

3. **Client State Management**
   - Zustand for UI state only
   - Theme, sidebar, language, modals

4. **Type Safety**
   - TypeScript strict mode
   - Types generated from OpenAPI spec
   - Typed API responses

## API Integration

API client is configured in `src/services/api.ts` with:
- Automatic authorization headers
- Request ID propagation
- Error mapping
- Token refresh handling

## Contributing

Follow the project structure and architectural principles strictly. This is a long-living product, not a demo.
