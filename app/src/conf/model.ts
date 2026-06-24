import { Model } from "@/api/types.tsx";

const OPENAI_DRAWING_MODELS = ["dalle", "dall-e-2", "dall-e-3", "gpt-image-1"];
const GEMINI_DRAWING_MODELS = [
  "gemini-2.5-flash-image",
  "gemini-3.1-flash-image",
  "gemini-3-pro-image",
];
export const DRAWING_MODEL_TAG = "image-generation";

export function getGrokModelName(id: string): string | null {
  const match = id.trim().match(/(?:^|\/)grok-(.+)$/i);
  if (!match) return null;

  const segments = match[1].split("-").filter(Boolean);
  const versionSegments: string[] = [];

  while (segments.length > 0 && /^\d+(?:\.\d+)?$/.test(segments[0])) {
    const segment = segments.shift();
    if (segment) versionSegments.push(segment);
  }

  const nameParts = [
    "Grok",
    versionSegments.length ? versionSegments.join(".") : "",
    ...segments.map((segment) =>
      segment.replace(/\b\w/g, (letter) => letter.toUpperCase()),
    ),
  ].filter(Boolean);

  return nameParts.join(" ");
}

export function normalizeModelDisplayNames<
  T extends Pick<Model, "id" | "name">,
>(models: T[]): T[] {
  return models.map((model) => {
    const grokName = getGrokModelName(model.id);
    const normalizedName = grokName ?? model.name.replace(/-/g, " ");
    if (normalizedName === model.name) return model;

    return {
      ...model,
      name: normalizedName,
    } as T;
  });
}

export function getModelFromId(market: Model[], id: string): Model | undefined {
  return market.find((model) => model.id === id);
}

export function isHighContextModel(market: Model[], id: string): boolean {
  const model = getModelFromId(market, id);
  return !!model && model.high_context;
}

export function supportsImageUpload(
  model?: Pick<Model, "vision_model"> | null,
): boolean {
  return !!model?.vision_model;
}

function matchesDrawingModelList(modelId: string, list: string[]): boolean {
  const normalized = modelId.trim().toLowerCase();
  return list.some(
    (model) => normalized === model || normalized.includes(model),
  );
}

export function isDrawingModel(
  model?: Pick<Model, "id" | "channel_type" | "drawing_model"> | string | null,
): boolean {
  if (!model) return false;

  const modelId = typeof model === "string" ? model : model.id;
  if (!modelId) return false;

  if (typeof model !== "string" && model.drawing_model) return true;

  const channelType =
    typeof model === "string" ? "" : (model.channel_type || "").toLowerCase();
  const isOpenAIImageModel = matchesDrawingModelList(
    modelId,
    OPENAI_DRAWING_MODELS,
  );
  const isGeminiImageModel = matchesDrawingModelList(
    modelId,
    GEMINI_DRAWING_MODELS,
  );

  switch (channelType) {
    case "openai":
    case "azure":
      return (
        isOpenAIImageModel && !modelId.toLowerCase().includes("gpt-4-dalle")
      );
    case "palm":
    case "gemini-enterprise-agent-platform":
      return isGeminiImageModel;
    case "":
      return (
        (isOpenAIImageModel &&
          !modelId.toLowerCase().includes("gpt-4-dalle")) ||
        isGeminiImageModel
      );
    default:
      return false;
  }
}

export function getResolvedModelTags(model: Model): string[] {
  let raw = [...(model.tag || [])];

  if (model.free && !raw.includes("free")) raw = ["free", ...raw];
  if (!model.free && raw.includes("free")) {
    raw = raw.filter((tag) => tag !== "free");
  }

  if (model.high_context && !raw.includes("high-context")) {
    raw = ["high-context", ...raw];
  }
  if (!model.high_context && raw.includes("high-context")) {
    raw = raw.filter((tag) => tag !== "high-context");
  }

  const isMultimodal = !!model.vision_model;
  if (isMultimodal && !raw.includes("multi-modal")) {
    raw = ["multi-modal", ...raw];
  }
  if (!isMultimodal && raw.includes("multi-modal")) {
    raw = raw.filter((tag) => tag !== "multi-modal");
  }

  if (isDrawingModel(model) && !raw.includes(DRAWING_MODEL_TAG)) {
    raw = [DRAWING_MODEL_TAG, ...raw];
  }

  return raw;
}
