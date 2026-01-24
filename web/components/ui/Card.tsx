import React from 'react';
import clsx from 'clsx';
import styles from './Card.module.css';

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
    children: React.ReactNode;
    padding?: 'none' | 'sm' | 'md' | 'lg';
    hoverable?: boolean;
}

export function Card({
    children,
    padding = 'md',
    hoverable = false,
    className,
    ...props
}: CardProps) {
    return (
        <div
            className={clsx(
                styles.card,
                styles[`padding-${padding}`],
                hoverable && styles.hoverable,
                className
            )}
            {...props}
        >
            {children}
        </div>
    );
}

export interface CardHeaderProps {
    children: React.ReactNode;
    icon?: React.ReactNode;
    action?: React.ReactNode;
}

export function CardHeader({ children, icon, action }: CardHeaderProps) {
    return (
        <div className={styles.header}>
            <div className={styles.headerContent}>
                {icon && <span className={styles.headerIcon}>{icon}</span>}
                <h3 className={styles.headerTitle}>{children}</h3>
            </div>
            {action && <div className={styles.headerAction}>{action}</div>}
        </div>
    );
}

export function CardContent({ children, className }: { children: React.ReactNode; className?: string }) {
    return <div className={clsx(styles.content, className)}>{children}</div>;
}

export default Card;
