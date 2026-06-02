export const modelColorMapper: Record<string, string> = {
  // OpenAI & Azure OpenAI
  "gpt-4": "purple-600",

  dalle: "green-600",
  "dall-e-2": "green-600",
  "dall-e-3": "purple-700",

  whisper: "gray-300",
  tts: "gray-300",
  openai: "gray-300",
  azure: "gray-300",
  xai: "gray-900",
  xiaomi: "orange-500",
  mimo: "orange-500",

  // Anthropic Claude
  "claude-3": "orange-500",
  claude: "orange-400",
  anthropic: "orange-400",
  minimax: "emerald-500",

  // Stable Diffusion
  "stable-diffusion": "gray-400",
  stablediffusion: "gray-400",
  stability: "gray-400",

  // Google Gemini & Gemma
  palm: "red-500",
  gemini: "red-500",
  gemma: "red-500",

  // DeepSeek
  deepseek: "blue-700",
  grok: "gray-900",

  // ChatGLM
  zhipu: "lime-500",
  glm: "lime-500",

  // Meta LLaMA
  llama: "sky-400",

  // OpenRouter
  openrouter: "purple-600",
};

const unknownColors = [
  "gray-700",
  "indigo-600",
  "green-500",
  "green-600",
  "purple-600",
  "purple-700",
  "orange-400",
  "blue-400",
  "red-500",
  "blue-700",
  "lime-500",
  "sky-400",
];

export function getUnknownModelColor(model: string): string {
  const char = model.length > 0 ? model[0] : "A";
  const code = char.charCodeAt(0);

  return unknownColors[code % unknownColors.length];
}

export function getModelColor(model: string): string {
  for (const key in modelColorMapper) {
    if (model.toLowerCase().includes(key)) {
      return modelColorMapper[key];
    }
  }

  return getUnknownModelColor(model);
}
