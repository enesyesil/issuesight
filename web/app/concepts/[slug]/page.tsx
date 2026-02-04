'use client';

import Link from 'next/link';
import useSWR from 'swr';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { ArrowLeft } from 'lucide-react';
import { Concept } from '@/lib/types';
import { TutorialViewer } from '@/components/tutorial/TutorialViewer';
import { Badge } from '@/components/ui/badge';

export default function ConceptDetailPage({ params }: { params: { slug: string } }) {
    const { slug } = params;
    const { data: concept, error, isLoading } = useSWR<Concept | null>(
        slug ? `/api/concepts/${slug}` : null,
        () => (slug ? api.concepts.get(slug) : Promise.resolve(null)),
        { revalidateOnFocus: false }
    );

    if (isLoading) {
        return (
            <div
                className="p-8 text-center"
                role="status"
                aria-live="polite"
            >
                <span>Loading concept...</span>
                <span className="sr-only">Please wait while the concept is loading</span>
            </div>
        );
    }

    if (error || concept == null) {
        return (
            <div className="flex flex-col gap-6 max-w-4xl mx-auto w-full">
                <div className="flex items-center gap-4">
                    <Link href="/concepts">
                        <Button variant="ghost" size="sm" className="gap-2" aria-label="Back to Concepts">
                            <ArrowLeft className="h-4 w-4" />
                            Back to Concepts
                        </Button>
                    </Link>
                </div>
                <div className="flex flex-col items-center justify-center min-h-[300px] border rounded-lg border-dashed p-8 text-center">
                    <h3 className="text-lg font-semibold">Concept not found</h3>
                    <p className="mt-2 text-sm text-muted-foreground">
                        This concept does not exist or is not yet available.
                    </p>
                    <Link href="/concepts" className="mt-4">
                        <Button variant="outline">Back to Concepts</Button>
                    </Link>
                </div>
            </div>
        );
    }

    return (
        <div className="flex flex-col gap-6 max-w-4xl mx-auto w-full">
            <div className="flex items-center gap-4">
                <Link href="/concepts">
                    <Button variant="ghost" size="sm" className="gap-2" aria-label="Back to Concepts">
                        <ArrowLeft className="h-4 w-4" />
                        Back to Concepts
                    </Button>
                </Link>
                <div className="flex-1 min-w-0" />
            </div>

            <div className="flex flex-col gap-2">
                <div className="flex flex-wrap items-center gap-2">
                    <h1 className="text-2xl font-bold">{concept.name || concept.slug}</h1>
                    {concept.category && <Badge variant="secondary">{concept.category}</Badge>}
                </div>
                {concept.description && (
                    <p className="text-muted-foreground whitespace-pre-wrap">{concept.description}</p>
                )}
            </div>

            <div className="rounded-lg border bg-card p-6">
                <TutorialViewer
                    content={
                        concept.tutorial_markdown ||
                        `# ${concept.name || concept.slug} — Step-by-step\\n\\n## Step 1 — Define it\\n**Goal:** Explain it in one sentence.\\n**Why:** Clarity prevents mistakes.\\n**What to do:** Rewrite the description in your own words.\\n**Checkpoint:** You can explain it without notes.\\n\\n## Step 2 — Find an example\\n**Goal:** Locate a real example.\\n**Why:** Examples make it stick.\\n**What to do:** Search for references and summarize one example.\\n**Checkpoint:** You can point to where it’s used.\\n\\n## Step 3 — Apply it once\\n**Goal:** Use the concept in a small change.\\n**Why:** Practice builds confidence.\\n**What to do:** Pick a tiny update and note how the concept guides it.\\n**Checkpoint:** You can explain how the change aligns with the concept.`
                    }
                    tutorialId={`concept-${concept.slug}`}
                    mode="concept"
                />
            </div>
        </div>
    );
}
