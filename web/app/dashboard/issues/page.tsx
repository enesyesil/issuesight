import { IssueForm } from '@/components/issue/IssueForm';

export default function IssuesPage() {
    return (
        <div className="flex flex-col gap-8">
            <div>
                <p className="text-xs uppercase tracking-wide text-muted-foreground">Submit</p>
                <h1 className="text-3xl font-bold tracking-tight">Issues</h1>
                <p className="text-muted-foreground">
                    Submit GitHub issues to generate AI-powered tutorials.
                </p>
            </div>

            <div className="grid gap-8 lg:grid-cols-[1.1fr_1fr] items-stretch">
                <div className="space-y-6">
                    <div className="rounded-2xl border border-border/70 bg-muted/30 p-6">
                        <h2 className="text-lg font-semibold">What you will get</h2>
                        <p className="mt-2 text-sm text-muted-foreground">
                            We turn the issue into a tutorial with context, a step-by-step plan, and
                            clear checkpoints so junior contributors can follow confidently.
                        </p>
                        <div className="mt-4 grid gap-3 sm:grid-cols-2 auto-rows-fr">
                            <div className="rounded-xl border border-border/70 bg-background p-4 text-sm h-full">
                                <p className="text-xs uppercase tracking-wide text-muted-foreground">Structure</p>
                                <p className="mt-2 font-medium">Context, plan, testing, pitfalls</p>
                                <p className="mt-1 text-muted-foreground">
                                    Everything you need to understand before writing code.
                                </p>
                            </div>
                            <div className="rounded-xl border border-border/70 bg-background p-4 text-sm h-full">
                                <p className="text-xs uppercase tracking-wide text-muted-foreground">Steps</p>
                                <p className="mt-2 font-medium">Guided checklist</p>
                                <p className="mt-1 text-muted-foreground">
                                    Each step includes goal, why, what to do, and checkpoint.
                                </p>
                            </div>
                            <div className="rounded-xl border border-border/70 bg-background p-4 text-sm h-full">
                                <p className="text-xs uppercase tracking-wide text-muted-foreground">Concepts</p>
                                <p className="mt-2 font-medium">Auto-linked knowledge</p>
                                <p className="mt-1 text-muted-foreground">
                                    Related concepts are pulled in for quick review.
                                </p>
                            </div>
                            <div className="rounded-xl border border-border/70 bg-background p-4 text-sm h-full">
                                <p className="text-xs uppercase tracking-wide text-muted-foreground">Delivery</p>
                                <p className="mt-2 font-medium">Ready to share</p>
                                <p className="mt-1 text-muted-foreground">
                                    Tutorials are saved to the dashboard for easy access.
                                </p>
                            </div>
                        </div>
                    </div>

                    <div className="rounded-2xl border border-border/70 bg-background p-6">
                        <h2 className="text-lg font-semibold">Before you submit</h2>
                        <div className="mt-3 grid gap-3 text-sm text-muted-foreground">
                            <div className="flex items-start gap-3">
                                <span className="mt-0.5 h-6 w-6 rounded-full border border-border/70 text-center text-xs font-semibold leading-6">
                                    1
                                </span>
                                <span>Make sure the issue has a clear title and description.</span>
                            </div>
                            <div className="flex items-start gap-3">
                                <span className="mt-0.5 h-6 w-6 rounded-full border border-border/70 text-center text-xs font-semibold leading-6">
                                    2
                                </span>
                                <span>Include links or screenshots if the issue is UI-related.</span>
                            </div>
                            <div className="flex items-start gap-3">
                                <span className="mt-0.5 h-6 w-6 rounded-full border border-border/70 text-center text-xs font-semibold leading-6">
                                    3
                                </span>
                                <span>Verify the issue is public and accessible.</span>
                            </div>
                        </div>
                    </div>
                </div>

                <IssueForm className="h-full" />
            </div>
        </div>
    );
}
