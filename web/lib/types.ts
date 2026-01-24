// API Types for IssueSight Frontend

export interface User {
    id: string;
    name: string;
    email: string;
    avatar_url: string;
    plan: 'free' | 'pro';
}

export interface Concept {
    slug: string;        // e.g., "message-queues"
    name: string;        // e.g., "Message Queues"
    description: string; // Brief explanation
}

export interface Tutorial {
    id: string;
    title: string;
    repo: string;
    issue_number: number;
    status: 'PENDING' | 'COMPLETED' | 'FAILED';
    markdown_body: string;
    concepts: Concept[];
    created_at: string;
    read_time?: number;
}

export interface TutorialListItem {
    id: string;
    title: string;
    repo: string;
    issue_number: number;
    status: 'PENDING' | 'COMPLETED' | 'FAILED';
    created_at: string;
    description?: string;
}

export interface QuotaInfo {
    used: number;
    limit: number;
    resets_at: string;
}

export interface IssueSubmitRequest {
    url: string;
}

export interface IssueSubmitResponse {
    status: string;
    message: string;
}

export interface ApiError {
    error: string;
    message: string;
}
