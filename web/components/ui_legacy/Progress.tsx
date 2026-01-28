import React from 'react';
import clsx from 'clsx';
import styles from './Progress.module.css';

export interface ProgressProps {
    value: number;
    max: number;
    variant?: 'default' | 'success' | 'warning' | 'error';
    size?: 'sm' | 'md';
    showLabel?: boolean;
    className?: string;
}

export function Progress({
    value,
    max,
    variant = 'default',
    size = 'md',
    showLabel = false,
    className,
}: ProgressProps) {
    const percentage = Math.min(100, Math.max(0, (value / max) * 100));

    // Auto-determine variant based on percentage if default
    const autoVariant = variant === 'default'
        ? percentage >= 100 ? 'error' : percentage >= 80 ? 'warning' : 'success'
        : variant;

    return (
        <div className={clsx(styles.wrapper, className)}>
            <div className={clsx(styles.track, styles[size])}>
                <div
                    className={clsx(styles.fill, styles[autoVariant])}
                    style={{ width: `${percentage}%` }}
                />
            </div>
            {showLabel && (
                <span className={styles.label}>
                    {value}/{max}
                </span>
            )}
        </div>
    );
}

export default Progress;
