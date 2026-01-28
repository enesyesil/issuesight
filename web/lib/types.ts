// API Types for IssueSight Frontend

export interface User {
    id: string;
    display_name: string;
    email: string;
    avatar_url: string;
}

export interface AuthenticatedUser extends User { }

export interface Tutorial {
    id: string;
    title: string;
    markdown_body: string;
    status: string; // 'pending' | 'completed' | 'failed'
    created_at: string;
    updated_at: string;
}

export interface TutorialListItem extends Tutorial { }

export interface TutorialListResponse {
    tutorials: Tutorial[];
    count: number;
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

export interface QuotaInfo {
    used: number;
    limit: number;
    resets_at: string;
}
