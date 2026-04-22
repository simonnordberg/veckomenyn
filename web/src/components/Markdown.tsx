import { createElement } from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

type Variant = "default" | "compact";

type Props = {
  source: string;
  variant?: Variant;
  className?: string;
  // Strip a single leading "# <heading>" if its text matches this string
  // (case-insensitive, whitespace-trimmed). Used for recipe bodies whose
  // first line repeats the dish name shown by the caller. Unmatched h1s
  // are preserved — no content is silently dropped.
  stripLeadingHeading?: string;
  // Demote every heading by N levels so a recipe's "## Ingredients" isn't
  // as big as the page title. 0 means render as-is.
  headingShift?: 0 | 1 | 2 | 3;
};

// Markdown renders recipe bodies, retrospectives, and assistant chat text
// through the Tailwind typography plugin.
export function Markdown({
  source,
  variant = "default",
  className = "",
  stripLeadingHeading,
  headingShift = 0,
}: Props) {
  if (!source) return null;

  let text = source;
  if (stripLeadingHeading) {
    text = stripMatchingLeadingH1(text, stripLeadingHeading);
  }

  const size = variant === "compact" ? "prose-sm" : "prose-base";
  const components = headingShift > 0 ? shiftedHeadingComponents(headingShift) : undefined;

  // break-words lets inline <code> product codes and long tokens wrap
  // instead of pushing the container wider than its parent (chat drawer).
  return (
    <div
      className={`prose prose-stone dark:prose-invert ${size} max-w-none break-words ${className}`.trim()}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {text}
      </ReactMarkdown>
    </div>
  );
}

function stripMatchingLeadingH1(source: string, expected: string): string {
  const m = source.match(/^\s*#\s+([^\n]+)\n+/);
  if (!m) return source;
  const normalize = (s: string) => s.trim().toLowerCase().replace(/\s+/g, " ");
  if (normalize(m[1]) !== normalize(expected)) return source;
  return source.slice(m[0].length);
}

function shiftedHeadingComponents(shift: number): Components {
  const at = (lvl: number) => `h${Math.min(6, lvl + shift)}`;
  const render =
    (lvl: number) =>
    // biome-ignore lint/suspicious/noExplicitAny: react-markdown heading props
    ({ node: _n, ...rest }: any) =>
      createElement(at(lvl), rest);
  return {
    h1: render(1),
    h2: render(2),
    h3: render(3),
    h4: render(4),
    h5: render(5),
  };
}
