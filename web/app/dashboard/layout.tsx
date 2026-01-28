'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import {
    LayoutDashboard,
    FileText,
    BookOpen,
    LogOut,
    Menu,
} from 'lucide-react';

export default function DashboardLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    const pathname = usePathname();
    const { user, logout } = useAuth();

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
    ];

    return (
        <div className="flex min-h-screen flex-col md:flex-row">
            {/* Sidebar */}
            <aside className="w-full border-r bg-muted/40 md:w-64 md:flex-col hidden md:flex">
                <div className="flex h-14 items-center border-b px-4 lg:h-[60px] lg:px-6">
                    <Link href="/" className="flex items-center gap-2 font-semibold">
                        <span className="">IssueSight</span>
                    </Link>
                </div>
                <div className="flex-1">
                    <nav className="grid items-start px-2 text-sm font-medium lg:px-4 mt-4 gap-1">
                        {navItems.map((item) => (
                            <Link
                                key={item.href}
                                href={item.href}
                                className={cn(
                                    "flex items-center gap-3 rounded-lg px-3 py-2 transition-all hover:text-primary",
                                    pathname === item.href
                                        ? "bg-muted text-primary"
                                        : "text-muted-foreground"
                                )}
                            >
                                <item.icon className="h-4 w-4" />
                                {item.title}
                            </Link>
                        ))}
                    </nav>
                </div>
                <div className="mt-auto p-4">
                    <div className="flex items-center gap-2 px-2 py-4 border-t">
                        {user && (
                            <div className="flex-1 overflow-hidden">
                                <p className="truncate text-sm font-medium">{user.display_name}</p>
                                <p className="truncate text-xs text-muted-foreground">{user.email}</p>
                            </div>
                        )}
                        <Button variant="ghost" size="icon" onClick={() => logout()}>
                            <LogOut className="h-4 w-4" />
                        </Button>
                    </div>
                </div>
            </aside>

            {/* Mobile Header (TODO: Add Sheet/Drawer) */}
            <div className="flex flex-col flex-1">
                <header className="flex h-14 items-center gap-4 border-b bg-muted/40 px-4 lg:h-[60px] lg:px-6 md:hidden">
                    <div className="w-full flex-1">
                        <span className="font-semibold">IssueSight</span>
                    </div>
                    <Button variant="ghost" size="icon" onClick={() => logout()}>
                        <LogOut className="h-4 w-4" />
                    </Button>
                </header>

                {/* Main Content */}
                <main className="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
                    {children}
                </main>
            </div>
        </div>
    );
}
