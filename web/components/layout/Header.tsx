'use client';

import React from 'react';
import Link from 'next/link';
import { Zap, Github } from 'lucide-react';
import { Button } from '../ui/Button';
import styles from './Header.module.css';

export interface HeaderProps {
    showDashboardLink?: boolean;
}

export function Header({ showDashboardLink = true }: HeaderProps) {
    return (
        <header className={styles.header}>
            <div className={styles.container}>
                {/* Logo */}
                <Link href="/" className={styles.logo}>
                    <span className={styles.logoIcon}>
                        <Zap size={24} />
                    </span>
                    <span className={styles.logoText}>IssueSight</span>
                </Link>

                {/* Navigation */}
                <nav className={styles.nav}>
                    <Link href="#features" className={styles.navLink}>
                        Features
                    </Link>
                    <Link href="#how-it-works" className={styles.navLink}>
                        How it Works
                    </Link>
                    {showDashboardLink && (
                        <Link href="/dashboard" className={styles.navLink}>
                            Dashboard
                        </Link>
                    )}
                </nav>

                {/* Right Side Actions */}
                <div className={styles.actions}>
                    <a
                        href="https://github.com/issuesight/issuesight"
                        target="_blank"
                        rel="noopener noreferrer"
                        className={styles.githubLink}
                        aria-label="View on GitHub"
                    >
                        <Github size={20} />
                    </a>
                    <Link href="/login" className={styles.signIn}>
                        Sign In
                    </Link>
                    <Link href="/login">
                        <Button variant="primary" size="sm">
                            Get Started
                        </Button>
                    </Link>
                </div>
            </div>
        </header>
    );
}

export default Header;
