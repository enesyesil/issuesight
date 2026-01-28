import { IssueForm } from '@/components/issue/IssueForm';

export default function IssuesPage() {
    return (
        <div className="flex flex-col gap-6">
            <div>
                <h1 className="text-3xl font-bold tracking-tight">Issues</h1>
                <p className="text-muted-foreground">
                    Submit GitHub issues to generate AI-powered tutorials.
                </p>
            </div>
            <IssueForm />
        </div>
    );
}
