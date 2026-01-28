import React from 'react';
import clsx from 'clsx';
import styles from './Input.module.css';

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
    label?: string;
    error?: string;
    icon?: React.ReactNode;
    fullWidth?: boolean;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
    ({ label, error, icon, fullWidth = true, className, ...props }, ref) => {
        return (
            <div className={clsx(styles.wrapper, fullWidth && styles.fullWidth)}>
                {label && <label className={styles.label}>{label}</label>}
                <div className={clsx(styles.inputWrapper, error && styles.hasError)}>
                    {icon && <span className={styles.icon}>{icon}</span>}
                    <input
                        ref={ref}
                        className={clsx(styles.input, icon && styles.hasIcon, className)}
                        {...props}
                    />
                </div>
                {error && <span className={styles.error}>{error}</span>}
            </div>
        );
    }
);

Input.displayName = 'Input';

export default Input;
