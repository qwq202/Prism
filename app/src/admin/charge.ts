export const tokenBilling = "token-billing";
export const timesBilling = "times-billing";
export const nonBilling = "non-billing";
export const imageBilling = "image-billing";

export const defaultChargeType = tokenBilling;
export const chargeTypes = [
  nonBilling,
  timesBilling,
  tokenBilling,
  imageBilling,
];
export type ChargeType = (typeof chargeTypes)[number];

export type ImageChargeConfig = {
  default?: number;
  request?: number;
  reference?: number;
  size?: Record<string, number>;
  quality?: Record<string, number>;
  output_count?: number;
  billing_unit?: string;
};

export type ChargeBaseProps = {
  type: string;
  anonymous: boolean;
  input: number;
  output: number;
  image?: ImageChargeConfig;
};

export type ChargeProps = ChargeBaseProps & {
  id: number;
  models: string[];
};
