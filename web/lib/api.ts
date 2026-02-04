// API Client for IssueSight Gateway

import { Tutorial, TutorialListItem, TutorialListResponse, User, IssueSubmitResponse, QuotaInfo, Concept, ConceptListResponse } from './types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Helper to get cookie by name
function getCookie(name: string): string | null {
    if (typeof document === 'undefined') return null;
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()?.split(';').shift() || null;
    return null;
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
    };

    // Merge any additional headers
    if (options?.headers) {
        const additionalHeaders = options.headers as Record<string, string>;
        Object.assign(headers, additionalHeaders);
    }

    // Add CSRF token for mutation requests
    if (options?.method && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(options.method.toUpperCase())) {
        const csrfToken = getCookie('csrf_token');
        if (csrfToken) {
            headers['X-CSRF-Token'] = csrfToken;
        }
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        credentials: 'include', // Important for sending cookies (auth and csrf)
        headers,
    });

    if (!response.ok) {
        // Handle 401 Unauthorized - redirect to login (client-side only)
        if (response.status === 401 && typeof window !== 'undefined') {
            // Avoid redirect loop if already on login page
            if (!window.location.pathname.startsWith('/login')) {
                window.location.href = '/login';
            }
        }

        const error = await response.json().catch(() => ({ message: 'Request failed' }));
        throw new Error(error.message || `HTTP error ${response.status}`);
    }

    return response.json();
}

async function fetchApiOptional<T>(endpoint: string, options?: RequestInit): Promise<T | null> {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
    };
    if (options?.headers) {
        const additionalHeaders = options.headers as Record<string, string>;
        Object.assign(headers, additionalHeaders);
    }
    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        credentials: 'include',
        headers,
    });
    if (response.status === 404 || !response.ok) return null;
    return response.json();
}

export const api = {
    issues: {
        submit: (url: string): Promise<IssueSubmitResponse> =>
            fetchApi('/api/issues', {
                method: 'POST',
                body: JSON.stringify({ url }),
            }),
    },

    tutorials: {
        list: (): Promise<TutorialListResponse> =>
            fetchApi('/api/tutorials'),

        get: (id: string): Promise<Tutorial> =>
            fetchApi(`/api/tutorials/${id}`),
    },

    auth: {
        me: (): Promise<User> =>
            fetchApi('/api/auth/me'),

        logout: (): Promise<void> =>
            fetchApi('/api/auth/logout', { method: 'POST' }),

        github: (): string =>
            `${API_BASE}/api/auth/github`,

        google: (): string =>
            `${API_BASE}/api/auth/google`,
    },

    quota: {
        get: (): Promise<QuotaInfo> =>
            fetchApi('/api/quota'),
    },

    concepts: {
        list: async (): Promise<ConceptListResponse> => {
            try {
                const res = await fetch(`${API_BASE}/api/concepts`, { credentials: 'include' });
                if (res.status === 404 || !res.ok) return { concepts: [], count: 0 };
                return res.json();
            } catch {
                return { concepts: [], count: 0 };
            }
        },
        get: (slug: string): Promise<Concept | null> =>
            fetchApiOptional(`/api/concepts/${encodeURIComponent(slug)}`),
    },
};

export default api;
