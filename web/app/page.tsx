import Link from 'next/link';
import Image from 'next/image';
import { Button } from '@/components/ui/button';

export default function Home() {
  return (
    <main className="relative min-h-screen">
      <div
        className="absolute inset-0 bg-cover bg-center"
        style={{ backgroundImage: "url('/hero-sight.png')" }}
        aria-hidden="true"
      />
      <div className="absolute inset-0 bg-black/55" aria-hidden="true" />

      <div className="relative z-10 flex min-h-screen flex-col items-center justify-center px-6 py-16 text-center">
        <Image
          src="/issue-logo.png"
          alt="IssueSight logo"
          width={140}
          height={140}
          className="mx-auto -mb-1"
          priority
        />
        <h1 className="text-4xl font-bold tracking-tight text-white sm:text-6xl mb-4 mt-0">
          IssueSight
        </h1>
        <p className="text-lg text-white/80 max-w-2xl mb-8">
          Turn GitHub issues into AI‑powered tutorials and context bridges for
          contributors.
        </p>
        <div className="flex flex-col sm:flex-row gap-4">
          <Link href="/login">
            <Button size="lg">Get Started</Button>
          </Link>
          <a href="https://github.com/issuesight/issuesight" target="_blank" rel="noopener noreferrer">
            <Button variant="outline" size="lg" className="bg-white/10 text-white border-white/30 hover:bg-white/20">
              View on GitHub
            </Button>
          </a>
        </div>
      </div>
    </main>
  );
}
