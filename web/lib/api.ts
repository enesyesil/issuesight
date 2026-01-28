// API Client for IssueSight Gateway

import { Tutorial, TutorialListItem, TutorialListResponse, User, IssueSubmitResponse, QuotaInfo } from './types';

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
    const headers: HeadersInit = {
        'Content-Type': 'application/json',
        ...options?.headers,
    };

    // Add CSRF token for mutation requests
    if (options?.method && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(options.method.toUpperCase())) {
        const csrfToken = getCookie('csrf_token');
        if (csrfToken) {
            (headers as any)['X-CSRF-Token'] = csrfToken;
        }
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        credentials: 'include', // Important for sending cookies (auth and csrf)
        headers,
    });

    if (!response.ok) {
        // Handle 401 Unauthorized (e.g., redirect to login)
        if (response.status === 401) {
            // Optional: window.location.href = '/login';
        }

        const error = await response.json().catch(() => ({ message: 'Request failed' }));
        throw new Error(error.message || `HTTP error ${response.status}`);
    }

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
};

export default api;
