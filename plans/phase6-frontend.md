# Phase 6: Frontend Service Implementation

## Status: 📋 PLANNING

This phase implements the Next.js frontend application that provides the user interface for IssueSight.

## Architecture

```
Frontend Layer
├── Next.js 14 (App Router)
│   ├── TypeScript
│   ├── React Components
│   ├── API Client
│   └── State Management
└── Gateway API (Backend)
```

## File Structure

```
web/
├── app/
│   ├── layout.tsx           # Root layout
│   ├── page.tsx             # Home page
│   ├── (auth)/
│   │   ├── login/
│   │   │   └── page.tsx     # Login page
│   │   └── callback/
│   │       └── page.tsx      # OAuth callback
│   ├── (dashboard)/
│   │   ├── issues/
│   │   │   ├── page.tsx      # Issue submission
│   │   │   └── [id]/
│   │   │       └── page.tsx  # Issue detail
│   │   └── tutorials/
│   │       ├── page.tsx      # Tutorial list
│   │       └── [id]/
│   │           └── page.tsx  # Tutorial view
│   └── api/
│       └── proxy/              # API proxy (if needed)
├── components/
│   ├── ui/                   # Reusable UI components
│   │   ├── Button.tsx
│   │   ├── Card.tsx
│   │   ├── Input.tsx
│   │   └── ...
│   ├── issue/
│   │   ├── IssueForm.tsx     # Issue submission form
│   │   ├── IssueCard.tsx     # Issue display card
│   │   └── IssueList.tsx     # Issue list
│   ├── tutorial/
│   │   ├── TutorialViewer.tsx # Tutorial markdown viewer
│   │   ├── TutorialCard.tsx   # Tutorial card
│   │   └── TutorialList.tsx   # Tutorial list
│   └── auth/
│       ├── LoginButton.tsx    # OAuth login button
│       └── UserMenu.tsx       # User menu
├── lib/
│   ├── api.ts                # API client
│   ├── auth.ts               # Auth utilities
│   └── utils.ts              # Utility functions
├── hooks/
│   ├── useAuth.ts            # Auth hook
│   ├── useTutorials.ts       # Tutorials hook
│   └── useIssues.ts          # Issues hook
├── types/
│   ├── api.ts                # API types
│   ├── issue.ts              # Issue types
│   └── tutorial.ts           # Tutorial types
└── styles/
    └── globals.css            # Global styles
```

---

## Implementation Tasks

### 1. Project Setup

**Status:** ✅ Basic setup exists

- Next.js 14 with App Router
- TypeScript configuration
- Basic layout and home page

**Next steps:**

- Install UI library (shadcn/ui, Tailwind CSS, or similar)
- Set up API client
- Configure environment variables

---

### 2. Authentication Flow

Implement OAuth authentication:

- **Login Page** (`app/(auth)/login/page.tsx`)
  - GitHub OAuth button
  - Google OAuth button
  - Redirect to OAuth provider

- **Callback Handler** (`app/(auth)/callback/page.tsx`)
  - Handle OAuth callback
  - Exchange code for session
  - Redirect to dashboard

- **Auth Hook** (`hooks/useAuth.ts`)
  - Check authentication status
  - Get current user
  - Logout functionality

**Edge cases to handle:**

- OAuth callback errors
- Session expiration
- Invalid state parameter
- Network errors during auth

---

### 3. Issue Submission

Implement issue submission flow:

- **Issue Form** (`components/issue/IssueForm.tsx`)
  - URL input field
  - Validation
  - Submit button
  - Loading states
  - Error handling

- **Issue Submission Page** (`app/(dashboard)/issues/page.tsx`)
  - Form component
  - Success/error messages
  - Redirect to issue detail after submission

**API Integration:**

```typescript
POST /api/issues
{
  "url": "https://github.com/owner/repo/issues/123"
}

Response:
{
  "id": "uuid",
  "status": "queued",
  "message": "Issue queued for processing"
}
```

**Edge cases to handle:**

- Invalid URL format
- Duplicate submission
- Quota exceeded
- Network errors
- Processing timeout

---

### 4. Issue Detail Page

Display issue information and processing status:

- **Issue Detail Page** (`app/(dashboard)/issues/[id]/page.tsx`)
  - Issue metadata (title, description, labels)
  - Processing status (queued, processing, completed, failed)
  - Link to tutorial (when available)
  - Loading states

**API Integration:**

```typescript
GET /api/issues/:id
GET /api/tutorials?issue_id=:id
```

---

### 5. Tutorial Viewer

Display AI-generated tutorial content:

- **Tutorial Viewer** (`components/tutorial/TutorialViewer.tsx`)
  - Markdown rendering
  - Syntax highlighting for code blocks
  - Table of contents
  - Print/export functionality

- **Tutorial Page** (`app/(dashboard)/tutorials/[id]/page.tsx`)
  - Tutorial content
  - Related concepts
  - Navigation

**Markdown Rendering:**

- Use `react-markdown` or similar
- Syntax highlighting with `prism.js` or `highlight.js`
- Math rendering (if needed) with `react-katex`

---

### 6. Tutorial List

Display user's tutorials:

- **Tutorial List** (`app/(dashboard)/tutorials/page.tsx`)
  - List of user's tutorials
  - Filtering and sorting
  - Search functionality
  - Pagination

- **Tutorial Card** (`components/tutorial/TutorialCard.tsx`)
  - Preview of tutorial
  - Status indicator
  - Link to full tutorial

**API Integration:**

```typescript
GET /api/tutorials
Response: Tutorial[]
```

---

### 7. API Client

Create API client for backend communication:

- **API Client** (`lib/api.ts`)
  - HTTP client setup (fetch or axios)
  - Request interceptors (add auth token)
  - Response interceptors (handle errors)
  - Type-safe API methods

**Example:**

```typescript
export const api = {
  issues: {
    submit: (url: string) => post('/api/issues', { url }),
    get: (id: string) => get(`/api/issues/${id}`),
  },
  tutorials: {
    list: () => get('/api/tutorials'),
    get: (id: string) => get(`/api/tutorials/${id}`),
  },
  auth: {
    login: (provider: string) => redirect(`/api/auth/${provider}`),
    logout: () => post('/api/auth/logout'),
  },
};
```

---

### 8. State Management

Manage application state:

- **React Hooks** (`hooks/`)
  - `useAuth.ts` - Authentication state
  - `useTutorials.ts` - Tutorials state
  - `useIssues.ts` - Issues state

- **Optional:** Add state management library if needed:
  - Zustand (lightweight)
  - TanStack Query (for server state)
  - Redux (if complex state needed)

---

### 9. UI Components

Build reusable UI components:

- **Base Components** (`components/ui/`)
  - Button, Input, Card, Modal, etc.
  - Use shadcn/ui or build custom

- **Feature Components** (`components/issue/`, `components/tutorial/`)
  - Issue-specific components
  - Tutorial-specific components

---

### 10. Styling

Implement consistent styling:

- **CSS Framework:**
  - Tailwind CSS (recommended)
  - Or CSS Modules
  - Or styled-components

- **Design System:**
  - Color palette
  - Typography
  - Spacing
  - Components

---

## Testing Strategy

### Unit Tests

- Component tests (React Testing Library)
- Hook tests
- Utility function tests
- API client tests (mock fetch)

### Integration Tests

- E2E tests (Playwright or Cypress)
- User flows:
  - Login → Submit issue → View tutorial
  - Browse tutorials
  - Authentication flow

---

## Error Handling

### User-Friendly Error Messages

- Network errors: "Unable to connect. Please check your internet."
- Auth errors: "Authentication failed. Please try again."
- Quota errors: "Daily limit reached. Try again tomorrow."
- Validation errors: Show field-specific errors

### Error Boundaries

- React Error Boundaries for component errors
- Global error handler for unhandled errors
- Error logging to monitoring service

---

## Performance Optimization

- **Code Splitting:** Route-based code splitting
- **Image Optimization:** Next.js Image component
- **Caching:** API response caching
- **Lazy Loading:** Lazy load heavy components
- **Bundle Size:** Monitor and optimize bundle size

---

## Accessibility

- **ARIA Labels:** Proper ARIA attributes
- **Keyboard Navigation:** Full keyboard support
- **Screen Readers:** Test with screen readers
- **Color Contrast:** WCAG AA compliance
- **Focus Management:** Visible focus indicators

---

## Dependencies

```json
{
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.0.0",
    "react-dom": "^18.0.0",
    "react-markdown": "^9.0.0",
    "prismjs": "^1.29.0",
    "zustand": "^4.4.0",
    "@tanstack/react-query": "^5.0.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "@types/react": "^18.0.0",
    "typescript": "^5.0.0",
    "tailwindcss": "^3.0.0",
    "@testing-library/react": "^14.0.0",
    "playwright": "^1.40.0"
  }
}
```

---

## Environment Variables

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_OAUTH_GITHUB_CLIENT_ID=...
NEXT_PUBLIC_OAUTH_GOOGLE_CLIENT_ID=...
```

---

## Next Steps

After Phase 6 completion:
1. Complete IssueSight platform
2. Users can submit issues and view tutorials
3. Full end-to-end flow operational
4. Ready for production deployment

**See:** Deployment documentation in `deployments/` directory.
