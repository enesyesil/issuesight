'use client';

import { useState, useEffect } from 'react';
import useSWR from 'swr';
import { api } from '@/lib/api';
import { TutorialViewer } from '@/components/tutorial/TutorialViewer';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { ArrowLeft, Clock, Copy, Bookmark, Share2 } from 'lucide-react';
import Link from 'next/link';
import { Badge } from '@/components/ui/badge';
import { Tutorial } from '@/lib/types';

const BOOKMARK_KEY_PREFIX = 'tutorial-bookmark-';

function readTimeMinutes(markdown: string): number {
    const words = (markdown || '').trim().split(/\s+/).filter(Boolean).length;
    return Math.max(1, Math.ceil(words / 200));
}

function formatDate(iso: string): string {
    try {
        return new Date(iso).toLocaleDateString(undefined, {
            month: 'numeric',
            day: 'numeric',
            year: 'numeric',
        });
    } catch {
        return iso;
    }
}

export default function TutorialPage({ params }: { params: { id: string } }) {
    const { data: tutorial, error, isLoading } = useSWR<Tutorial>(
        `/api/tutorials/${params.id}`,
        () => api.tutorials.get(params.id)
    );

    const [copied, setCopied] = useState(false);
    const [bookmarked, setBookmarked] = useState(false);

    useEffect(() => {
        if (typeof window === 'undefined' || !params.id) return;
        const key = BOOKMARK_KEY_PREFIX + params.id;
        setBookmarked(localStorage.getItem(key) === '1');
    }, [params.id]);

    const handleCopyLink = async () => {
        try {
            await navigator.clipboard.writeText(typeof window !== 'undefined' ? window.location.href : '');
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        } catch {
            // ignore
        }
    };

    const handleBookmark = () => {
        if (typeof window === 'undefined' || !params.id) return;
        const key = BOOKMARK_KEY_PREFIX + params.id;
        const next = !bookmarked;
        if (next) localStorage.setItem(key, '1');
        else localStorage.removeItem(key);
        setBookmarked(next);
    };

    const handleShare = async () => {
        if (typeof window === 'undefined' || !tutorial) return;
        if (navigator.share) {
            try {
                await navigator.share({
                    title: tutorial.title,
                    url: window.location.href,
                });
            } catch {
                handleCopyLink();
            }
        } else {
            handleCopyLink();
        }
    };

    if (isLoading) {
        return (
            <div 
                className="p-8 text-center"
                role="status"
                aria-live="polite"
            >
                <span>Loading tutorial...</span>
                <span className="sr-only">Please wait while the tutorial is loading</span>
            </div>
        );
    }

    if (error || !tutorial) {
        return <div className="p-8 text-center text-red-500">Failed to load tutorial</div>;
    }

    const readTime = readTimeMinutes(tutorial.markdown_body || '');

    return (
        <div className="flex flex-col gap-6 max-w-4xl mx-auto w-full">
            <div className="flex items-center gap-4">
                <Link href="/dashboard/tutorials">
                    <Button variant="ghost" size="sm" className="gap-2" aria-label="Back to Dashboard">
                        <ArrowLeft className="h-4 w-4" />
                        Back to Dashboard
                    </Button>
                </Link>
                <div className="flex-1 min-w-0" />
            </div>

            <Card>
                <CardHeader className="gap-3">
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Tutorial</p>
                    <div className="flex flex-wrap items-center gap-2">
                        <h1 className="text-2xl font-bold truncate">{tutorial.title}</h1>
                        <Badge variant={tutorial.status === 'completed' ? 'default' : 'secondary'}>
                            {tutorial.status}
                        </Badge>
                    </div>
                    <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
                        <span>{readTime} min read</span>
                        <span className="flex items-center gap-1">
                            <Clock className="h-3.5 w-3.5" />
                            {formatDate(tutorial.created_at)}
                        </span>
                    </div>
                </CardHeader>
                <CardContent className="pt-0">
                    <div className="flex flex-wrap items-center gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleCopyLink}
                            className="gap-2"
                        >
                            <Copy className="h-4 w-4" />
                            {copied ? 'Copied' : 'Copy Link'}
                        </Button>
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleBookmark}
                            className="gap-2"
                            aria-pressed={bookmarked}
                        >
                            <Bookmark className={`h-4 w-4 ${bookmarked ? 'fill-current' : ''}`} />
                            {bookmarked ? 'Bookmarked' : 'Bookmark'}
                        </Button>
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleShare}
                            className="gap-2"
                        >
                            <Share2 className="h-4 w-4" />
                            Share
                        </Button>
                    </div>
                </CardContent>
            </Card>

            <div className="rounded-2xl border border-border/70 bg-muted/20 p-4">
                <TutorialViewer
                    tutorialId={tutorial.id}
                    content={tutorial.markdown_body || '# No content available'}
                    concepts={tutorial.concepts}
                />
            </div>
        </div>
    );
}
