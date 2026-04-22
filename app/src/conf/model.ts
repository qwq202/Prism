import { Model } from "@/api/types.tsx";

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
