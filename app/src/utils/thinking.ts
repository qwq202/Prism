export function stripThinkTags(content: string): string {
  return content.replace(/<\s*\/?\s*think\s*>/gi, "").trim();
}
