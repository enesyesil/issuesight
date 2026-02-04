/**
 * Parses tutorial markdown into prerequisite concepts and step-by-step sections
 * for the checklist-style tutorial viewer.
 */

export interface TutorialStep {
  id: string;
  title: string;
  content: string;
}

export interface TutorialSection {
  id: string;
  title: string;
  content: string;
}

export interface ParsedTutorial {
  prerequisites: string[];
  sections: TutorialSection[];
  steps: TutorialStep[];
}

const PREREQUISITE_HEADINGS = [
  'prerequisites',
  'prerequisite concepts',
  'concepts',
  'key concepts',
  'before you begin',
  'before we start',
  'what you need',
  'what you\'ll need',
  'required knowledge',
  'prior knowledge',
  'assumed knowledge',
  'background',
  'you should know',
];

const SECTION_HEADINGS = [
  'context',
  'plan',
  'assumptions',
  'testing',
  'pitfalls',
  'summary',
];

function normalizeHeading(text: string): string {
  return text.replace(/#/g, '').trim().toLowerCase();
}

function extractListItems(block: string): string[] {
  const items: string[] = [];
  const seen = new Set<string>();
  const add = (s: string) => {
    const t = s.trim();
    if (t && t.length <= 80 && !seen.has(t.toLowerCase())) {
      seen.add(t.toLowerCase());
      items.push(t);
    }
  };
  const lines = block.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    // Match **Bold label**: or **Bold label** at start of line – use only the label as concept name
    const boldMatch = trimmed.match(/^\*\*(.+?)\*\*\s*:?\s*/);
    if (boldMatch) {
      add(boldMatch[1].trim());
      continue;
    }
    // Match - item or * item or 1. item
    const listMatch = trimmed.match(/^[-*]\s+(.+)$/) || trimmed.match(/^\d+\.\s+(.+)$/);
    if (listMatch) {
      const rest = listMatch[1].trim();
      // One line can be "Concept A, Concept B" – split by comma
      if (rest.includes(',')) {
        rest.split(',').forEach((part) => add(part));
      } else {
        add(rest);
      }
      continue;
    }
    // Line like "Concepts: X, Y, Z" or "Keywords: A, B"
    const labelMatch = trimmed.match(/^(?:concepts?|keywords?|terms?)\s*[:\-]\s*(.+)$/i);
    if (labelMatch) {
      labelMatch[1].split(',').forEach((part) => add(part));
      continue;
    }
    // Short non-list line as single concept (skip long lines or full sentences)
    if (
      trimmed.length <= 60 &&
      !trimmed.startsWith('#') &&
      !trimmed.startsWith('```') &&
      !trimmed.endsWith('.')
    ) {
      add(trimmed);
    }
  }
  return items;
}

/**
 * Splits markdown by ## (or first #) headings and returns blocks with their titles.
 */
type HeadingBlock = {
  title: string;
  normalizedTitle: string;
  content: string;
};

function splitByHeadings(markdown: string): HeadingBlock[] {
  const blocks: HeadingBlock[] = [];
  // Normalize: treat single # as ## for first heading so we get one intro block
  const normalized = markdown.trimStart().replace(/^(#)\s/m, '## ');
  const parts = normalized.split(/(?=^##\s)/m);
  for (const part of parts) {
    const trimmed = part.trim();
    if (!trimmed) continue;
    const firstLineEnd = trimmed.indexOf('\n');
    const firstLine = firstLineEnd === -1 ? trimmed : trimmed.slice(0, firstLineEnd);
    const rawTitle = firstLine.replace(/^##\s*/, '').trim();
    const normalizedTitle = normalizeHeading(rawTitle);
    const content = firstLineEnd === -1 ? '' : trimmed.slice(firstLineEnd + 1).trim();
    blocks.push({ title: rawTitle, normalizedTitle, content });
  }
  return blocks;
}

/**
 * Parses tutorial markdown into prerequisites and steps.
 * - Prerequisites: section whose title matches "Prerequisites", "Concepts", etc.; list items become concepts.
 * - Steps: every other ## section becomes a step card (intro + numbered steps).
 */
export function parseTutorialContent(markdown: string): ParsedTutorial {
  const prerequisites: string[] = [];
  const sections: TutorialSection[] = [];
  const steps: TutorialStep[] = [];
  const blocks = splitByHeadings(markdown);

  for (const block of blocks) {
    const titleLower = block.normalizedTitle;
    const isPrereq =
      PREREQUISITE_HEADINGS.some((h) => titleLower === h || titleLower.startsWith(h)) ||
      titleLower.includes('prerequisite') ||
      titleLower.includes('concept');
    if (isPrereq) {
      const items = extractListItems(block.content);
      prerequisites.push(...items);
      continue;
    }
    const isStep = titleLower.startsWith('step');
    if (isStep) {
      const id = `step-${steps.length + 1}`;
      steps.push({
        id,
        title: cleanStepTitle(block.title) || `Step ${steps.length + 1}`,
        content: block.content,
      });
      continue;
    }

    const isSection =
      SECTION_HEADINGS.some((h) => titleLower === h || titleLower.startsWith(h)) ||
      SECTION_HEADINGS.some((h) => titleLower.includes(h));
    if (isSection) {
      const id = `section-${sections.length + 1}`;
      sections.push({
        id,
        title: titleCase(block.title || 'Section'),
        content: block.content,
      });
      continue;
    }

    const id = `step-${steps.length + 1}`;
    steps.push({
      id,
      title: block.title ? titleCase(block.title) : `Step ${steps.length + 1}`,
      content: block.content,
    });
  }

  // If no ## blocks, treat whole content as a single step (e.g. raw markdown with # title only)
  if (steps.length === 0 && markdown.trim()) {
    const firstLineEnd = markdown.indexOf('\n');
    const firstLine = firstLineEnd === -1 ? markdown : markdown.slice(0, firstLineEnd);
    const title = firstLine.replace(/^#+\s*/, '').trim() || 'Introduction';
    const content = firstLineEnd === -1 ? '' : markdown.slice(firstLineEnd + 1).trim();
    steps.push({ id: 'step-1', title, content });
  }

  return { prerequisites, sections, steps };
}

function titleCase(text: string): string {
  if (!text) return '';
  return text.charAt(0).toUpperCase() + text.slice(1);
}

function cleanStepTitle(title: string): string {
  if (!title) return '';
  const trimmed = title.trim();
  const match = trimmed.match(/^step\s*\d*\s*[-–—:]?\s*(.*)$/i);
  if (match && match[1]) {
    return match[1].trim();
  }
  return trimmed;
}
