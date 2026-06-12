import { Model } from "@/api/types.tsx";

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

export function normalizeModelDisplayNames<T extends Pick<Model, "id" | "name">>(
  models: T[],
): T[] {
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

  return raw;
}
