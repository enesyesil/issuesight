import React from 'react';
import clsx from 'clsx';
import styles from './ConceptTag.module.css';

export interface ConceptTagProps {
    name: string;
    description?: string;
    className?: string;
}

export function ConceptTag({ name, description, className }: ConceptTagProps) {
    return (
        <div className={clsx(styles.tag, className)} title={description}>
            <span className={styles.name}>{name}</span>
        </div>
    );
}

export interface ConceptCardProps {
    name: string;
    slug: string;
    description: string;
}

export function ConceptCard({ name, slug, description }: ConceptCardProps) {
    return (
        <div className={styles.card}>
            <div className={styles.cardHeader}>
                <span className={styles.cardName}>{name}</span>
                <span className={styles.cardSlug}>{slug}</span>
            </div>
            <p className={styles.cardDescription}>{description}</p>
        </div>
    );
}

export default ConceptTag;
