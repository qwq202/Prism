import { FileArray, FileObject } from "@/api/file.ts";

export type MessageFilePart = {
  type: "file";
  file: FileObject;
};

export type MessageTextPart = {
  type: "text";
  content: string;
};

export type MessagePart = MessageFilePart | MessageTextPart;

const fileBlockPattern =
  /```file\r?\n\[\[([^\r\n]*)]]\r?\n([\s\S]*?)\r?\n```/g;

export function getFile(file: FileObject): string {
  return `\`\`\`file
[[${file.name}]]
${file.content}
\`\`\``;
}

export function formatMessage(files: FileArray, message: string): string {
  message = message.trim();

  const data = files.map((file) => getFile(file)).join("\n\n");
  return files.length > 0 ? `${data}\n\n${message}` : message;
}

export function getFileMarkdown(file: FileObject): string {
  return `[[${file.name}]]\n${file.content}`;
}

export function splitFileMessage(message: string): MessagePart[] {
  const parts: MessagePart[] = [];
  let lastIndex = 0;
  fileBlockPattern.lastIndex = 0;

  for (const match of message.matchAll(fileBlockPattern)) {
    const index = match.index ?? 0;
    if (index > lastIndex) {
      parts.push({
        type: "text",
        content: message.slice(lastIndex, index),
      });
    }

    parts.push({
      type: "file",
      file: {
        name: match[1],
        content: match[2],
      },
    });

    lastIndex = index + match[0].length;
  }

  if (lastIndex < message.length) {
    parts.push({
      type: "text",
      content: message.slice(lastIndex),
    });
  }

  return parts.length > 0 ? parts : [{ type: "text", content: message }];
}

export function filterMessage(message: string): string {
  return splitFileMessage(message)
    .filter((part): part is MessageTextPart => part.type === "text")
    .map((part) => part.content)
    .join("")
    .trim();
}

export function extractMessage(
  message: string,
  length: number = 50,
  flow: string = "...",
) {
  return message.length > length ? message.slice(0, length) + flow : message;
}

export function escapeRegExp(str: string): string {
  // convert \n to [enter], \t to [tab], \r to [return], \s to [space], \" to [quote], \' to [single-quote]
  return str
    .replace(/\\n/g, "\n")
    .replace(/\\t/g, "\t")
    .replace(/\\r/g, "\r")
    .replace(/\\s/g, " ")
    .replace(/\\"/g, '"')
    .replace(/\\'/g, "'");
}

export function handleLine(
  data: string,
  max_line: number,
  end?: boolean,
): string {
  const segment = data.split("\n");
  const line = segment.length;
  if (line > max_line) {
    return end ?? true
      ? segment.slice(line - max_line).join("\n")
      : segment.slice(0, max_line).join("\n");
  } else {
    return data;
  }
}

export function handleGenerationData(data: string): string {
  data = data
    .replace(/{\s*"result":\s*{/g, "")
    .trim()
    .replace(/}\s*$/g, "");
  return handleLine(escapeRegExp(data), 6);
}

export function getReadableNumber(
  num: number,
  fixed?: number,
  must_k?: boolean,
): string {
  if (num >= 1e9) return (num / 1e9).toFixed(fixed) + "b";
  if (num >= 1e6) return (num / 1e6).toFixed(fixed) + "m";
  if (num >= 1e3 || (num !== 0 && must_k))
    return (num / 1e3).toFixed(fixed) + "k";
  return num.toFixed(0);
}
