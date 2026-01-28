import React from 'react';
import clsx from 'clsx';
import styles from './Badge.module.css';

export interface BadgeProps {
    children: React.ReactNode;
    variant?: 'default' | 'success' | 'warning' | 'error' | 'info';
    size?: 'sm' | 'md';
    className?: string;
}

export function Badge({
    children,
    variant = 'default',
    size = 'sm',
    className,
}: BadgeProps) {
    return (
        <span className={clsx(styles.badge, styles[variant], styles[size], className)}>
            {children}
        </span>
    );
}

export function StatusBadge({ status }: { status: 'PENDING' | 'COMPLETED' | 'FAILED' }) {
    const variantMap = {
        PENDING: 'warning',
        COMPLETED: 'success',
        FAILED: 'error',
    } as const;

    const labelMap = {
        PENDING: 'processing',
        COMPLETED: 'completed',
        FAILED: 'failed',
    };

    return (
        <Badge variant={variantMap[status]}>
            {labelMap[status]}
        </Badge>
    );
}

export default Badge;
