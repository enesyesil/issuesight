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
  "what you'll need",
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

type HeadingBlock = {
  title: string;
  normalizedTitle: string;
  content: string;
};

function normalizeHeading(text: string): string {
  return text.replace(/#/g, '').trim().toLowerCase();
}

function parseV2SectionNumber(title: string): number | null {
  const match = title.trim().match(/^(\d+)\)/);
  if (!match) return null;
  const number = Number.parseInt(match[1], 10);
  return Number.isNaN(number) ? null : number;
}

function splitByHeadings(markdown: string): HeadingBlock[] {
  const blocks: HeadingBlock[] = [];
  const normalized = markdown.trimStart().replace(/^(#)\s/m, '## ');
  const parts = normalized.split(/(?=^##\s)/m);

  for (const part of parts) {
    const trimmed = part.trim();
    if (!trimmed) continue;

    const firstLineEnd = trimmed.indexOf('\n');
    const firstLine = firstLineEnd === -1 ? trimmed : trimmed.slice(0, firstLineEnd);
    const rawTitle = firstLine.replace(/^##\s*/, '').trim();
    const content = firstLineEnd === -1 ? '' : trimmed.slice(firstLineEnd + 1).trim();

    blocks.push({
      title: rawTitle,
      normalizedTitle: normalizeHeading(rawTitle),
      content,
    });
  }

  return blocks;
}

function splitBySubheadings(content: string): HeadingBlock[] {
  const blocks: HeadingBlock[] = [];
  const parts = content.trim().split(/(?=^###\s)/m);

  for (const part of parts) {
    const trimmed = part.trim();
    if (!trimmed.startsWith('### ')) continue;

    const firstLineEnd = trimmed.indexOf('\n');
    const firstLine = firstLineEnd === -1 ? trimmed : trimmed.slice(0, firstLineEnd);
    const rawTitle = firstLine.replace(/^###\s*/, '').trim();
    const body = firstLineEnd === -1 ? '' : trimmed.slice(firstLineEnd + 1).trim();

    blocks.push({
      title: rawTitle,
      normalizedTitle: normalizeHeading(rawTitle),
      content: body,
    });
  }

  return blocks;
}

function extractListItems(block: string): string[] {
  const items: string[] = [];
  const seen = new Set<string>();
  const add = (s: string) => {
    const t = s.trim();
    if (!t || t.length > 80) return;
    const key = t.toLowerCase();
    if (seen.has(key)) return;
    seen.add(key);
    items.push(t);
  };

  const lines = block.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    const boldMatch = trimmed.match(/^\*\*(.+?)\*\*\s*:?\s*/);
    if (boldMatch) {
      add(boldMatch[1].trim());
      continue;
    }

    const listMatch = trimmed.match(/^[-*]\s+(.+)$/) || trimmed.match(/^\d+\.\s+(.+)$/);
    if (listMatch) {
      const rest = listMatch[1].trim();
      const conceptLabel =
        rest.match(/^\*\*concept:\*\*\s*(.+)$/i) ||
        rest.match(/^\*\*concept\*\*:\s*(.+)$/i) ||
        rest.match(/^concept:\s*(.+)$/i);
      if (conceptLabel && conceptLabel[1]) {
        add(conceptLabel[1]);
        continue;
      }

      if (rest.includes(',')) {
        rest.split(',').forEach((part) => add(part));
      } else {
        add(rest);
      }
      continue;
    }

    const labelMatch = trimmed.match(/^(?:concepts?|keywords?|terms?)\s*[:\-]\s*(.+)$/i);
    if (labelMatch) {
      labelMatch[1].split(',').forEach((part) => add(part));
      continue;
    }

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

function extractConceptsFromSectionOne(content: string): string[] {
  const lines = content.split('\n');
  const concepts: string[] = [];
  const seen = new Set<string>();
  let inConceptSubsection = false;

  const add = (value: string) => {
    const clean = value.trim();
    if (!clean || clean.length > 80) return;
    const key = clean.toLowerCase();
    if (seen.has(key)) return;
    seen.add(key);
    concepts.push(clean);
  };

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    if (trimmed.startsWith('### ')) {
      const title = normalizeHeading(trimmed.replace(/^###\s*/, ''));
      if (title.startsWith('1.1 concepts')) {
        inConceptSubsection = true;
        continue;
      }
      if (inConceptSubsection) break;
      continue;
    }

    if (!inConceptSubsection) continue;

    const listMatch = trimmed.match(/^[-*]\s+(.+)$/) || trimmed.match(/^\d+\.\s+(.+)$/);
    if (!listMatch) continue;

    const value = listMatch[1].trim();
    const conceptMatch =
      value.match(/^\*\*concept:\*\*\s*(.+)$/i) ||
      value.match(/^\*\*concept\*\*:\s*(.+)$/i) ||
      value.match(/^concept:\s*(.+)$/i);

    if (conceptMatch && conceptMatch[1]) {
      add(conceptMatch[1]);
    }
  }

  return concepts;
}

function extractStepsFromSectionFive(content: string): TutorialStep[] {
  const steps: TutorialStep[] = [];
  const blocks = splitBySubheadings(content);

  for (const block of blocks) {
    if (!block.normalizedTitle.startsWith('step')) continue;
    const id = `step-${steps.length + 1}`;
    steps.push({
      id,
      title: cleanStepTitle(block.title) || `Step ${steps.length + 1}`,
      content: block.content,
    });
  }

  return steps;
}

function parseV2Tutorial(blocks: HeadingBlock[], markdown: string): ParsedTutorial {
  const prerequisites: string[] = [];
  const sections: TutorialSection[] = [];
  const steps: TutorialStep[] = [];

  for (const block of blocks) {
    const sectionNumber = parseV2SectionNumber(block.title);
    if (sectionNumber === null) continue;

    if (sectionNumber === 1) {
      prerequisites.push(...extractConceptsFromSectionOne(block.content));
      sections.push({
        id: `section-${sections.length + 1}`,
        title: titleCase(block.title || 'Section 1'),
        content: block.content,
      });
      continue;
    }

    if (sectionNumber === 5) {
      const parsedSteps = extractStepsFromSectionFive(block.content);
      if (parsedSteps.length > 0) {
        steps.push(...parsedSteps);
      } else if (block.content.trim()) {
        steps.push({
          id: `step-${steps.length + 1}`,
          title: 'Implementation Plan',
          content: block.content,
        });
      }
      continue;
    }

    sections.push({
      id: `section-${sections.length + 1}`,
      title: titleCase(block.title || `Section ${sectionNumber}`),
      content: block.content,
    });
  }

  const dedupedPrerequisites = Array.from(
    new Map(prerequisites.map((item) => [item.toLowerCase(), item])).values()
  );

  if (steps.length === 0 && markdown.trim()) {
    const firstLineEnd = markdown.indexOf('\n');
    const firstLine = firstLineEnd === -1 ? markdown : markdown.slice(0, firstLineEnd);
    const title = firstLine.replace(/^#+\s*/, '').trim() || 'Introduction';
    const content = firstLineEnd === -1 ? '' : markdown.slice(firstLineEnd + 1).trim();
    steps.push({ id: 'step-1', title, content });
  }

  return { prerequisites: dedupedPrerequisites, sections, steps };
}

function parseLegacyTutorial(blocks: HeadingBlock[], markdown: string): ParsedTutorial {
  const prerequisites: string[] = [];
  const sections: TutorialSection[] = [];
  const steps: TutorialStep[] = [];

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

  if (steps.length === 0 && markdown.trim()) {
    const firstLineEnd = markdown.indexOf('\n');
    const firstLine = firstLineEnd === -1 ? markdown : markdown.slice(0, firstLineEnd);
    const title = firstLine.replace(/^#+\s*/, '').trim() || 'Introduction';
    const content = firstLineEnd === -1 ? '' : markdown.slice(firstLineEnd + 1).trim();
    steps.push({ id: 'step-1', title, content });
  }

  return { prerequisites, sections, steps };
}

/**
 * Parses tutorial markdown into prerequisites and steps.
 */
export function parseTutorialContent(markdown: string): ParsedTutorial {
  const blocks = splitByHeadings(markdown);
  const hasNumberedSections = blocks.some((block) => parseV2SectionNumber(block.title) !== null);

  if (hasNumberedSections) {
    return parseV2Tutorial(blocks, markdown);
  }
  return parseLegacyTutorial(blocks, markdown);
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
