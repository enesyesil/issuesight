'use client';

import { useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { AlertCircle, CheckCircle2 } from 'lucide-react';

// Regex for validating GitHub issue URLs
// Matches: https://github.com/owner/repo/issues/123
const GITHUB_ISSUE_URL_REGEX = /^https:\/\/github\.com\/[\w.-]+\/[\w.-]+\/issues\/\d+$/;

/**
 * Validates if a URL is a valid GitHub issue URL
 */
function isValidGitHubIssueURL(url: string): boolean {
    return GITHUB_ISSUE_URL_REGEX.test(url.trim());
}

export function IssueForm({ className }: { className?: string }) {
    const [url, setUrl] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);
    const [validationError, setValidationError] = useState<string | null>(null);
    const router = useRouter();

    const validateUrl = useCallback((value: string) => {
        if (!value.trim()) {
            setValidationError(null);
            return;
        }
        
        if (!isValidGitHubIssueURL(value)) {
            setValidationError('Please enter a valid GitHub issue URL (e.g., https://github.com/owner/repo/issues/123)');
        } else {
            setValidationError(null);
        }
    }, []);

    const handleUrlChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const value = e.target.value;
        setUrl(value);
        // Clear validation error while typing, validate on blur
        if (validationError && isValidGitHubIssueURL(value)) {
            setValidationError(null);
        }
    };

    const handleBlur = () => {
        validateUrl(url);
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        
        // Validate before submission
        if (!isValidGitHubIssueURL(url)) {
            setValidationError('Please enter a valid GitHub issue URL');
            return;
        }
        
        setIsLoading(true);
        setError(null);
        setSuccess(null);
        setValidationError(null);

        try {
            const response = await api.issues.submit(url);
            setSuccess(response.message || 'Issue submitted successfully!');
            setUrl('');
        } catch (err: unknown) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to submit issue';
            setError(errorMessage);
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <Card className={`w-full max-w-3xl mx-auto overflow-hidden flex flex-col ${className ?? ''}`}>
            <CardHeader className="border-b border-border/60 bg-muted/30">
                <div className="space-y-2">
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Submit</p>
                    <CardTitle>Submit GitHub Issue</CardTitle>
                    <CardDescription>
                        Paste the URL of a GitHub issue to generate a junior-friendly tutorial with clear steps.
                    </CardDescription>
                </div>
            </CardHeader>
            <form onSubmit={handleSubmit} className="flex flex-1 flex-col">
                <CardContent className="space-y-6 flex-1 pt-6">
                    <div className="space-y-6">
                        <div className="rounded-2xl border border-border/60 bg-background/60 p-4 text-sm text-muted-foreground">
                            <p className="text-xs uppercase tracking-wide text-muted-foreground">How it works</p>
                            <ol className="mt-3 space-y-3 text-sm">
                                <li className="flex gap-3">
                                    <span className="mt-0.5 h-6 w-6 shrink-0 rounded-full border border-border/70 text-center text-xs font-semibold leading-6">
                                        1
                                    </span>
                                    <div>
                                        Paste the issue link and submit.
                                    </div>
                                </li>
                                <li className="flex gap-3">
                                    <span className="mt-0.5 h-6 w-6 shrink-0 rounded-full border border-border/70 text-center text-xs font-semibold leading-6">
                                        2
                                    </span>
                                    <div>
                                        We analyze the issue and extract key concepts.
                                    </div>
                                </li>
                                <li className="flex gap-3">
                                    <span className="mt-0.5 h-6 w-6 shrink-0 rounded-full border border-border/70 text-center text-xs font-semibold leading-6">
                                        3
                                    </span>
                                    <div>
                                        Your tutorial appears in the Tutorials tab.
                                    </div>
                                </li>
                            </ol>
                        </div>

                        <div className="rounded-2xl border border-border/70 bg-background p-5">
                            <div className="space-y-3">
                                <Label htmlFor="url">Issue URL</Label>
                                <Input
                                    id="url"
                                    placeholder="https://github.com/owner/repo/issues/123"
                                    value={url}
                                    onChange={handleUrlChange}
                                    onBlur={handleBlur}
                                    required
                                    disabled={isLoading}
                                    className={validationError ? 'border-destructive text-base h-12' : 'text-base h-12'}
                                    aria-invalid={!!validationError}
                                    aria-describedby={validationError ? 'url-error' : undefined}
                                />
                                {validationError && (
                                    <p id="url-error" className="text-sm text-destructive">
                                        {validationError}
                                    </p>
                                )}
                                <p className="text-xs text-muted-foreground">
                                    Example: <span className="font-medium">https://github.com/owner/repo/issues/123</span>
                                </p>
                            </div>
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
                    </div>
                </CardContent>
                <CardFooter>
                    <Button
                        type="submit"
                        disabled={isLoading || !!validationError || !url.trim()}
                        className="w-full"
                    >
                        {isLoading ? 'Processing...' : 'Generate Tutorial'}
                    </Button>
                </CardFooter>
            </form>
        </Card>
    );
}
