import {
  ChargeBaseProps,
  imageBilling,
  imageBillingModeMatrix,
  imageBillingModeOfficialUsage,
  imageBillingModePerImage,
  imageMissingPricePolicyDefault,
  imageMissingPricePolicyReject,
  ImageChargeConfig,
  ImageChargeRule,
  nonBilling,
  timesBilling,
} from "@/admin/charge.ts";

const imageBillingModes = [
  imageBillingModePerImage,
  imageBillingModeMatrix,
  imageBillingModeOfficialUsage,
];
const imageMissingPricePolicies = [
  imageMissingPricePolicyDefault,
  imageMissingPricePolicyReject,
];

export type ImageBillingRequest = {
  size?: string;
  quality?: string;
  mimeType?: string;
  aspectRatio?: string;
};

export type ImageUsageTokens = {
  input?: number;
  output?: number;
  image?: number;
};

function normalizePriceMap(values?: Record<string, number>) {
  const next: Record<string, number> = {};
  Object.entries(values || {}).forEach(([key, value]) => {
    const name = key.trim();
    const price = Number(value);
    if (name && Number.isFinite(price) && price > 0) {
      next[name] = price;
    }
  });
  return next;
}

export function normalizeImageChargeConfig(
  image?: ImageChargeConfig,
  fallback = 0,
): ImageChargeConfig {
  const config = image || {};
  const defaultPrice = Math.max(0, Number(config.default ?? fallback ?? 0));
  const request = Math.max(0, Number(config.request ?? 0));
  const reference = Math.max(0, Number(config.reference ?? 0));
  const outputCount = Math.max(1, Number(config.output_count ?? 1));

  return {
    mode: imageBillingModes.includes(config.mode || "")
      ? config.mode
      : imageBillingModePerImage,
    missing_price_policy: imageMissingPricePolicies.includes(
      config.missing_price_policy || "",
    )
      ? config.missing_price_policy
      : imageMissingPricePolicyDefault,
    default: defaultPrice,
    request,
    reference,
    output_count: outputCount,
    billing_unit: config.billing_unit || "final_image",
    size: normalizePriceMap(config.size),
    quality: normalizePriceMap(config.quality),
    rules: (config.rules || []).map((rule) => ({
      size: rule.size?.trim() || undefined,
      quality: rule.quality?.trim() || undefined,
      mime_type: rule.mime_type?.trim() || undefined,
      aspect_ratio: rule.aspect_ratio?.trim() || undefined,
      quota: Math.max(0, Number(rule.quota || 0)),
    })),
    usage: {
      input: Math.max(0, Number(config.usage?.input ?? 0)),
      output: Math.max(0, Number(config.usage?.output ?? 0)),
      image: Math.max(0, Number(config.usage?.image ?? 0)),
    },
  };
}

function normalizeImageBillingKey(value: string): string {
  return value.trim().toLowerCase();
}

export function getImagePriceForKey(
  prices: Record<string, number> | undefined,
  key: string,
): number {
  if (!prices) return 0;
  const normalized = normalizeImageBillingKey(key);
  const hit = Object.entries(prices).find(
    ([name]) => normalizeImageBillingKey(name) === normalized,
  );
  return hit ? Number(hit[1]) || 0 : 0;
}

function imageRuleFieldMatches(expected: string | undefined, actual: string) {
  const normalizedExpected = normalizeImageBillingKey(expected || "");
  if (!normalizedExpected) return true;
  return normalizedExpected === normalizeImageBillingKey(actual);
}

function imageRuleMatches(
  rule: ImageChargeRule,
  request: Required<ImageBillingRequest>,
) {
  return (
    imageRuleFieldMatches(rule.size, request.size) &&
    imageRuleFieldMatches(rule.quality, request.quality) &&
    imageRuleFieldMatches(rule.mime_type, request.mimeType) &&
    imageRuleFieldMatches(rule.aspect_ratio, request.aspectRatio)
  );
}

function findImageChargeRule(
  rules: ImageChargeRule[] | undefined,
  request: Required<ImageBillingRequest>,
) {
  return (rules || []).find(
    (rule) => Number(rule.quota) > 0 && imageRuleMatches(rule, request),
  );
}

function getLegacyImageUnitPrice(
  config: ImageChargeConfig,
  fallbackOutput: number,
  request: Required<ImageBillingRequest>,
) {
  let price = Math.max(0, Number(config.default ?? fallbackOutput) || 0);
  const sizePrice = getImagePriceForKey(config.size, request.size);
  if (sizePrice > 0) {
    price = sizePrice;
  }
  const qualityPrice = getImagePriceForKey(config.quality, request.quality);
  if (qualityPrice > 0) {
    price += qualityPrice;
  }
  return Math.max(0, price);
}

function getOfficialUsageQuota(
  config: ImageChargeConfig,
  usageTokens?: ImageUsageTokens,
) {
  const usage = config.usage;
  if (!usage) return 0;

  const inputRate = Math.max(0, Number(usage.input || 0));
  const outputRate = Math.max(0, Number(usage.output || 0));
  const imageRate = Math.max(0, Number(usage.image || 0));
  if (inputRate <= 0 && outputRate <= 0 && imageRate <= 0) {
    return 0;
  }

  const inputTokens = Math.max(0, Number(usageTokens?.input || 0));
  const outputTokens = Math.max(0, Number(usageTokens?.output || 0));
  const imageTokens = Math.max(0, Number(usageTokens?.image || 0));
  if (inputTokens <= 0 && outputTokens <= 0 && imageTokens <= 0) {
    return 0;
  }

  return (
    (inputTokens / 1000) * inputRate +
    (outputTokens / 1000) * outputRate +
    (imageTokens / 1000) * imageRate
  );
}

export function estimateImageQuota(
  charge: ChargeBaseProps | undefined,
  request: ImageBillingRequest = {},
  referenceImages = 0,
  usageTokens?: ImageUsageTokens,
): number | null {
  if (!charge) return null;

  if (charge.type === nonBilling) {
    return 0;
  }

  if (charge.type === timesBilling) {
    return Math.max(0, Number(charge.output) || 0);
  }

  if (charge.type !== imageBilling) {
    return null;
  }

  const config = normalizeImageChargeConfig(charge.image, charge.output);
  const outputCount = Math.max(1, Number(config.output_count) || 1);
  const refs = Math.max(0, referenceImages);
  const requestQuota = Math.max(0, Number(config.request) || 0);
  const referenceQuota =
    Math.max(0, Number(config.reference) || 0) * refs;
  const normalizedRequest: Required<ImageBillingRequest> = {
    size: request.size?.trim() || "",
    quality: request.quality?.trim() || "",
    mimeType: request.mimeType?.trim() || "",
    aspectRatio: request.aspectRatio?.trim() || "",
  };

  if (config.mode === imageBillingModeOfficialUsage) {
    const usageQuota = getOfficialUsageQuota(config, usageTokens);
    if (usageQuota > 0) {
      return requestQuota + referenceQuota + usageQuota;
    }

    const fallbackUnit = getLegacyImageUnitPrice(
      config,
      charge.output,
      normalizedRequest,
    );
    if (fallbackUnit > 0) {
      return requestQuota + referenceQuota + fallbackUnit * outputCount;
    }

    if (config.missing_price_policy === imageMissingPricePolicyReject) {
      return null;
    }
    return requestQuota + referenceQuota;
  }

  let unitQuota = 0;
  if (config.mode === imageBillingModeMatrix) {
    const matchedRule = findImageChargeRule(config.rules, normalizedRequest);
    if (matchedRule?.quota) {
      unitQuota = Math.max(0, Number(matchedRule.quota) || 0);
    } else if (
      config.missing_price_policy === imageMissingPricePolicyReject
    ) {
      return null;
    }
  }

  if (unitQuota <= 0) {
    unitQuota = getLegacyImageUnitPrice(
      config,
      charge.output,
      normalizedRequest,
    );
  }

  return requestQuota + referenceQuota + unitQuota * outputCount;
}

export function countImagePreviewQuota(
  image: ImageChargeConfig | undefined,
  size: string,
  quality: string,
  referenceImages: number,
  usageTokens?: ImageUsageTokens,
) {
  return (
    estimateImageQuota(
      {
        type: imageBilling,
        anonymous: false,
        input: 0,
        output: 0,
        image,
      },
      { size, quality },
      referenceImages,
      usageTokens,
    ) ?? 0
  );
}
