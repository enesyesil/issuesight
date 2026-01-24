'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Github, Sparkles, Clock, ExternalLink, Timer } from 'lucide-react';
import { Card, CardHeader, CardContent } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Progress } from '@/components/ui/Progress';
import { StatusBadge } from '@/components/ui/Badge';
import { UserMenu } from '@/components/layout/UserMenu';
import styles from './page.module.css';

// Mock data for demonstration
const mockUser = {
    name: 'Demo User',
    plan: 'free' as const,
};

const mockQuota = {
    used: 1,
    limit: 1,
    resets_at: 'Monday, Jan 26',
};

const mockTutorials = [
    {
        id: '1',
        title: 'Understanding React Server Components',
        repo: 'facebook/react',
        issue_number: 25456,
        status: 'COMPLETED' as const,
        created_at: '2026-01-13',
        description: 'A comprehensive guide to understanding React Server Components, their benefits, and how to implement them in your Next.js applications...',
    },
    {
        id: '2',
        title: 'Fixing TypeScript Strict Mode Errors',
        repo: 'microsoft/TypeScript',
        issue_number: 51234,
        status: 'COMPLETED' as const,
        created_at: '2026-01-12',
        description: 'Learn how to resolve common TypeScript strict mode errors and improve type safety in your codebase...',
    },
];

const mockStats = {
    total: 3,
    completed: 2,
    processing: 1,
};

export default function DashboardPage() {
    const [issueUrl, setIssueUrl] = useState('');
    const [isSubmitting, setIsSubmitting] = useState(false);

    const isQuotaExhausted = mockQuota.used >= mockQuota.limit;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!issueUrl || isQuotaExhausted) return;

        setIsSubmitting(true);
        // TODO: Call API
        setTimeout(() => {
            setIsSubmitting(false);
            setIssueUrl('');
        }, 1500);
    };

    return (
        <div className={styles.dashboard}>
            {/* Left Column */}
            <div className={styles.mainColumn}>
                {/* Page Header */}
                <div className={styles.pageHeader}>
                    <h1 className={styles.pageTitle}>Dashboard</h1>
                    <p className={styles.pageSubtitle}>Submit issues and manage your generated tutorials</p>
                </div>

                {/* Submit Issue Card */}
                <Card>
                    <CardHeader icon={<Github size={20} />}>Submit GitHub Issue</CardHeader>
                    <CardContent>
                        <p className={styles.cardDescription}>
                            Paste a GitHub issue URL to generate a comprehensive tutorial
                        </p>
                        <form onSubmit={handleSubmit} className={styles.submitForm}>
                            <Input
                                value={issueUrl}
                                onChange={(e) => setIssueUrl(e.target.value)}
                                placeholder="https://github.com/owner/repo/issues/123"
                                disabled={isQuotaExhausted}
                            />
                            <Button
                                type="submit"
                                variant="primary"
                                disabled={isQuotaExhausted || !issueUrl}
                                loading={isSubmitting}
                                icon={<Sparkles size={16} />}
                            >
                                Generate
                            </Button>
                        </form>
                        {isQuotaExhausted && (
                            <p className={styles.quotaWarning}>
                                You've used your weekly quota. Come back next week for more insights!
                            </p>
                        )}
                    </CardContent>
                </Card>

                {/* Tutorials Section */}
                <section className={styles.tutorialsSection}>
                    <h2 className={styles.sectionTitle}>
                        <span className={styles.sectionIcon}>📚</span>
                        Your Tutorials
                    </h2>

                    <div className={styles.tutorialsList}>
                        {mockTutorials.map((tutorial) => (
                            <Card key={tutorial.id} hoverable>
                                <div className={styles.tutorialCard}>
                                    <div className={styles.tutorialHeader}>
                                        <h3 className={styles.tutorialTitle}>{tutorial.title}</h3>
                                        <StatusBadge status={tutorial.status} />
                                    </div>
                                    <div className={styles.tutorialMeta}>
                                        <Github size={14} />
                                        <span>{tutorial.repo} #{tutorial.issue_number}</span>
                                    </div>
                                    <p className={styles.tutorialDescription}>{tutorial.description}</p>
                                    <div className={styles.tutorialFooter}>
                                        <div className={styles.tutorialDate}>
                                            <Clock size={14} />
                                            <span>{tutorial.created_at}</span>
                                        </div>
                                        <Link href={`/tutorial/${tutorial.id}`} className={styles.readLink}>
                                            <span>Read Tutorial</span>
                                            <ExternalLink size={14} />
                                        </Link>
                                    </div>
                                </div>
                            </Card>
                        ))}
                    </div>
                </section>
            </div>

            {/* Right Column */}
            <aside className={styles.sidebar}>
                {/* User Info */}
                <UserMenu name={mockUser.name} plan={mockUser.plan} />

                {/* Weekly Quota Card */}
                <Card>
                    <CardHeader icon={<Timer size={20} />}>Weekly Quota</CardHeader>
                    <CardContent className={styles.quotaContent}>
                        <div className={styles.quotaHeader}>
                            <span>Used this week</span>
                            <span className={styles.quotaValue}>{mockQuota.used}/{mockQuota.limit}</span>
                        </div>
                        <Progress value={mockQuota.used} max={mockQuota.limit} />
                        <div className={styles.quotaInfo}>
                            <Clock size={14} />
                            <span>Resets on {mockQuota.resets_at}</span>
                        </div>
                        {isQuotaExhausted && (
                            <p className={styles.quotaExhausted}>Quota exhausted for this week</p>
                        )}
                    </CardContent>
                </Card>

                {/* Quick Stats Card */}
                <Card>
                    <CardHeader>Quick Stats</CardHeader>
                    <CardContent>
                        <div className={styles.statsGrid}>
                            <div className={styles.statRow}>
                                <span>Total Tutorials</span>
                                <span className={styles.statNumber}>{mockStats.total}</span>
                            </div>
                            <div className={styles.statRow}>
                                <span>Completed</span>
                                <span className={styles.statNumberGreen}>{mockStats.completed}</span>
                            </div>
                            <div className={styles.statRow}>
                                <span>Processing</span>
                                <span className={styles.statNumber}>{mockStats.processing}</span>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </aside>
        </div>
    );
}
