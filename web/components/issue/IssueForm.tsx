'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { AlertCircle, CheckCircle2 } from 'lucide-react';

export function IssueForm() {
    const [url, setUrl] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);
    const router = useRouter();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsLoading(true);
        setError(null);
        setSuccess(null);

        try {
            const response = await api.issues.submit(url);
            setSuccess(response.message || 'Issue submitted successfully!');
            setUrl('');
            // Optional: Redirect to issue detail or tutorials
            // router.push(`/dashboard/issues/${response.id}`);
        } catch (err: any) {
            setError(err.message || 'Failed to submit issue');
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <Card className="w-full max-w-2xl mx-auto">
            <CardHeader>
                <CardTitle>Submit GitHub Issue</CardTitle>
                <CardDescription>
                    Paste the URL of a GitHub issue to generate a tutorial.
                </CardDescription>
            </CardHeader>
            <form onSubmit={handleSubmit}>
                <CardContent className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="url">Issue URL</Label>
                        <Input
                            id="url"
                            placeholder="https://github.com/owner/repo/issues/123"
                            value={url}
                            onChange={(e) => setUrl(e.target.value)}
                            required
                            disabled={isLoading}
                        />
                    </div>
                    {error && (
                        <div className="flex items-center gap-2 text-sm text-destructive bg-destructive/10 p-3 rounded-md">
                            <AlertCircle className="h-4 w-4" />
                            <span>{error}</span>
                        </div>
                    )}
                    {success && (
                        <div className="flex items-center gap-2 text-sm text-green-600 bg-green-50 p-3 rounded-md dark:bg-green-900/20 dark:text-green-400">
                            <CheckCircle2 className="h-4 w-4" />
                            <span>{success}</span>
                        </div>
                    )}
                </CardContent>
                <CardFooter>
                    <Button type="submit" disabled={isLoading} className="w-full">
                        {isLoading ? 'Processing...' : 'Generate Tutorial'}
                    </Button>
                </CardFooter>
            </form>
        </Card>
    );
}
