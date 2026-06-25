export const tokenBilling = "token-billing";
export const timesBilling = "times-billing";
export const nonBilling = "non-billing";
export const imageBilling = "image-billing";
export const imageBillingModePerImage = "per_image";
export const imageBillingModeMatrix = "matrix";
export const imageBillingModeOfficialUsage = "official_usage";
export const imageMissingPricePolicyDefault = "default";
export const imageMissingPricePolicyReject = "reject";

export const defaultChargeType = tokenBilling;
export const chargeTypes = [
  nonBilling,
  timesBilling,
  tokenBilling,
  imageBilling,
];
export type ChargeType = (typeof chargeTypes)[number];

export type ImageChargeRule = {
  size?: string;
  quality?: string;
  mime_type?: string;
  aspect_ratio?: string;
  quota?: number;
};

export type ImageUsageChargeConfig = {
  input?: number;
  output?: number;
  image?: number;
};

export type ImageChargeConfig = {
  mode?: string;
  missing_price_policy?: string;
  default?: number;
  request?: number;
  reference?: number;
  size?: Record<string, number>;
  quality?: Record<string, number>;
  rules?: ImageChargeRule[];
  usage?: ImageUsageChargeConfig;
  output_count?: number;
  billing_unit?: string;
};

export type ChargeBaseProps = {
  type: string;
  anonymous: boolean;
  input: number;
  output: number;
  cache_hit?: number;
  cache_miss?: number;
  image?: ImageChargeConfig;
};

export type ChargeProps = ChargeBaseProps & {
  id: number;
  models: string[];
};
