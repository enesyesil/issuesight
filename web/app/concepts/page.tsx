'use client';

import Link from 'next/link';
import useSWR from 'swr';
import { api } from '@/lib/api';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ConceptListResponse } from '@/lib/types';

export default function ConceptsPage() {
    const { data, error, isLoading } = useSWR<ConceptListResponse>(
        '/api/concepts',
        () => api.concepts.list()
    );

    const concepts = data?.concepts ?? [];

    if (isLoading) {
        return (
            <div
                className="p-8 text-center bg-muted/20 rounded-lg animate-pulse h-64 flex items-center justify-center"
                role="status"
                aria-live="polite"
            >
                <span>Loading concepts...</span>
                <span className="sr-only">Please wait while concepts are loading</span>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-8 text-center text-destructive bg-destructive/10 rounded-lg border border-destructive/20">
                Failed to load concepts. Please try again.
            </div>
        );
    }

    if (!concepts.length) {
        return (
            <div className="flex flex-col gap-6">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Concepts</h1>
                    <p className="text-muted-foreground">
                        Discover core concepts across open-source projects.
                    </p>
                </div>
                <div className="flex flex-col items-center justify-center min-h-[400px] border rounded-lg border-dashed p-8 text-center animate-in fade-in-50">
                    <div className="mx-auto flex max-w-[420px] flex-col items-center justify-center text-center">
                        <h3 className="mt-4 text-lg font-semibold">No concepts yet</h3>
                        <p className="mb-4 mt-2 text-sm text-muted-foreground">
                            Concepts will appear here once projects are analyzed.
                        </p>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="flex flex-col gap-6">
            <div className="flex flex-wrap items-end justify-between gap-4">
                <div>
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Catalog</p>
                    <h1 className="text-3xl font-bold tracking-tight">Concepts</h1>
                    <p className="text-muted-foreground">
                        Discover core concepts across open-source projects.
                    </p>
                </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {concepts.map((concept) => (
                    <Link key={concept.id} href={`/concepts/${concept.slug}`} className="group">
                        <Card className="h-full transition-all duration-200 group-hover:-translate-y-0.5 group-hover:shadow-md flex flex-col">
                            <CardHeader>
                                <div className="flex items-start justify-between gap-2">
                                    <CardTitle className="line-clamp-2 text-lg">
                                        {concept.name || concept.slug}
                                    </CardTitle>
                                    {concept.category && (
                                        <Badge variant="secondary">{concept.category}</Badge>
                                    )}
                                </div>
                                <CardDescription className="line-clamp-3">
                                    {concept.description || 'No description'}
                                </CardDescription>
                            </CardHeader>
                        </Card>
                    </Link>
                ))}
            </div>
        </div>
    );
}
