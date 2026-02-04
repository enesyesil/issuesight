'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import ErrorBoundary from '@/components/ErrorBoundary';
import {
    LayoutDashboard,
    BookOpen,
    Lightbulb,
    LogOut,
    Loader2,
} from 'lucide-react';

export default function DashboardLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    const pathname = usePathname();
    const router = useRouter();
    const { user, isLoading, isAuthenticated, logout } = useAuth();
    const [isLoggingOut, setIsLoggingOut] = useState(false);

    // Auth guard - redirect to login if not authenticated
    useEffect(() => {
        if (!isLoading && !isAuthenticated) {
            router.push('/login');
        }
    }, [isLoading, isAuthenticated, router]);

    // Show loading while checking auth
    if (isLoading) {
        return (
            <div className="flex min-h-screen items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    // Don't render dashboard if not authenticated (will redirect)
    if (!isAuthenticated) {
        return null;
    }

    const handleLogout = async () => {
        setIsLoggingOut(true);
        try {
            await logout();
        } finally {
            setIsLoggingOut(false);
        }
    };

    const navItems = [
        {
            title: 'Issues',
            href: '/dashboard/issues',
            icon: LayoutDashboard,
        },
        {
            title: 'Tutorials',
            href: '/dashboard/tutorials',
            icon: BookOpen,
        },
        {
            title: 'Concepts',
            href: '/dashboard/concepts',
            icon: Lightbulb,
        },
    ];

    return (
        <div className="min-h-screen bg-background flex flex-col">
            <header className="sticky top-0 z-30 border-b bg-background/90 backdrop-blur shadow-sm px-4 lg:px-6">
                <div className="mx-auto flex w-full max-w-6xl items-center gap-6 py-4">
                    <Link href="/" className="flex items-center gap-3 font-semibold tracking-wide">
                        <Image
                            src="/issue-logo.png"
                            alt="IssueSight logo"
                            width={40}
                            height={40}
                            className="h-10 w-10"
                        />
                        <span>IssueSight</span>
                    </Link>
                    <nav className="hidden md:flex items-center gap-2 text-sm font-medium">
                        {navItems.map((item) => (
                            <Link
                                key={item.href}
                                href={item.href}
                                className={cn(
                                    "flex items-center gap-2 rounded-full px-4 py-2 transition-colors",
                                    pathname === item.href
                                        ? "bg-primary/10 text-primary"
                                        : "text-muted-foreground hover:text-foreground"
                                )}
                            >
                                <item.icon className="h-4 w-4" />
                                {item.title}
                            </Link>
                        ))}
                    </nav>
                    <div className="ml-auto flex items-center gap-3">
                        {user && (
                            <div className="hidden sm:flex flex-col text-right">
                                <span className="text-xs text-muted-foreground">Signed in as</span>
                                <span className="text-sm font-medium truncate max-w-[200px]">
                                    {user.display_name}
                                </span>
                            </div>
                        )}
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={handleLogout}
                            disabled={isLoggingOut}
                            aria-label="Log out"
                        >
                            {isLoggingOut ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                                <LogOut className="h-4 w-4" />
                            )}
                        </Button>
                    </div>
                </div>
                <div className="md:hidden border-t pb-3">
                    <nav className="flex gap-2 overflow-x-auto pt-3 text-sm font-medium">
                        {navItems.map((item) => (
                            <Link
                                key={item.href}
                                href={item.href}
                                className={cn(
                                    "flex items-center gap-2 whitespace-nowrap rounded-full px-4 py-2",
                                    pathname === item.href
                                        ? "bg-primary/15 text-primary"
                                        : "bg-muted/60 text-muted-foreground"
                                )}
                            >
                                <item.icon className="h-4 w-4" />
                                {item.title}
                            </Link>
                        ))}
                    </nav>
                </div>
            </header>

            <main className="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
                <ErrorBoundary>
                    <div className="mx-auto w-full max-w-6xl">
                        {children}
                    </div>
                </ErrorBoundary>
            </main>

            <footer className="border-t bg-muted/30 px-4 lg:px-6">
                <div className="mx-auto flex w-full max-w-6xl flex-col gap-3 py-6 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
                    <span>IssueSight — turn GitHub issues into guided tutorials.</span>
                    <div className="flex flex-wrap items-center gap-4">
                        <Link href="/" className="hover:text-foreground">Home</Link>
                        <Link href="/dashboard/issues" className="hover:text-foreground">Submit Issue</Link>
                        <Link href="/dashboard/tutorials" className="hover:text-foreground">Tutorials</Link>
                    </div>
                </div>
            </footer>
        </div>
    );
}
