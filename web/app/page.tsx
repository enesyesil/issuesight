import Link from 'next/link';
import { ArrowRight, Sparkles, Github } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import styles from './page.module.css';

export default function Home() {
  return (
    <div className={styles.page}>
      {/* Hero Section */}
      <section className={styles.hero}>
        <div className={styles.heroContent}>
          {/* Badge */}
          <div className={styles.badge}>
            <Sparkles size={16} />
            <span>AI-Powered Tutorial Generation</span>
          </div>

          {/* Title */}
          <h1 className={styles.title}>
            Transform GitHub Issues
            <br />
            into <span className={styles.titleAccent}>Learning Tutorials</span>
          </h1>

          {/* Subtitle */}
          <p className={styles.subtitle}>
            IssueSight analyzes GitHub issues and generates comprehensive, step-by-step
            tutorials. Turn complex problems into teachable moments.
          </p>

          {/* CTAs */}
          <div className={styles.ctas}>
            <Link href="/dashboard">
              <Button variant="primary" size="lg" icon={<ArrowRight size={18} />} iconPosition="right">
                Start Generating
              </Button>
            </Link>
            <a
              href="https://github.com/issuesight/issuesight"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Button variant="secondary" size="lg" icon={<Github size={18} />}>
                View on GitHub
              </Button>
            </a>
          </div>
        </div>
      </section>

      {/* Stats Section */}
      <section className={styles.stats}>
        <div className={styles.statsContainer}>
          <div className={styles.stat}>
            <span className={styles.statValue}>10K+</span>
            <span className={styles.statLabel}>Tutorials Generated</span>
          </div>
          <div className={styles.stat}>
            <span className={styles.statValue}>500+</span>
            <span className={styles.statLabel}>Active Users</span>
          </div>
          <div className={styles.stat}>
            <span className={styles.statValue}>95%</span>
            <span className={styles.statLabel}>Satisfaction Rate</span>
          </div>
        </div>
      </section>
    </div>
  );
}
