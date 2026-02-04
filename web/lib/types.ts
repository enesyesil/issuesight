// API Types for IssueSight Frontend

export interface User {
    id: string;
    display_name: string;
    email: string;
    avatar_url: string;
}

export interface AuthenticatedUser extends User { }

export type TutorialStatus = 'pending' | 'processing' | 'completed' | 'failed';

export interface Tutorial {
    id: string;
    title: string;
    markdown_body: string;
    status: TutorialStatus;
    created_at: string;
    updated_at: string;
    concepts?: Concept[];
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
    remaining: number;
    limit: number;
    resets_at: string;
    is_exceeded: boolean;
}

// Concepts (interface-defined for future backend)
export interface Concept {
    id: string;
    slug: string;
    name: string;
    description?: string;
    category?: string;
    tutorial_markdown?: string;
    created_at?: string;
}

export interface ConceptListResponse {
    concepts: Concept[];
    count: number;
}
