/**
 * Lightweight Markdown-to-HTML renderer with basic XSS protection.
 *
 * - All raw text is HTML-escaped before inline formatting is applied.
 * - Only http://, https://, and # anchors are allowed in links.
 * - No raw HTML passthrough — anything that looks like HTML tags is escaped.
 */

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/** Process inline Markdown within a single line of text. */
function processInline(text: string): string {
  // Links — validate href scheme
  text = text.replace(
    /\[([^\]]+?)\]\(([^)]+?)\)/g,
    (_match, linkText: string, url: string) => {
      if (!/^https?:\/\//.test(url) && !/^#/.test(url)) {
        // Untrusted scheme — render as plain text
        return `<span>${escapeHtml(linkText)} (${escapeHtml(url)})</span>`;
      }
      return `<a href="${escapeHtml(url)}" target="_blank" rel="noopener noreferrer">${escapeHtml(linkText)}</a>`;
    }
  );

  // Bold
  text = text.replace(/\*\*([^*]+?)\*\*/g, "<strong>$1</strong>");

  // Italic (must come after bold)
  text = text.replace(/\*([^*]+?)\*/g, "<em>$1</em>");

  // Inline code
  text = text.replace(/`([^`]+?)`/g, "<code>$1</code>");

  return text;
}

/**
 * Convert Markdown text to safe HTML.
 *
 * Supported syntax:
 *   - Headers (# to ######)
 *   - Bold (**text**)
 *   - Italic (*text*)
 *   - Inline code (`code`)
 *   - Fenced code blocks (```…```)
 *   - Unordered lists (- / *)
 *   - Ordered lists (1. 2. …)
 *   - Links ([text](url))
 *   - Paragraphs
 */
export function renderMarkdown(md: string): string {
  const lines = md.split("\n");
  const result: string[] = [];
  let i = 0;
  let inCodeBlock = false;
  let codeBlockContent: string[] = [];

  while (i < lines.length) {
    const line = lines[i];

    // Fenced code blocks
    if (line.startsWith("```")) {
      if (inCodeBlock) {
        result.push(
          `<pre><code>${escapeHtml(codeBlockContent.join("\n"))}</code></pre>`
        );
        codeBlockContent = [];
        inCodeBlock = false;
      } else {
        inCodeBlock = true;
      }
      i++;
      continue;
    }

    if (inCodeBlock) {
      codeBlockContent.push(line);
      i++;
      continue;
    }

    // Empty line
    if (!line.trim()) {
      result.push("");
      i++;
      continue;
    }

    // Headers
    const hMatch = line.match(/^(#{1,6})\s+(.*)$/);
    if (hMatch) {
      const level = hMatch[1].length;
      const content = processInline(escapeHtml(hMatch[2]));
      result.push(`<h${level}>${content}</h${level}>`);
      i++;
      continue;
    }

    // List items (unordered or ordered)
    const ulMatch = line.match(/^(\s*)[-*]\s+(.*)$/);
    const olMatch = line.match(/^(\s*)\d+\.\s+(.*)$/);
    if (ulMatch || olMatch) {
      const isUl = !!ulMatch;
      const content = isUl ? ulMatch[2] : olMatch![2];

      // Determine if we need to open a new list
      const prevIsListItem =
        result.length > 0 && result[result.length - 1].startsWith("<li>");
      const prevIsListTag =
        result.length > 0 &&
        /^<(ul|ol)[\s>]/.test(result[result.length - 1]);

      if (!prevIsListItem && !prevIsListTag) {
        result.push(`<${isUl ? "ul" : "ol"}>`);
      }

      result.push(`<li>${processInline(escapeHtml(content))}</li>`);

      // Close list if next line is not a list item
      const nextLine = lines[i + 1];
      const nextIsList =
        nextLine &&
        (/^(\s*)[-*]\s+/.test(nextLine) || /^(\s*)\d+\.\s+/.test(nextLine));
      if (!nextIsList) {
        result.push(`</${isUl ? "ul" : "ol"}>`);
      }

      i++;
      continue;
    }

    // Regular paragraph
    result.push(`<p>${processInline(escapeHtml(line))}</p>`);
    i++;
  }

  // Close any unclosed code block
  if (inCodeBlock) {
    result.push(
      `<pre><code>${escapeHtml(codeBlockContent.join("\n"))}</code></pre>`
    );
  }

  return result.join("\n");
}
