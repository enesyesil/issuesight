// API Client for IssueSight Gateway

import { Tutorial, TutorialListItem, User, IssueSubmitResponse, QuotaInfo } from './types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers,
        },
    });

    if (!response.ok) {
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
        list: (): Promise<TutorialListItem[]> =>
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
