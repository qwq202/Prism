const protectedLatexTokenPrefix = "__PRISM_MARKDOWN_PROTECTED_";

type LatexMatch = {
  pre: string;
  body: string;
  post: string;
};

function isEscapedAt(text: string, index: number) {
  let slashCount = 0;
  for (let i = index - 1; i >= 0 && text[i] === "\\"; i -= 1) {
    slashCount += 1;
  }

  return slashCount % 2 === 1;
}

function findLatexDelimiterMatch(
  text: string,
  openDelimiter: string,
  closeDelimiter: string,
): LatexMatch | null {
  for (let start = 0; start <= text.length - openDelimiter.length; start += 1) {
    if (!text.startsWith(openDelimiter, start) || isEscapedAt(text, start)) {
      continue;
    }

    let depth = 1;
    for (
      let end = start + openDelimiter.length;
      end <= text.length - closeDelimiter.length;
      end += 1
    ) {
      const isOpen =
        text.startsWith(openDelimiter, end) && !isEscapedAt(text, end);
      const isClose =
        text.startsWith(closeDelimiter, end) && !isEscapedAt(text, end);

      if (!isOpen && !isClose) {
        continue;
      }

      depth += isOpen ? 1 : -1;
      if (depth === 0) {
        return {
          pre: text.slice(0, start),
          body: text.slice(start + openDelimiter.length, end),
          post: text.slice(end + closeDelimiter.length),
        };
      }

      end += (isOpen ? openDelimiter : closeDelimiter).length - 1;
    }
  }

  return null;
}

function replaceLatexDelimiters(
  content: string,
  openDelimiter: string,
  closeDelimiter: string,
  format: (body: string) => string,
) {
  let result = "";
  let remaining = content;

  while (remaining.length > 0) {
    const match = findLatexDelimiterMatch(
      remaining,
      openDelimiter,
      closeDelimiter,
    );
    if (!match) {
      result += remaining;
      break;
    }

    result += match.pre + format(match.body);
    remaining = match.post;
  }

  return result;
}

export function normalizeLatexDelimiters(content: string) {
  if (!/\\[[(]/.test(content)) {
    return content;
  }

  const protectedParts: string[] = [];
  const protect = (value: string) => {
    const token = `${protectedLatexTokenPrefix}${protectedParts.length}__`;
    protectedParts.push(value);
    return token;
  };

  const protectedContent = content
    .replace(/(```[\s\S]*?```|`[^`]*`)/g, protect)
    .replace(/\[[^\]]*]\([^)]*\)/g, protect);

  const normalizedBlocks = replaceLatexDelimiters(
    protectedContent,
    "\\[",
    "\\]",
    (body) => `\n$$\n${body.trim()}\n$$\n`,
  );
  const normalizedInline = replaceLatexDelimiters(
    normalizedBlocks,
    "\\(",
    "\\)",
    (body) => `$${body}$`,
  );

  return normalizedInline.replace(
    new RegExp(`${protectedLatexTokenPrefix}(\\d+)__`, "g"),
    (token, index: string) => protectedParts[Number(index)] ?? token,
  );
}
