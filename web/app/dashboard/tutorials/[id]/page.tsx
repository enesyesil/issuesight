'use client';

import useSWR from 'swr';
import { api } from '@/lib/api';
import { TutorialViewer } from '@/components/tutorial/TutorialViewer';
import { Button } from '@/components/ui/button';
import { ArrowLeft, ExternalLink } from 'lucide-react';
import Link from 'next/link';
import { Badge } from '@/components/ui/badge';
import { Tutorial } from '@/lib/types';

export default function TutorialPage({ params }: { params: { id: string } }) {
    const { data: tutorial, error, isLoading } = useSWR<Tutorial>(
        `/api/tutorials/${params.id}`,
        () => api.tutorials.get(params.id)
    );

    if (isLoading) {
        return <div className="p-8 text-center">Loading tutorial...</div>;
    }

    if (error || !tutorial) {
        return <div className="p-8 text-center text-red-500">Failed to load tutorial</div>;
    }

    return (
        <div className="flex flex-col gap-6 max-w-4xl mx-auto w-full">
            <div className="flex items-center gap-4">
                <Link href="/dashboard/tutorials">
                    <Button variant="ghost" size="icon">
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                </Link>
                <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                        <h1 className="text-2xl font-bold truncate">{tutorial.title}</h1>
                        <Badge variant={tutorial.status === 'completed' ? 'default' : 'secondary'}>
                            {tutorial.status}
                        </Badge>
                    </div>
                </div>
            </div>

            <TutorialViewer content={tutorial.markdown_body || '# No content available'} />
        </div>
    );
}
