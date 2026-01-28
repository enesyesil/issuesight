import Link from 'next/link';
import { Button } from '@/components/ui/button';

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-24 text-center">
      <h1 className="text-4xl font-bold tracking-tight sm:text-6xl mb-6">
        IssueSight
      </h1>
      <p className="text-lg text-muted-foreground max-w-2xl mb-8">
        Turn GitHub Issues into AI-powered Tutorials instantly.
        Streamline your documentation workflow with intelligent automation.
      </p>
      <div className="flex gap-4">
        <Link href="/login">
          <Button size="lg">Get Started</Button>
        </Link>
        <a href="https://github.com/issuesight/issuesight" target="_blank" rel="noopener noreferrer">
          <Button variant="outline" size="lg">View on GitHub</Button>
        </a>
      </div>
    </main>
  );
}
