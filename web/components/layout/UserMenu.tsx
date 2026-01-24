import React from 'react';
import styles from './UserMenu.module.css';

export interface UserMenuProps {
    name: string;
    plan: 'free' | 'pro';
    avatarUrl?: string;
}

export function UserMenu({ name, plan, avatarUrl }: UserMenuProps) {
    const initials = name
        .split(' ')
        .map((n) => n[0])
        .join('')
        .toUpperCase()
        .slice(0, 2);

    return (
        <div className={styles.userMenu}>
            <div className={styles.avatar}>
                {avatarUrl ? (
                    <img src={avatarUrl} alt={name} className={styles.avatarImage} />
                ) : (
                    <span className={styles.avatarInitials}>{initials}</span>
                )}
            </div>
            <div className={styles.info}>
                <span className={styles.name}>{name}</span>
                <span className={styles.plan}>{plan === 'pro' ? 'Pro Plan' : 'Free Plan'}</span>
            </div>
        </div>
    );
}

export default UserMenu;
