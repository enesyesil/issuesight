import React from 'react';
import clsx from 'clsx';
import styles from './Button.module.css';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: 'primary' | 'secondary' | 'ghost';
    size?: 'sm' | 'md' | 'lg';
    children: React.ReactNode;
    icon?: React.ReactNode;
    iconPosition?: 'left' | 'right';
    fullWidth?: boolean;
    loading?: boolean;
}

export function Button({
    variant = 'primary',
    size = 'md',
    children,
    icon,
    iconPosition = 'left',
    fullWidth = false,
    loading = false,
    disabled,
    className,
    ...props
}: ButtonProps) {
    return (
        <button
            className={clsx(
                styles.button,
                styles[variant],
                styles[size],
                fullWidth && styles.fullWidth,
                loading && styles.loading,
                className
            )}
            disabled={disabled || loading}
            {...props}
        >
            {loading ? (
                <span className={styles.spinner} />
            ) : (
                <>
                    {icon && iconPosition === 'left' && <span className={styles.icon}>{icon}</span>}
                    <span>{children}</span>
                    {icon && iconPosition === 'right' && <span className={styles.icon}>{icon}</span>}
                </>
            )}
        </button>
    );
}

export default Button;
