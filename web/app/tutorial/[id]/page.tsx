'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';
import {
    ArrowLeft, Copy, Bookmark, Share2, Clock, Github, Check,
    BookOpen, CheckSquare, ListChecks, ChevronRight, ExternalLink
} from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardHeader, CardContent } from '@/components/ui/Card';
import styles from './page.module.css';

// Mock data for demonstration
const mockTutorial = {
    id: '1',
    title: 'Understanding React Server Components',
    repo: 'facebook/react',
    issue_number: 25456,
    status: 'COMPLETED' as const,
    created_at: '2026-01-13',
    read_time: 8,

    // Section 1: Concepts needed for this project
    concepts: [
        {
            slug: 'server-components',
            name: 'Server Components',
            description: 'React components that render exclusively on the server, reducing client-side JavaScript bundle size.',
        },
        {
            slug: 'code-splitting',
            name: 'Code Splitting',
            description: 'Technique to split code into smaller chunks that can be loaded on demand.',
        },
        {
            slug: 'react-hooks',
            name: 'React Hooks',
            description: 'Functions that let you use state and other React features in functional components.',
        },
        {
            slug: 'suspense',
            name: 'React Suspense',
            description: 'A mechanism for handling async operations in React with loading states.',
        },
    ],

    // Section 2: Checklist of the issue
    checklist: [
        { id: 1, text: 'Understand what Server Components are', completed: true },
        { id: 2, text: 'Learn the difference between Server and Client Components', completed: true },
        { id: 3, text: 'Identify when to use `"use client"` directive', completed: false },
        { id: 4, text: 'Implement a basic Server Component', completed: false },
        { id: 5, text: 'Handle data fetching in Server Components', completed: false },
        { id: 6, text: 'Mix Server and Client Components properly', completed: false },
    ],

    // Section 3: Step-by-step tutorial
    steps: [
        {
            id: 1,
            title: 'Understanding Server Components',
            content: `React Server Components (RSC) represent a paradigm shift in how we build React applications. Unlike traditional React components that run on both server and client, Server Components render **exclusively on the server**.

### Key Characteristics

- **Zero JavaScript to client**: Server Components don't add to your bundle
- **Direct backend access**: Query databases directly without API routes
- **Automatic code splitting**: The framework handles everything

\`\`\`tsx
// This is a Server Component (default in Next.js 13+)
async function UserProfile({ userId }: { userId: string }) {
  // Direct database access - no API needed!
  const user = await db.users.findUnique({
    where: { id: userId }
  });

  return (
    <div className="profile">
      <h1>{user.name}</h1>
      <p>{user.bio}</p>
    </div>
  );
}
\`\`\``,
        },
        {
            id: 2,
            title: 'Server vs Client Components',
            content: `Understanding when to use each type is crucial for building performant applications.

### Server Components (Default)
- Fetch data
- Access backend resources
- Keep sensitive info on server (API keys, etc.)
- Large dependencies that shouldn't ship to client

### Client Components
- Add interactivity (onClick, onChange)
- Use state and lifecycle effects
- Use browser-only APIs
- Use custom hooks that depend on state/effects

| Feature | Server Component | Client Component |
|---------|------------------|------------------|
| Fetch data | ✅ Preferred | ⚠️ Works |
| Use state | ❌ No | ✅ Yes |
| Event handlers | ❌ No | ✅ Yes |
| Browser APIs | ❌ No | ✅ Yes |`,
        },
        {
            id: 3,
            title: 'Using the "use client" Directive',
            content: `Add the \`'use client'\` directive at the top of a file to mark it as a Client Component.

\`\`\`tsx
'use client';

import { useState } from 'react';

export function Counter() {
  const [count, setCount] = useState(0);
  
  return (
    <button onClick={() => setCount(c => c + 1)}>
      Count: {count}
    </button>
  );
}
\`\`\`

### Important Notes

1. The directive must be at the **very top** of the file
2. All modules imported by a Client Component become part of the client bundle
3. You can import Server Components into Client Components, but they render on the server`,
        },
        {
            id: 4,
            title: 'Implementing Your First Server Component',
            content: `Let's create a practical Server Component that fetches and displays blog posts.

\`\`\`tsx
// app/posts/page.tsx - This is a Server Component

import { db } from '@/lib/db';

export default async function PostsPage() {
  // This runs on the server only
  const posts = await db.post.findMany({
    orderBy: { createdAt: 'desc' },
    take: 10,
  });

  return (
    <main>
      <h1>Latest Posts</h1>
      <ul>
        {posts.map(post => (
          <li key={post.id}>
            <a href={\`/posts/\${post.id}\`}>{post.title}</a>
          </li>
        ))}
      </ul>
    </main>
  );
}
\`\`\`

### Benefits of This Approach

- **No loading spinners**: Data is fetched before sending HTML
- **SEO friendly**: Content is in the initial HTML
- **No API route needed**: Direct database access`,
        },
    ],
};

export default function TutorialPage({ params }: { params: { id: string } }) {
    const [copied, setCopied] = useState(false);
    const [checkedItems, setCheckedItems] = useState<Set<number>>(
        new Set(mockTutorial.checklist.filter(item => item.completed).map(item => item.id))
    );
    const [activeStep, setActiveStep] = useState(1);

    const handleCopyLink = () => {
        navigator.clipboard.writeText(window.location.href);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    const toggleCheckItem = (id: number) => {
        setCheckedItems(prev => {
            const next = new Set(prev);
            if (next.has(id)) {
                next.delete(id);
            } else {
                next.add(id);
            }
            return next;
        });
    };

    const progress = Math.round((checkedItems.size / mockTutorial.checklist.length) * 100);

    return (
        <div className={styles.tutorialPage}>
            {/* Back Navigation */}
            <Link href="/dashboard" className={styles.backLink}>
                <ArrowLeft size={16} />
                <span>Back to Dashboard</span>
            </Link>

            {/* Tutorial Header */}
            <header className={styles.header}>
                <div className={styles.badges}>
                    <Badge variant="success">Tutorial</Badge>
                    <span className={styles.readTime}>
                        <Clock size={14} />
                        {mockTutorial.read_time} min read
                    </span>
                </div>

                <h1 className={styles.title}>{mockTutorial.title}</h1>

                <div className={styles.meta}>
                    <div className={styles.repoInfo}>
                        <Github size={16} />
                        <span>{mockTutorial.repo} #{mockTutorial.issue_number}</span>
                    </div>
                    <div className={styles.date}>
                        <Clock size={14} />
                        <span>{mockTutorial.created_at}</span>
                    </div>
                </div>

                {/* Action Buttons */}
                <div className={styles.actions}>
                    <Button variant="secondary" size="sm" icon={copied ? <Check size={14} /> : <Copy size={14} />} onClick={handleCopyLink}>
                        {copied ? 'Copied!' : 'Copy Link'}
                    </Button>
                    <Button variant="secondary" size="sm" icon={<Bookmark size={14} />}>
                        Bookmark
                    </Button>
                    <Button variant="secondary" size="sm" icon={<Share2 size={14} />}>
                        Share
                    </Button>
                </div>
            </header>

            {/* Main Content Area - 3 Sections */}
            <div className={styles.contentLayout}>

                {/* Section 1: Key Concepts */}
                <section className={styles.section}>
                    <div className={styles.sectionHeader}>
                        <BookOpen size={20} className={styles.sectionIcon} />
                        <h2 className={styles.sectionTitle}>Key Concepts</h2>
                        <span className={styles.sectionBadge}>{mockTutorial.concepts.length} concepts</span>
                    </div>
                    <p className={styles.sectionDescription}>
                        Before diving in, make sure you understand these foundational concepts.
                    </p>
                    <div className={styles.conceptsGrid}>
                        {mockTutorial.concepts.map((concept) => (
                            <Link
                                href={`/concept/${concept.slug}`}
                                key={concept.slug}
                                className={styles.conceptCard}
                            >
                                <div className={styles.conceptHeader}>
                                    <span className={styles.conceptName}>{concept.name}</span>
                                    <ChevronRight size={16} className={styles.conceptArrow} />
                                </div>
                                <p className={styles.conceptDescription}>{concept.description}</p>
                            </Link>
                        ))}
                    </div>
                </section>

                {/* Section 2: Issue Checklist */}
                <section className={styles.section}>
                    <div className={styles.sectionHeader}>
                        <ListChecks size={20} className={styles.sectionIcon} />
                        <h2 className={styles.sectionTitle}>Issue Checklist</h2>
                        <span className={styles.progressBadge}>{progress}% complete</span>
                    </div>
                    <p className={styles.sectionDescription}>
                        Track your progress through this issue's requirements.
                    </p>
                    <div className={styles.checklist}>
                        {mockTutorial.checklist.map((item) => (
                            <label
                                key={item.id}
                                className={`${styles.checklistItem} ${checkedItems.has(item.id) ? styles.checked : ''}`}
                            >
                                <input
                                    type="checkbox"
                                    checked={checkedItems.has(item.id)}
                                    onChange={() => toggleCheckItem(item.id)}
                                    className={styles.checkbox}
                                />
                                <CheckSquare
                                    size={20}
                                    className={checkedItems.has(item.id) ? styles.checkboxChecked : styles.checkboxUnchecked}
                                />
                                <span className={styles.checklistText}>{item.text}</span>
                            </label>
                        ))}
                    </div>
                    <div className={styles.progressBar}>
                        <div className={styles.progressFill} style={{ width: `${progress}%` }} />
                    </div>
                </section>

                {/* Section 3: Step-by-Step Tutorial */}
                <section className={styles.section}>
                    <div className={styles.sectionHeader}>
                        <BookOpen size={20} className={styles.sectionIcon} />
                        <h2 className={styles.sectionTitle}>Step-by-Step Tutorial</h2>
                        <span className={styles.sectionBadge}>{mockTutorial.steps.length} steps</span>
                    </div>
                    <p className={styles.sectionDescription}>
                        Follow along with this detailed walkthrough to complete the issue.
                    </p>

                    {/* Step Navigation */}
                    <div className={styles.stepNav}>
                        {mockTutorial.steps.map((step) => (
                            <button
                                key={step.id}
                                className={`${styles.stepNavItem} ${activeStep === step.id ? styles.stepNavActive : ''}`}
                                onClick={() => setActiveStep(step.id)}
                            >
                                <span className={styles.stepNumber}>{step.id}</span>
                                <span className={styles.stepNavTitle}>{step.title}</span>
                            </button>
                        ))}
                    </div>

                    {/* Active Step Content */}
                    {mockTutorial.steps.map((step) => (
                        <article
                            key={step.id}
                            className={`${styles.stepContent} ${activeStep === step.id ? styles.stepActive : ''}`}
                        >
                            <h3 className={styles.stepTitle}>
                                <span className={styles.stepIndicator}>Step {step.id}</span>
                                {step.title}
                            </h3>
                            <div className={styles.markdownContent}>
                                <ReactMarkdown
                                    components={{
                                        code({ node, inline, className, children, ...props }: any) {
                                            const match = /language-(\w+)/.exec(className || '');
                                            return !inline && match ? (
                                                <SyntaxHighlighter
                                                    style={oneDark}
                                                    language={match[1]}
                                                    PreTag="div"
                                                    {...props}
                                                >
                                                    {String(children).replace(/\n$/, '')}
                                                </SyntaxHighlighter>
                                            ) : (
                                                <code className={className} {...props}>
                                                    {children}
                                                </code>
                                            );
                                        },
                                    }}
                                >
                                    {step.content}
                                </ReactMarkdown>
                            </div>

                            {/* Step Navigation Buttons */}
                            <div className={styles.stepActions}>
                                {step.id > 1 && (
                                    <Button
                                        variant="ghost"
                                        onClick={() => setActiveStep(step.id - 1)}
                                        icon={<ArrowLeft size={16} />}
                                    >
                                        Previous Step
                                    </Button>
                                )}
                                {step.id < mockTutorial.steps.length && (
                                    <Button
                                        variant="primary"
                                        onClick={() => setActiveStep(step.id + 1)}
                                        icon={<ChevronRight size={16} />}
                                        iconPosition="right"
                                    >
                                        Next Step
                                    </Button>
                                )}
                                {step.id === mockTutorial.steps.length && (
                                    <Button
                                        variant="primary"
                                        icon={<Check size={16} />}
                                    >
                                        Complete Tutorial
                                    </Button>
                                )}
                            </div>
                        </article>
                    ))}
                </section>
            </div>
        </div>
    );
}
