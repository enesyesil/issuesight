'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Check, ChevronRight, BookOpen, Copy } from 'lucide-react';
import { cn, slugify } from '@/lib/utils';
import {
  parseTutorialContent,
  type ParsedTutorial,
  type TutorialSection,
  type TutorialStep,
} from '@/lib/tutorialParser';
import type { Components } from 'react-markdown';
import type { Concept } from '@/lib/types';

const STORAGE_KEY_PREFIX = 'tutorial-progress-';

interface CodeProps {
  node?: unknown;
  inline?: boolean;
  className?: string;
  children?: React.ReactNode;
}

const markdownComponents: Components = {
  code({ inline, className, children, ...props }: CodeProps) {
    const match = /language-(\w+)/.exec(className || '');
    return !inline && match ? (
      <SyntaxHighlighter style={vscDarkPlus} language={match[1]} PreTag="div">
        {String(children).replace(/\n$/, '')}
      </SyntaxHighlighter>
    ) : (
      <code {...props} className={className}>
        {children}
      </code>
    );
  },
};

export interface TutorialViewerProps {
  content: string;
  tutorialId: string;
  concepts?: Concept[];
  mode?: 'issue' | 'concept';
}

function getStoredProgress(tutorialId: string): Set<string> {
  if (typeof window === 'undefined') return new Set();
  try {
    const raw = localStorage.getItem(STORAGE_KEY_PREFIX + tutorialId);
    if (!raw) return new Set();
    const arr = JSON.parse(raw) as string[];
    return new Set(Array.isArray(arr) ? arr : []);
  } catch {
    return new Set();
  }
}

function setStoredProgress(tutorialId: string, completedIds: Set<string>) {
  try {
    localStorage.setItem(
      STORAGE_KEY_PREFIX + tutorialId,
      JSON.stringify([...completedIds])
    );
  } catch {
    // ignore
  }
}

export function TutorialViewer({
  content,
  tutorialId,
  concepts,
  mode = 'issue',
}: TutorialViewerProps) {
  const parsed = useMemo((): ParsedTutorial => {
    return parseTutorialContent(content || '# No content available');
  }, [content]);

  const [completedStepIds, setCompletedStepIds] = useState<Set<string>>(() =>
    getStoredProgress(tutorialId)
  );
  const [copyStatus, setCopyStatus] = useState<'idle' | 'success' | 'error'>('idle');

  useEffect(() => {
    setCompletedStepIds(getStoredProgress(tutorialId));
  }, [tutorialId]);

  const toggleStep = useCallback(
    (stepId: string) => {
      setCompletedStepIds((prev) => {
        const next = new Set(prev);
        if (next.has(stepId)) {
          next.delete(stepId);
        } else {
          next.add(stepId);
        }
        setStoredProgress(tutorialId, next);
        return next;
      });
    },
    [tutorialId]
  );

  const handleCopyMarkdown = useCallback(async () => {
    if (typeof window === 'undefined' || !navigator?.clipboard) {
      setCopyStatus('error');
      setTimeout(() => setCopyStatus('idle'), 2000);
      return;
    }

    try {
      await navigator.clipboard.writeText(content || '');
      setCopyStatus('success');
    } catch {
      setCopyStatus('error');
    }

    setTimeout(() => setCopyStatus('idle'), 2000);
  }, [content]);

  const { prerequisites, sections, steps } = parsed;
  const hasSteps = steps.length > 0;
  const hasSections = sections.length > 0 && mode === 'issue';
  const completedCount = completedStepIds.size;
  const progress = hasSteps ? Math.round((completedCount / steps.length) * 100) : 0;
  const wordCount = (content || '').trim().split(/\s+/).filter(Boolean).length;
  const readTime = Math.max(1, Math.ceil(wordCount / 220));

  const displayConcepts = useMemo(() => {
    if (concepts && concepts.length > 0) {
      return concepts.map((c) => ({
        label: c.name || c.slug,
        slug: c.slug,
      }));
    }
    return prerequisites.map((name) => ({
      label: name,
      slug: slugify(name),
    }));
  }, [concepts, prerequisites]);

  const hasConcepts = displayConcepts.length > 0;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex justify-end">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleCopyMarkdown}
          className="gap-2"
          aria-label="Copy tutorial as markdown"
        >
          <Copy className="h-4 w-4" />
          {copyStatus === 'success'
            ? 'Copied Markdown'
            : copyStatus === 'error'
              ? 'Copy Failed'
              : 'Copy Markdown'}
        </Button>
      </div>

      {/* Prerequisite concepts – above the step cards; each links to concept detail page */}
      {hasConcepts && (
        <Card className="border-border bg-card dark:bg-card">
          <CardHeader className="pb-2">
            <div className="flex items-center gap-2 text-foreground">
              <BookOpen className="h-5 w-5 text-primary" />
              <h2 className="text-lg font-semibold">Prerequisite concepts</h2>
            </div>
            <p className="text-sm text-muted-foreground">
              Review these before starting. Click a concept to learn more.
            </p>
          </CardHeader>
          <CardContent className="pt-0">
            <ul className="flex flex-wrap gap-2">
              {displayConcepts.map((concept, i) => (
                <li key={`${concept.slug}-${i}`}>
                  <Link
                    href={`/concepts/${concept.slug}`}
                    className="inline-flex items-center rounded-full border border-border bg-muted/50 px-3 py-1.5 text-sm text-foreground hover:bg-muted hover:border-primary/30 transition-colors"
                  >
                    {concept.label}
                  </Link>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {hasSections && (
        <div className="grid gap-4 md:grid-cols-2 items-stretch">
          {sections.map((section) => (
            <SectionCard key={section.id} section={section} />
          ))}
        </div>
      )}

      {/* Step-by-step checklist cards */}
      {hasSteps ? (
        <div className="space-y-4">
          <div className="rounded-2xl border border-border/70 bg-muted/30 p-4">
            <div className="grid gap-4 md:grid-cols-[1.2fr_1fr]">
              <div className="space-y-2">
                <p className="text-xs uppercase tracking-wide text-muted-foreground">
                  {mode === 'concept' ? "What you'll learn" : 'Steps'}
                </p>
                <h2 className="text-lg font-semibold text-foreground">
                  {mode === 'concept' ? 'Step-by-step walkthrough' : 'Execution checklist'}
                </h2>
                <p className="text-sm text-muted-foreground">
                  {mode === 'concept'
                    ? 'Follow the sequence to build intuition quickly.'
                    : 'Complete each step to resolve the issue with confidence.'}
                </p>
                <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                  <span>{steps.length} steps</span>
                  <span>·</span>
                  <span>{readTime} min read</span>
                </div>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{completedCount} completed</span>
                  <span>{steps.length} total</span>
                </div>
                <div className="h-2 rounded-full bg-muted/60">
                  <div
                    className="h-2 rounded-full bg-primary transition-all"
                    style={{ width: `${progress}%` }}
                  />
                </div>
                <div className="grid grid-cols-2 gap-2 pt-1 sm:grid-cols-3 lg:grid-cols-4">
                  {steps.map((step, index) => (
                    <div
                      key={`step-pill-${step.id}`}
                      className={cn(
                        'flex items-center gap-2 rounded-full border px-3 py-1 text-xs',
                        completedStepIds.has(step.id)
                          ? 'border-primary/40 bg-primary/10 text-primary'
                          : 'border-border/70 bg-background text-muted-foreground'
                      )}
                    >
                      <span className="font-semibold">{index + 1}</span>
                      <span className="max-w-[120px] truncate">{step.title}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>

          {steps.map((step, index) => (
            <StepCard
              key={step.id}
              step={step}
              index={index + 1}
              isCompleted={completedStepIds.has(step.id)}
              onToggle={() => toggleStep(step.id)}
            />
          ))}
        </div>
      ) : (
        <Card className="border-border bg-card dark:bg-card">
          <CardContent className="pt-6 prose dark:prose-invert max-w-none">
            <ReactMarkdown components={markdownComponents}>
              {content}
            </ReactMarkdown>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

interface SectionCardProps {
  section: TutorialSection;
}

function SectionCard({ section }: SectionCardProps) {
  return (
    <Card className="border-border bg-card dark:bg-card h-full">
      <CardHeader className="pb-3">
        <p className="text-xs uppercase tracking-wide text-muted-foreground">
          Section
        </p>
        <h3 className="text-sm font-semibold text-foreground">
          {section.title}
        </h3>
      </CardHeader>
      <CardContent className="pt-0 prose prose-sm dark:prose-invert max-w-none">
        <ReactMarkdown components={markdownComponents}>
          {section.content}
        </ReactMarkdown>
      </CardContent>
    </Card>
  );
}

interface StepCardProps {
  step: TutorialStep;
  index: number;
  isCompleted: boolean;
  onToggle: () => void;
}

function StepCard({ step, index, isCompleted, onToggle }: StepCardProps) {
  const [expanded, setExpanded] = useState(index === 1 || isCompleted);

  return (
    <Card
      className={cn(
        'border transition-all duration-200 relative overflow-hidden',
        isCompleted
          ? 'border-primary/30 bg-primary/5 dark:bg-primary/10'
          : 'border-border bg-card dark:bg-card hover:-translate-y-0.5 hover:shadow-md'
      )}
    >
      <div
        className={cn(
          "pointer-events-none absolute inset-y-0 left-0 w-1 before:content-[''] before:absolute before:inset-y-0 before:left-0 before:w-1 before:bg-border/60",
          isCompleted && 'before:bg-primary/40'
        )}
      />
      <CardHeader className="cursor-pointer py-5" onClick={() => setExpanded((e) => !e)}>
        <div className="flex items-start gap-4">
          <div className="flex flex-col items-center gap-2">
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onToggle();
              }}
              className={cn(
                'mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border-2 transition-colors',
                isCompleted
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-muted-foreground/50 bg-transparent hover:border-primary/50'
              )}
              aria-label={isCompleted ? 'Mark step incomplete' : 'Mark step complete'}
            >
              {isCompleted ? <Check className="h-4 w-4" /> : null}
            </button>
            <span className="text-[0.65rem] font-semibold uppercase tracking-wide text-muted-foreground">
              {index}
            </span>
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-muted-foreground">
                Step {index}
              </span>
              <span className="font-semibold text-foreground">{step.title}</span>
            </div>
            <button
              type="button"
              className="mt-1 flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
              onClick={(e) => {
                e.stopPropagation();
                setExpanded((e) => !e);
              }}
            >
              {expanded ? 'Collapse' : 'Expand'}
              <ChevronRight
                className={cn('h-4 w-4 transition-transform', expanded && 'rotate-90')}
              />
            </button>
          </div>
        </div>
      </CardHeader>
      {expanded && (
        <CardContent className="border-t border-border pt-4 prose prose-sm dark:prose-invert max-w-none">
          <ReactMarkdown components={markdownComponents}>
            {step.content}
          </ReactMarkdown>
        </CardContent>
      )}
    </Card>
  );
}
