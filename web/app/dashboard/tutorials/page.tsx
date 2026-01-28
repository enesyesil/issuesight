'use client';

import Link from 'next/link';
import useSWR from 'swr';
import { api } from '@/lib/api';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Calendar } from 'lucide-react';
import { TutorialListResponse } from '@/lib/types';

export default function TutorialsPage() {
    const { data, error, isLoading } = useSWR<TutorialListResponse>('/api/tutorials', () => api.tutorials.list());

    // api.tutorials.list returns Promise<TutorialListResponse> now (I need to check api.ts)
    // Wait, api.ts might interpret the response.
    // If api.ts returns `any` or `TutorialListItem[]`, I need to fix api.ts too.
    // Assuming api.ts returns the JSON response directly.

    const tutorials = data?.tutorials;

    if (isLoading) {
        return <div className="p-8 text-center bg-muted/20 rounded-lg animate-pulse h-64 flex items-center justify-center">Loading tutorials...</div>;
    }

    if (error) {
        return (
            <div className="p-8 text-center text-destructive bg-destructive/10 rounded-lg border border-destructive/20">
                Failed to load tutorials. Please try again.
            </div>
        );
    }

    if (!tutorials || tutorials.length === 0) {
        return (
            <div className="flex flex-col items-center justify-center min-h-[400px] border rounded-lg border-dashed p-8 text-center animate-in fade-in-50">
                <div className="mx-auto flex max-w-[420px] flex-col items-center justify-center text-center">
                    <h3 className="mt-4 text-lg font-semibold">No tutorials created yet</h3>
                    <p className="mb-4 mt-2 text-sm text-muted-foreground">
                        Submit an issue to generate your first tutorial.
                    </p>
                    <Link
                        href="/dashboard/issues"
                        className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground shadow hover:bg-primary/90 h-9 px-4 py-2"
                    >
                        Create Tutorial
                    </Link>
                </div>
            </div>
        );
    }

    return (
        <div className="flex flex-col gap-6">
            <div>
                <h1 className="text-3xl font-bold tracking-tight">Tutorials</h1>
                <p className="text-muted-foreground">
                    Your library of generated tutorials.
                </p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {tutorials.map((tutorial) => (
                    <Link key={tutorial.id} href={`/dashboard/tutorials/${tutorial.id}`}>
                        <Card className="h-full hover:bg-muted/50 transition-colors flex flex-col">
                            <CardHeader>
                                <div className="flex justify-between items-start gap-2">
                                    <CardTitle className="line-clamp-2 text-lg">
                                        {tutorial.title || 'Untitled Tutorial'}
                                    </CardTitle>
                                    <Badge variant={tutorial.status === 'completed' ? 'default' : 'secondary'}>
                                        {tutorial.status}
                                    </Badge>
                                </div>
                                <CardDescription className="line-clamp-3">
                                    {/* Backend doesn't return issue title, so we show date/id or just date */}
                                    Created {new Date(tutorial.created_at).toLocaleDateString()}
                                </CardDescription>
                            </CardHeader>
                            <CardContent className="mt-auto pt-0">
                                <div className="flex items-center text-xs text-muted-foreground gap-2">
                                    <Calendar className="h-3 w-3" />
                                    <span>{new Date(tutorial.created_at).toLocaleTimeString()}</span>
                                </div>
                            </CardContent>
                        </Card>
                    </Link>
                ))}
            </div>
        </div>
    );
}
