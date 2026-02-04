'use client';

import { useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { AlertCircle, RefreshCw, ArrowLeft } from 'lucide-react';
import Link from 'next/link';

export default function DashboardError({
    error,
    reset,
}: {
    error: Error & { digest?: string };
    reset: () => void;
}) {
    useEffect(() => {
        // Log the error to console (in production, send to error reporting service)
        console.error('Dashboard error:', error);
    }, [error]);

    return (
        <div className="flex flex-col items-center justify-center min-h-[400px] p-8 border rounded-lg border-destructive/20 bg-destructive/5">
            <AlertCircle className="h-10 w-10 text-destructive mb-4" />
            <h2 className="text-xl font-bold mb-2">Dashboard Error</h2>
            <p className="text-muted-foreground mb-6 text-center max-w-md">
                Something went wrong while loading this page. Please try again.
            </p>
            <div className="flex gap-4">
                <Button onClick={reset} variant="default" aria-label="Try again">
                    <RefreshCw className="h-4 w-4 mr-2" />
                    Try again
                </Button>
                <Button variant="outline" asChild>
                    <Link href="/dashboard/issues" aria-label="Go to issues page">
                        <ArrowLeft className="h-4 w-4 mr-2" />
                        Go to Issues
                    </Link>
                </Button>
            </div>
        </div>
    );
}
