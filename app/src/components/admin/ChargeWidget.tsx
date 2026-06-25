import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group.tsx";
import { Label } from "@/components/ui/label.tsx";
import {
  ChargeProps,
  chargeTypes,
  defaultChargeType,
  imageBilling,
  imageBillingModeMatrix,
  imageBillingModeOfficialUsage,
  imageBillingModePerImage,
  ImageChargeConfig,
  ImageChargeRule,
  imageMissingPricePolicyDefault,
  imageMissingPricePolicyReject,
  nonBilling,
  timesBilling,
  tokenBilling,
} from "@/admin/charge.ts";
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input.tsx";
import { useMemo, useReducer, useState } from "react";
import { Button } from "@/components/ui/button.tsx";
import {
  Activity,
  AlertCircle,
  BoxIcon,
  Check,
  Cloud,
  Copy,
  DownloadCloud,
  Eraser,
  EyeOff,
  FileImage,
  Image as ImageIcon,
  KanbanSquareDashed,
  Layers3,
  Minus,
  PencilLine,
  Plus,
  RotateCw,
  Ruler,
  Search,
  Settings2,
  SlidersHorizontal,
  Trash,
  UploadCloud,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu.tsx";
import {
  Command,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command.tsx";
import { withNotify } from "@/api/common.ts";
import { Switch } from "@/components/ui/switch.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { NumberInput } from "@/components/ui/number-input.tsx";
import {
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import OperationAction from "@/components/OperationAction.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import {
  deleteCharge,
  listCharge,
  setCharge,
  syncCharge,
  fetchUpstreamCharge,
} from "@/admin/api/charge.ts";
import { useEffectAsync } from "@/utils/hook.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert.tsx";
import Tips from "@/components/Tips.tsx";
import { getQuerySelector, scrollUp, useClipboard } from "@/utils/dom.ts";
import PopupDialog, { popupTypes } from "@/components/PopupDialog.tsx";
import { getV1Path } from "@/api/v1.ts";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import { getUniqueList, isEnter, parseNumber } from "@/utils/base.ts";
import { defaultChannelModels } from "@/admin/channel.ts";
import { getPricing } from "@/admin/datasets/charge.ts";
import { useAllModels } from "@/admin/hook.tsx";
import { toast } from "sonner";
import { formatDecimal } from "@/utils/base.ts";
import { isDrawingModel } from "@/conf/model.ts";

const initialState: ChargeProps = {
  id: -1,
  type: defaultChargeType,
  models: [],
  anonymous: false,
  input: 0,
  output: 0,
};

const imageSizePriceKeys = [
  "512px",
  "1K",
  "2K",
  "4K",
  "256x256",
  "512x512",
  "1024x1024",
  "1024x1536",
  "1536x1024",
  "1024x1792",
  "1792x1024",
];
const imageQualityPriceKeys = ["low", "medium", "high"];
const imageBillingModes = [
  imageBillingModePerImage,
  imageBillingModeMatrix,
  imageBillingModeOfficialUsage,
];
const imageMissingPricePolicies = [
  imageMissingPricePolicyDefault,
  imageMissingPricePolicyReject,
];

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

function normalizeImageChargeConfig(
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
    rules: (config.rules || [])
      .map((rule) => ({
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

function getImagePriceForKey(
  prices: Record<string, number> | undefined,
  key: string,
): number {
  if (!prices) return 0;
  const normalized = key.trim().toLowerCase();
  const hit = Object.entries(prices).find(
    ([name]) => name.trim().toLowerCase() === normalized,
  );
  return hit ? Number(hit[1]) || 0 : 0;
}

function countImagePreviewQuota(
  image: ImageChargeConfig | undefined,
  size: string,
  quality: string,
  referenceImages: number,
  usageTokens?: { input: number; output: number; image: number },
) {
  const config = normalizeImageChargeConfig(image);
  if (config.mode === imageBillingModeOfficialUsage) {
    return (
      (config.request || 0) +
      (config.reference || 0) * Math.max(0, referenceImages) +
      Math.max(0, usageTokens?.input || 0) / 1000 *
        Math.max(0, config.usage?.input || 0) +
      Math.max(0, usageTokens?.output || 0) / 1000 *
        Math.max(0, config.usage?.output || 0) +
      Math.max(0, usageTokens?.image || 0) / 1000 *
        Math.max(0, config.usage?.image || 0)
    );
  }
  const matchedRule = (config.rules || []).find((rule) => {
    const ruleSize = (rule.size || "").trim().toLowerCase();
    const ruleQuality = (rule.quality || "").trim().toLowerCase();
    return (
      (!ruleSize || ruleSize === size.trim().toLowerCase()) &&
      (!ruleQuality || ruleQuality === quality.trim().toLowerCase())
    );
  });
  if (config.mode === imageBillingModeMatrix && matchedRule?.quota) {
    return (
      (config.request || 0) +
      matchedRule.quota * Math.max(1, config.output_count || 1) +
      (config.reference || 0) * Math.max(0, referenceImages)
    );
  }
  const sizePrice = getImagePriceForKey(config.size, size);
  const qualityPrice = getImagePriceForKey(config.quality, quality);
  const unitPrice = (sizePrice > 0 ? sizePrice : config.default || 0) +
    qualityPrice;

  return (
    (config.request || 0) +
    unitPrice * Math.max(1, config.output_count || 1) +
    (config.reference || 0) * Math.max(0, referenceImages)
  );
}

function formatImageChargeSummary(charge: ChargeProps) {
  const image = normalizeImageChargeConfig(charge.image, charge.output);
  if (image.mode === imageBillingModeOfficialUsage) {
    return "Usage";
  }
  if (image.mode === imageBillingModeMatrix && (image.rules || []).length > 0) {
    return `${image.rules?.length || 0} rules`;
  }
  const sizes = Object.entries(image.size || {}).filter(([, value]) => value > 0);
  if (sizes.length > 0) {
    return sizes
      .slice(0, 3)
      .map(([size, value]) => `${size}: ${formatDecimal(value)}`)
      .join(" · ");
  }
  return `${formatDecimal(image.default || charge.output || 0)} / image`;
}

function hasTokenCachePrices(charge: ChargeProps) {
  return (charge.cache_hit ?? 0) > 0 || (charge.cache_miss ?? 0) > 0;
}

type ChargeAction =
  | { type: "set"; payload: ChargeProps }
  | { type: "set-models"; payload: string[] }
  | { type: "add-model"; payload: string }
  | { type: "toggle-model"; payload: string }
  | { type: "remove-model"; payload: string }
  | { type: "set-type"; payload: string }
  | { type: "set-anonymous"; payload: boolean }
  | { type: "set-input"; payload: number }
  | { type: "set-output"; payload: number }
  | { type: "set-cache-hit"; payload: number }
  | { type: "set-cache-miss"; payload: number }
  | { type: "set-image"; payload: ImageChargeConfig }
  | { type: "set-image-mode"; payload: string }
  | { type: "set-image-missing-policy"; payload: string }
  | { type: "set-image-number"; key: keyof ImageChargeConfig; payload: number }
  | { type: "set-image-size-price"; key: string; payload: number }
  | { type: "set-image-all-size-prices"; payload: number }
  | { type: "set-image-quality-price"; key: string; payload: number }
  | { type: "set-image-usage-price"; key: "input" | "output" | "image"; payload: number }
  | { type: "add-image-rule" }
  | { type: "remove-image-rule"; index: number }
  | {
      type: "set-image-rule";
      index: number;
      key: keyof ImageChargeRule;
      payload: string | number;
    }
  | { type: "clear" }
  | { type: "clear-param" };

type ChargeDispatch = (action: ChargeAction) => void;

function reducer(state: ChargeProps, action: ChargeAction): ChargeProps {
  switch (action.type) {
    case "set":
      return { ...action.payload };
    case "set-models":
      return { ...state, models: action.payload };
    case "add-model": {
      const model = action.payload.trim();
      if (model.length === 0 || state.models.includes(model)) return state;
      return { ...state, models: [...state.models, model] };
    }
    case "toggle-model":
      if (action.payload.trim().length === 0) return state;
      return state.models.includes(action.payload)
        ? {
            ...state,
            models: state.models.filter((model) => model !== action.payload),
          }
        : { ...state, models: [...state.models, action.payload] };
    case "remove-model":
      return {
        ...state,
        models: state.models.filter((model) => model !== action.payload),
      };
    case "set-type":
      return action.payload === imageBilling
        ? {
            ...state,
            type: action.payload,
            input: 0,
            image: normalizeImageChargeConfig(state.image, state.output),
          }
        : { ...state, type: action.payload };
    case "set-anonymous":
      return { ...state, anonymous: action.payload };
    case "set-input":
      return { ...state, input: action.payload };
    case "set-output":
      return { ...state, output: action.payload };
    case "set-cache-hit":
      return {
        ...state,
        cache_hit: action.payload > 0 ? action.payload : undefined,
      };
    case "set-cache-miss":
      return {
        ...state,
        cache_miss: action.payload > 0 ? action.payload : undefined,
      };
    case "set-image": {
      const image = normalizeImageChargeConfig(action.payload, state.output);
      return { ...state, image, output: image.default || state.output };
    }
    case "set-image-mode": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      return { ...state, image: { ...image, mode: action.payload } };
    }
    case "set-image-missing-policy": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      return {
        ...state,
        image: { ...image, missing_price_policy: action.payload },
      };
    }
    case "set-image-number": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      const value = Math.max(
        action.key === "output_count" ? 1 : 0,
        action.payload,
      );
      const next = { ...image, [action.key]: value };
      return {
        ...state,
        image: next,
        output: action.key === "default" ? value : state.output,
      };
    }
    case "set-image-size-price": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      const nextSize = { ...(image.size || {}) };
      if (action.payload > 0) {
        nextSize[action.key] = action.payload;
      } else {
        delete nextSize[action.key];
      }
      return { ...state, image: { ...image, size: nextSize } };
    }
    case "set-image-all-size-prices": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      const nextSize: Record<string, number> = {};
      if (action.payload > 0) {
        imageSizePriceKeys.forEach((key) => {
          nextSize[key] = action.payload;
        });
      }
      return { ...state, image: { ...image, size: nextSize } };
    }
    case "set-image-quality-price": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      const nextQuality = { ...(image.quality || {}) };
      if (action.payload > 0) {
        nextQuality[action.key] = action.payload;
      } else {
        delete nextQuality[action.key];
      }
      return { ...state, image: { ...image, quality: nextQuality } };
    }
    case "set-image-usage-price": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      return {
        ...state,
        image: {
          ...image,
          usage: {
            ...(image.usage || {}),
            [action.key]: Math.max(0, action.payload),
          },
        },
      };
    }
    case "add-image-rule": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      return {
        ...state,
        image: {
          ...image,
          rules: [
            ...(image.rules || []),
            {
              size: "1K",
              quality: "",
              mime_type: "",
              aspect_ratio: "",
              quota: image.default || state.output || 0,
            },
          ],
        },
      };
    }
    case "remove-image-rule": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      return {
        ...state,
        image: {
          ...image,
          rules: (image.rules || []).filter((_, index) => index !== action.index),
        },
      };
    }
    case "set-image-rule": {
      const image = normalizeImageChargeConfig(state.image, state.output);
      const rules = [...(image.rules || [])];
      const current = rules[action.index] || {};
      rules[action.index] = {
        ...current,
        [action.key]:
          action.key === "quota"
            ? Math.max(0, Number(action.payload || 0))
            : String(action.payload || ""),
      };
      return { ...state, image: { ...image, rules } };
    }
    case "clear":
      return initialState;
    case "clear-param":
      return { ...initialState, id: state.id };
    default:
      return state;
  }
}

function preflight(state: ChargeProps): ChargeProps {
  state.models = state.models
    .map((model) => model.trim())
    .filter((model) => model.length > 0);
  switch (state.type) {
    case nonBilling:
      state.input = 0;
      state.output = 0;
      state.cache_hit = undefined;
      state.cache_miss = undefined;
      break;
    case timesBilling:
      state.input = 0;
      state.anonymous = false;
      state.cache_hit = undefined;
      state.cache_miss = undefined;
      break;
    case tokenBilling:
      state.anonymous = false;
      if ((state.cache_hit ?? 0) <= 0) state.cache_hit = undefined;
      if ((state.cache_miss ?? 0) <= 0) state.cache_miss = undefined;
      break;
    case imageBilling:
      state.input = 0;
      state.anonymous = false;
      state.cache_hit = undefined;
      state.cache_miss = undefined;
      state.image = normalizeImageChargeConfig(state.image, state.output);
      state.image.rules = (state.image.rules || []).filter(
        (rule) =>
          (rule.size || rule.quality || rule.mime_type || rule.aspect_ratio) &&
          (rule.quota || 0) > 0,
      );
      state.output = state.image.default || state.output || 0;
      break;
  }

  if (state.input < 0) state.input = 0;
  if (state.output < 0) state.output = 0;

  return state;
}

type SyncDialogProps = {
  current: string[];
  builtin: boolean;
  open: boolean;
  setOpen: (open: boolean) => void;
  onRefresh: () => void;
  system: string;
};

function SyncDialog({
  builtin,
  current,
  open,
  setOpen,
  onRefresh,
  system,
}: SyncDialogProps) {
  const { t } = useTranslation();

  const [siteCharge, setSiteCharge] = useState<ChargeProps[]>([]);
  const [siteOpen, setSiteOpen] = useState(false);

  const [overwrite, setOverwrite] = useState(false);
  const siteModels = useMemo(
    () => siteCharge.flatMap((charge) => charge.models),
    [siteCharge],
  );
  const influencedModels = useMemo(
    () =>
      overwrite
        ? siteModels
        : siteModels.filter((model) => !current.includes(model)),
    [overwrite, siteModels, current],
  );

  return (
    <>
      <PopupDialog
        type={popupTypes.Number}
        title={t("admin.charge.sync-builtin")}
        name={t("admin.charge.usd-currency")}
        open={open && builtin}
        setOpen={setOpen}
        defaultValue={"7.1"}
        onSubmit={async (_currency: string): Promise<boolean> => {
          const currency = parseNumber(_currency);
          const pricing = getPricing(currency);

          setSiteCharge(pricing);
          setSiteOpen(true);

          return true;
        }}
      />
      <PopupDialog
        type={popupTypes.Text}
        title={t("admin.charge.sync")}
        name={t("admin.charge.sync-site")}
        placeholder={t("admin.charge.sync-placeholder")}
        open={open && !builtin}
        setOpen={setOpen}
        defaultValue={"https://api.chatnio.net"}
        alert={system === "" ? t("admin.format-only") : undefined}
        onSubmit={async (endpoint): Promise<boolean> => {
          const path = system === "newapi"
            ? `${endpoint.replace(/\/$/, "")}/api/ratio_config`
            : getV1Path("/v1/charge", { endpoint });
          const resp = await fetchUpstreamCharge({ endpoint, system });

          if (!resp.status || resp.data.length === 0) {
            toast.error(t("admin.charge.sync-failed"), {
              description: t("admin.charge.sync-failed-prompt", {
                endpoint: path,
              }),
            });
            return false;
          }

          setSiteCharge(resp.data);
          setSiteOpen(true);
          return true;
        }}
      />
      <Dialog open={siteOpen} onOpenChange={setSiteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admin.charge.sync-option")}</DialogTitle>
            <DialogDescription className={`pt-1.5`}>
              {t("admin.charge.sync-prompt", {
                length: siteModels.length,
                influence: influencedModels.length,
              })}
            </DialogDescription>
          </DialogHeader>
          <div className={`pt-1 flex flex-row items-center justify-center`}>
            <span className={`mr-4 whitespace-nowrap`}>
              {t("admin.charge.sync-overwrite")}
            </span>
            <Switch checked={overwrite} onCheckedChange={setOverwrite} />
          </div>
          <DialogFooter>
            <Button
              unClickable
              variant={`outline`}
              onClick={() => {
                setSiteOpen(false);
                setSiteCharge([]);
              }}
            >
              {t("cancel")}
            </Button>
            <Button
              unClickable
              loading={true}
              variant={overwrite ? `destructive` : `default`}
              onClick={async () => {
                const resp = await syncCharge({
                  data: siteCharge,
                  overwrite,
                });
                withNotify(t, resp, true);

                if (resp.status) {
                  setOpen(false);
                  setSiteOpen(false);
                  setSiteCharge([]);

                  onRefresh();
                }
              }}
            >
              {t("admin.charge.sync-confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

type ChargeActionProps = {
  loading: boolean;
  onRefresh: () => void;
  currentModels: string[];
};

function ChargeAction({
  loading,
  onRefresh,
  currentModels,
}: ChargeActionProps) {
  const { t } = useTranslation();
  const [popup, setPopup] = useState(false);
  const [builtin, setBuiltin] = useState(false);
  const [system, setSystem] = useState("");

  const open = (builtin: boolean) => {
    setBuiltin(builtin);
    setPopup(true);
  };

  return (
    <div className={`flex flex-row w-full h-max`}>
      <SyncDialog
        builtin={builtin}
        onRefresh={onRefresh}
        current={currentModels}
        open={popup}
        setOpen={setPopup}
        system={system}
      />
      <Button variant={`default`} className={`mr-2`} onClick={() => open(true)}>
        <KanbanSquareDashed className={`w-4 h-4 mr-2`} />
        {t("admin.charge.sync-builtin")}
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant={`outline`}>
            <Activity className={`w-4 h-4 mr-2`} />
            {t("admin.charge.sync")}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align={`start`}>
          <DropdownMenuItem
            onSelect={() => {
              setSystem("");
              open(false);
            }}
          >
            Prism
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => {
              setSystem("newapi");
              open(false);
            }}
          >
            NewAPI
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <div className={`grow`} />
      <Button variant={`outline`} size={`icon`} onClick={onRefresh}>
        <RotateCw className={cn("w-4 h-4", loading && "animate-spin")} />
      </Button>
    </div>
  );
}

type ChargeAlertProps = {
  models: string[];
  onClick: (model: string) => void;
};

function ChargeAlert({ models, onClick }: ChargeAlertProps) {
  const { t } = useTranslation();

  return (
    models.length > 0 && (
      <Alert className={`charge-alert`}>
        <AlertTitle className={`flex flex-row items-center select-none`}>
          <AlertCircle className="h-4 w-4 mr-2" />
          <p>{t("admin.charge.unused-model")}</p>
          <Tips content={t("admin.charge.unused-model-tip")} />
        </AlertTitle>
        <AlertDescription className={`model-list`}>
          {models.slice(0, 15).map((model, index) => (
            <Button
              key={index}
              variant={`outline`}
              className={`cursor-pointer h-8 select-none flex flex-row items-center`}
              onClick={() => onClick(model)}
            >
              <BoxIcon className={`w-3.5 h-3.5 mr-1`} />
              {model}
            </Button>
          ))}
        </AlertDescription>
      </Alert>
    )
  );
}

type ImageBillingEditorProps = {
  form: ChargeProps;
  dispatch: ChargeDispatch;
};

function ImageBillingEditor({ form, dispatch }: ImageBillingEditorProps) {
  const { t } = useTranslation();
  const image = normalizeImageChargeConfig(form.image, form.output);
  const [previewSize, setPreviewSize] = useState("1K");
  const [previewQuality, setPreviewQuality] = useState("");
  const [previewReferences, setPreviewReferences] = useState(0);
  const [previewInputTokens, setPreviewInputTokens] = useState(0);
  const [previewOutputTokens, setPreviewOutputTokens] = useState(0);
  const [previewImageTokens, setPreviewImageTokens] = useState(0);
  const [bulkSizePrice, setBulkSizePrice] = useState(0);
  const previewQuota = countImagePreviewQuota(
    image,
    previewSize,
    previewQuality,
    previewReferences,
    {
      input: previewInputTokens,
      output: previewOutputTokens,
      image: previewImageTokens,
    },
  );
  const isMatrixBilling = image.mode === imageBillingModeMatrix;
  const isOfficialUsageBilling = image.mode === imageBillingModeOfficialUsage;

  return (
    <div className="mt-5 rounded-lg border border-border/70 bg-muted/20 p-4">
      <div className="mb-4 flex flex-row items-start gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-background border border-border">
          <ImageIcon className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <p className="text-sm font-semibold">
            {t("admin.charge.image-billing-title")}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("admin.charge.image-billing-desc")}
          </p>
        </div>
      </div>

      <div className="mb-4 grid gap-3 md:grid-cols-2">
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">
            {t("admin.charge.image-billing-mode")}
          </Label>
          <Select
            value={image.mode || imageBillingModePerImage}
            onValueChange={(value) =>
              dispatch({ type: "set-image-mode", payload: value })
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {imageBillingModes.map((mode) => (
                <SelectItem key={mode} value={mode}>
                  {t(`admin.charge.image-billing-mode-${mode}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">
            {t("admin.charge.image-missing-policy")}
          </Label>
          <Select
            value={image.missing_price_policy || imageMissingPricePolicyDefault}
            onValueChange={(value) =>
              dispatch({ type: "set-image-missing-policy", payload: value })
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {imageMissingPricePolicies.map((policy) => (
                <SelectItem key={policy} value={policy}>
                  {t(`admin.charge.image-missing-policy-${policy}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div className="flex flex-row items-center gap-2">
          <FileImage className="h-4 w-4 shrink-0 text-muted-foreground" />
          <Label className="grow">{t("admin.charge.image-default")}</Label>
          <NumberInput
            value={image.default || 0}
            onValueChange={(value) =>
              dispatch({
                type: "set-image-number",
                key: "default",
                payload: value,
              })
            }
            acceptNegative={false}
            className="w-24"
            min={0}
            max={99999}
          />
        </div>
        <div className="flex flex-row items-center gap-2">
          <Layers3 className="h-4 w-4 shrink-0 text-muted-foreground" />
          <Label className="grow">{t("admin.charge.image-output-count")}</Label>
          <NumberInput
            value={image.output_count || 1}
            onValueChange={(value) =>
              dispatch({
                type: "set-image-number",
                key: "output_count",
                payload: value,
              })
            }
            acceptNegative={false}
            className="w-24"
            min={1}
            max={99}
          />
        </div>
        <div className="flex flex-row items-center gap-2">
          <SlidersHorizontal className="h-4 w-4 shrink-0 text-muted-foreground" />
          <Label className="grow">{t("admin.charge.image-request")}</Label>
          <NumberInput
            value={image.request || 0}
            onValueChange={(value) =>
              dispatch({
                type: "set-image-number",
                key: "request",
                payload: value,
              })
            }
            acceptNegative={false}
            className="w-24"
            min={0}
            max={99999}
          />
        </div>
        <div className="flex flex-row items-center gap-2">
          <UploadCloud className="h-4 w-4 shrink-0 text-muted-foreground" />
          <Label className="grow">{t("admin.charge.image-reference")}</Label>
          <NumberInput
            value={image.reference || 0}
            onValueChange={(value) =>
              dispatch({
                type: "set-image-number",
                key: "reference",
                payload: value,
              })
            }
            acceptNegative={false}
            className="w-24"
            min={0}
            max={99999}
          />
        </div>
      </div>

      <div className="mt-5 grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <div className="space-y-4">
          <div>
            <div className="mb-2 flex flex-wrap items-center gap-2 text-sm font-medium">
              <Ruler className="h-4 w-4 text-muted-foreground" />
              <span className="grow">{t("admin.charge.image-size-prices")}</span>
              <NumberInput
                value={bulkSizePrice}
                onValueChange={setBulkSizePrice}
                acceptNegative={false}
                className="h-8 w-20"
                min={0}
                max={99999}
              />
              <Button
                type="button"
                variant="secondary"
                size="sm"
                className="h-8 gap-1 px-2"
                onClick={() =>
                  dispatch({
                    type: "set-image-all-size-prices",
                    payload: bulkSizePrice,
                  })
                }
              >
                <Check className="h-3.5 w-3.5" />
                {t("admin.charge.image-apply-all")}
              </Button>
            </div>
            <div className="grid gap-x-3 gap-y-2 rounded-md border border-border/50 bg-muted/30 p-3 sm:grid-cols-2 lg:grid-cols-3">
              {imageSizePriceKeys.map((size) => (
                <div key={size} className="flex items-center gap-2">
                  <span className="w-20 shrink-0 whitespace-nowrap text-xs font-medium tabular-nums text-muted-foreground">
                    {size}
                  </span>
                  <NumberInput
                    value={getImagePriceForKey(image.size, size)}
                    onValueChange={(value) =>
                      dispatch({
                        type: "set-image-size-price",
                        key: size,
                        payload: value,
                      })
                    }
                    acceptNegative={false}
                    className="h-8 min-w-0 flex-1"
                    min={0}
                    max={99999}
                  />
                </div>
              ))}
            </div>
          </div>

          {isMatrixBilling && (
            <div>
              <div className="mb-2 flex items-center gap-2 text-sm font-medium">
                <KanbanSquareDashed className="h-4 w-4 text-muted-foreground" />
                <span className="grow">
                  {t("admin.charge.image-matrix-rules")}
                </span>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  className="h-8 gap-1 px-2"
                  onClick={() => dispatch({ type: "add-image-rule" })}
                >
                  <Plus className="h-3.5 w-3.5" />
                  {t("admin.charge.image-add-matrix-rule")}
                </Button>
              </div>
              <div className="space-y-2 rounded-md border border-border/50 bg-muted/30 p-3">
                {(image.rules || []).length === 0 && (
                  <p className="py-2 text-center text-xs text-muted-foreground">
                    {t("admin.charge.image-empty-matrix-rules")}
                  </p>
                )}
                {(image.rules || []).map((rule, index) => (
                  <div
                    key={index}
                    className="grid gap-2 rounded-md border border-border/70 bg-background p-2 sm:grid-cols-[repeat(5,minmax(0,1fr))_auto]"
                  >
                    <Input
                      value={rule.size || ""}
                      onChange={(event) =>
                        dispatch({
                          type: "set-image-rule",
                          index,
                          key: "size",
                          payload: event.target.value,
                        })
                      }
                      placeholder={t("admin.charge.image-preview-size")}
                      className="h-8"
                    />
                    <Input
                      value={rule.quality || ""}
                      onChange={(event) =>
                        dispatch({
                          type: "set-image-rule",
                          index,
                          key: "quality",
                          payload: event.target.value,
                        })
                      }
                      placeholder={t("admin.charge.image-preview-quality")}
                      className="h-8"
                    />
                    <Input
                      value={rule.mime_type || ""}
                      onChange={(event) =>
                        dispatch({
                          type: "set-image-rule",
                          index,
                          key: "mime_type",
                          payload: event.target.value,
                        })
                      }
                      placeholder={t("admin.charge.image-rule-format")}
                      className="h-8"
                    />
                    <Input
                      value={rule.aspect_ratio || ""}
                      onChange={(event) =>
                        dispatch({
                          type: "set-image-rule",
                          index,
                          key: "aspect_ratio",
                          payload: event.target.value,
                        })
                      }
                      placeholder={t("admin.charge.image-rule-ratio")}
                      className="h-8"
                    />
                    <NumberInput
                      value={rule.quota || 0}
                      onValueChange={(value) =>
                        dispatch({
                          type: "set-image-rule",
                          index,
                          key: "quota",
                          payload: value,
                        })
                      }
                      acceptNegative={false}
                      className="h-8"
                      min={0}
                      max={99999}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() =>
                        dispatch({ type: "remove-image-rule", index })
                      }
                    >
                      <Trash className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {isOfficialUsageBilling && (
            <div>
              <div className="mb-2 flex items-center gap-2 text-sm font-medium">
                <Activity className="h-4 w-4 text-muted-foreground" />
                {t("admin.charge.image-usage-prices")}
              </div>
              <div className="grid gap-2 rounded-md border border-border/50 bg-muted/30 p-3 sm:grid-cols-3">
                {(["input", "output", "image"] as const).map((key) => (
                  <div
                    key={key}
                    className="flex items-center gap-2 rounded-md border border-border/70 bg-background px-3 py-2"
                  >
                    <span className="w-16 text-sm font-medium">
                      {t(`admin.charge.image-usage-${key}`)}
                    </span>
                    <NumberInput
                      value={image.usage?.[key] || 0}
                      onValueChange={(value) =>
                        dispatch({
                          type: "set-image-usage-price",
                          key,
                          payload: value,
                        })
                      }
                      acceptNegative={false}
                      className="h-9 flex-1"
                      min={0}
                      max={99999}
                    />
                  </div>
                ))}
              </div>
            </div>
          )}

          <div>
            <div className="mb-2 flex items-center gap-2 text-sm font-medium">
              <Settings2 className="h-4 w-4 text-muted-foreground" />
              {t("admin.charge.image-quality-prices")}
            </div>
            <div className="grid gap-2 sm:grid-cols-3">
              {imageQualityPriceKeys.map((quality) => (
                <div
                  key={quality}
                  className="flex items-center gap-2 rounded-md border border-border/70 bg-background px-3 py-2"
                >
                  <span className="w-16 text-sm font-medium">
                    {t(`admin.charge.image-quality-${quality}`)}
                  </span>
                  <NumberInput
                    value={getImagePriceForKey(image.quality, quality)}
                    onValueChange={(value) =>
                      dispatch({
                        type: "set-image-quality-price",
                        key: quality,
                        payload: value,
                      })
                    }
                    acceptNegative={false}
                    className="h-9 flex-1"
                    min={0}
                    max={99999}
                  />
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border/70 bg-background p-4">
          <div className="mb-3 flex items-center gap-2 text-sm font-semibold">
            <Activity className="h-4 w-4 text-muted-foreground" />
            {t("admin.charge.image-preview")}
          </div>
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <div>
                <Label className="mb-1 block text-xs text-muted-foreground">
                  {t("admin.charge.image-preview-size")}
                </Label>
                <Input
                  value={previewSize}
                  onChange={(e) => setPreviewSize(e.target.value)}
                  className="h-9"
                />
              </div>
              <div>
                <Label className="mb-1 block text-xs text-muted-foreground">
                  {t("admin.charge.image-preview-quality")}
                </Label>
                <Input
                  value={previewQuality}
                  onChange={(e) => setPreviewQuality(e.target.value)}
                  placeholder={t("admin.charge.image-preview-quality-empty")}
                  className="h-9"
                />
              </div>
            </div>
            <div>
              <Label className="mb-1 block text-xs text-muted-foreground">
                {t("admin.charge.image-preview-reference")}
              </Label>
              <NumberInput
                value={previewReferences}
                onValueChange={setPreviewReferences}
                acceptNegative={false}
                className="h-9"
                min={0}
                max={99}
              />
            </div>
            {isOfficialUsageBilling && (
              <div className="grid grid-cols-3 gap-2">
                <div>
                  <Label className="mb-1 block text-xs text-muted-foreground">
                    {t("admin.charge.image-usage-input")}
                  </Label>
                  <NumberInput
                    value={previewInputTokens}
                    onValueChange={setPreviewInputTokens}
                    acceptNegative={false}
                    className="h-9"
                    min={0}
                    max={99999999}
                  />
                </div>
                <div>
                  <Label className="mb-1 block text-xs text-muted-foreground">
                    {t("admin.charge.image-usage-output")}
                  </Label>
                  <NumberInput
                    value={previewOutputTokens}
                    onValueChange={setPreviewOutputTokens}
                    acceptNegative={false}
                    className="h-9"
                    min={0}
                    max={99999999}
                  />
                </div>
                <div>
                  <Label className="mb-1 block text-xs text-muted-foreground">
                    {t("admin.charge.image-usage-image")}
                  </Label>
                  <NumberInput
                    value={previewImageTokens}
                    onValueChange={setPreviewImageTokens}
                    acceptNegative={false}
                    className="h-9"
                    min={0}
                    max={99999999}
                  />
                </div>
              </div>
            )}
            <div className="rounded-md border border-border bg-muted/30 p-3">
              <p className="text-xs text-muted-foreground">
                {t("admin.charge.image-preview-total")}
              </p>
              <p className="mt-1 text-2xl font-semibold tabular-nums">
                {formatDecimal(previewQuota)}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

type ChargeEditorProps = {
  form: ChargeProps;
  dispatch: ChargeDispatch;
  onRefresh: () => void;
  usedModels: string[];
  allModels: string[];
  unitM: boolean;
  setUnitM: (v: boolean | ((prev: boolean) => boolean)) => void;
};

function ChargeEditor({
  form,
  dispatch,
  onRefresh,
  usedModels,
  allModels,
  unitM,
  setUnitM,
}: ChargeEditorProps) {
  const { t } = useTranslation();

  const [model, setModel] = useState("");
  const multiplier = unitM ? 1000 : 1;

  const channelModels = useMemo(
    () => getUniqueList([...allModels, ...defaultChannelModels]),
    [allModels],
  );

  const unusedModels = useMemo(() => {
    const candidates =
      form.type === imageBilling
        ? channelModels.filter((model) => isDrawingModel(model))
        : channelModels;
    return candidates.filter(
      (model) =>
        !form.models.includes(model) &&
        !usedModels.includes(model) &&
        model.trim() !== "",
    );
  }, [channelModels, form.models, form.type, usedModels]);

  const disabled = useMemo(() => {
    if (model.trim() !== "") return false;
    return form.models.length === 0;
  }, [model, form.models]);

  const [loading, setLoading] = useState(false);

  async function post() {
    const raw = model.trim();
    const data = preflight({ ...form });
    if (raw !== "" && !data.models.includes(raw)) {
      data.models = [raw, ...data.models];
      setModel("");
    }

    const resp = await setCharge(data);
    withNotify(t, resp, true);

    if (resp.status) clear();
    onRefresh();
  }

  function clear() {
    dispatch({ type: "clear" });
    setModel("");
  }

  return (
    <div className={`charge-editor`}>
      <div className={`w-full h-max mb-5`}>
        <RadioGroup
          value={form.type}
          onValueChange={(value) =>
            dispatch({ type: "set-type", payload: value })
          }
          className={`flex flex-row gap-5 whitespace-nowrap flex-wrap`}
        >
          {chargeTypes.map((chargeType, index) => (
            <div
              className="flex items-center space-x-2 cursor-pointer"
              key={index}
            >
              <RadioGroupItem
                className={`transition-all duration-200`}
                value={chargeType}
                id={chargeType}
              />
              <Label htmlFor={chargeType} className={`cursor-pointer`}>
                {t(`admin.charge.${chargeType}`)}
              </Label>
            </div>
          ))}
        </RadioGroup>
      </div>
      <div className={`flex flex-row w-full h-max mb-4`}>
        <Button
          onClick={() => {
            dispatch({ type: "add-model", payload: model });
            setModel("");
          }}
          size={`icon`}
          className={`mr-2 shrink-0`}
        >
          <Plus className={`w-4 h-4`} />
        </Button>
        <Input
          value={model}
          onChange={(e) => setModel(e.target.value)}
          placeholder={t("admin.channels.model")}
          onKeyDown={(e) => {
            if (isEnter(e)) {
              dispatch({ type: "add-model", payload: model });
              setModel("");
            }
          }}
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size={`icon`} className={`ml-2 shrink-0`}>
              <Search className={`w-4 h-4`} />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align={`end`} asChild>
            <Command>
              <CommandInput placeholder={t("admin.channels.search-model")} />
              <CommandList className={`thin-scrollbar`}>
                {unusedModels.map((model, idx) => (
                  <CommandItem
                    key={idx}
                    value={model}
                    onSelect={(value) =>
                      dispatch({ type: "add-model", payload: value })
                    }
                    className={`px-2`}
                  >
                    {model}
                  </CommandItem>
                ))}
              </CommandList>
            </Command>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className={`flex flex-col w-full h-max mb-2`}>
        {form.models.map((model, index) => (
          <div
            className={`flex flex-row w-full h-max shrink-0 mb-2 select-none`}
            key={index}
          >
            <Input value={model} readOnly />
            <Button
              onClick={() => dispatch({ type: "remove-model", payload: model })}
              size={`icon`}
              variant={`outline`}
              className={`ml-2 shrink-0`}
            >
              <Minus className={`w-4 h-4`} />
            </Button>
          </div>
        ))}
      </div>

      {form.type === nonBilling && (
        <div className={`flex flex-row w-full h-max items-center mt-4 mb-6`}>
          <EyeOff className={`w-4 h-4 mr-2`} />
          <Label className={`grow`}>{t("admin.charge.anonymous")}</Label>
          <Switch
            checked={form.anonymous}
            onCheckedChange={(checked) =>
              dispatch({ type: "set-anonymous", payload: checked })
            }
          />
        </div>
      )}

      {form.type === timesBilling && (
        <div className={`flex flex-row w-full h-max items-center`}>
          <Cloud className={`w-4 h-4 mr-2`} />
          <Label className={`grow`}>{t("admin.charge.time-count")}</Label>
          <NumberInput
            value={form.output}
            onValueChange={(value) =>
              dispatch({ type: "set-output", payload: value })
            }
            acceptNegative={false}
            className={`w-20`}
            min={0}
            max={99999}
          />
        </div>
      )}

      {form.type === tokenBilling && (
        <div className={`flex flex-col w-full h-max gap-2`}>
          <div className={`flex flex-row w-full h-max items-center`}>
            <UploadCloud className={`w-4 h-4 mr-2`} />
            <Label className={`grow`}>
              {t("admin.charge.input-count")}
              <span
                className={`token cursor-pointer select-none hover:text-foreground transition-colors ml-0.5`}
                onClick={() => setUnitM((v) => !v)}
                title={unitM ? "切换到 1k tokens" : "切换到 1M tokens"}
              >
                {" / "}{unitM ? "1M" : "1k"}{" tokens ↕"}
              </span>
            </Label>
            <NumberInput
              value={parseFloat((form.input * multiplier).toPrecision(10))}
              onValueChange={(value) =>
                dispatch({ type: "set-input", payload: value / multiplier })
              }
              acceptNegative={false}
              className={`w-20`}
              min={0}
              max={99999999}
            />
          </div>
          <div className={`flex flex-row w-full h-max items-center`}>
            <DownloadCloud className={`w-4 h-4 mr-2`} />
            <Label className={`grow`}>
              {t("admin.charge.output-count")}
              <span
                className={`token cursor-pointer select-none hover:text-foreground transition-colors ml-0.5`}
                onClick={() => setUnitM((v) => !v)}
                title={unitM ? "切换到 1k tokens" : "切换到 1M tokens"}
              >
                {" / "}{unitM ? "1M" : "1k"}{" tokens ↕"}
              </span>
            </Label>
            <NumberInput
              value={parseFloat((form.output * multiplier).toPrecision(10))}
              onValueChange={(value) =>
                dispatch({ type: "set-output", payload: value / multiplier })
              }
              acceptNegative={false}
              className={`w-20`}
              min={0}
              max={99999999}
            />
          </div>
          <div className={`flex flex-row w-full h-max items-center`}>
            <Cloud className={`w-4 h-4 mr-2`} />
            <Label className={`grow`}>
              {t("admin.charge.cache-hit-count")}
              <span
                className={`token cursor-pointer select-none hover:text-foreground transition-colors ml-0.5`}
                onClick={() => setUnitM((v) => !v)}
                title={unitM ? "切换到 1k tokens" : "切换到 1M tokens"}
              >
                {" / "}
                {unitM ? "1M" : "1k"}
                {" tokens ↕"}
              </span>
            </Label>
            <NumberInput
              value={parseFloat(
                (((form.cache_hit ?? 0) * multiplier) || 0).toPrecision(10),
              )}
              onValueChange={(value) =>
                dispatch({ type: "set-cache-hit", payload: value / multiplier })
              }
              acceptNegative={false}
              className={`w-20`}
              min={0}
              max={99999999}
            />
          </div>
          <div className={`flex flex-row w-full h-max items-center`}>
            <UploadCloud className={`w-4 h-4 mr-2`} />
            <Label className={`grow`}>
              {t("admin.charge.cache-miss-count")}
              <span
                className={`token cursor-pointer select-none hover:text-foreground transition-colors ml-0.5`}
                onClick={() => setUnitM((v) => !v)}
                title={unitM ? "切换到 1k tokens" : "切换到 1M tokens"}
              >
                {" / "}
                {unitM ? "1M" : "1k"}
                {" tokens ↕"}
              </span>
            </Label>
            <NumberInput
              value={parseFloat(
                (((form.cache_miss ?? 0) * multiplier) || 0).toPrecision(10),
              )}
              onValueChange={(value) =>
                dispatch({
                  type: "set-cache-miss",
                  payload: value / multiplier,
                })
              }
              acceptNegative={false}
              className={`w-20`}
              min={0}
              max={99999999}
            />
          </div>
        </div>
      )}

      {form.type === imageBilling && (
        <ImageBillingEditor form={form} dispatch={dispatch} />
      )}

      <div
        className={`flex flex-row w-full h-max mt-5 gap-2 items-center flex-wrap`}
      >
        <div className={`object-id`}>
          <span className={`mr-2`}>ID</span>
          {form.id === -1 ? (
            <Plus className={`w-3 h-3`} />
          ) : (
            <span className={`id`}>{form.id}</span>
          )}
        </div>
        <div className={`grow`} />
        <Button
          variant={`outline`}
          size={`icon`}
          className={`shrink-0`}
          onClick={clear}
        >
          <Eraser className={`w-4 h-4`} />
        </Button>
        <Button
          disabled={disabled}
          onClick={post}
          loading={true}
          onLoadingChange={setLoading}
          className={`whitespace-nowrap shrink-0`}
        >
          {form.id === -1 ? (
            <>
              {!loading && <Plus className={`w-4 h-4 mr-2`} />}
              {t("admin.charge.add-rule")}
            </>
          ) : (
            <>
              {!loading && <PencilLine className={`w-4 h-4 mr-2`} />}
              {t("admin.charge.update-rule")}
            </>
          )}
        </Button>
      </div>
    </div>
  );
}

type ChargeTableProps = {
  data: ChargeProps[];
  dispatch: ChargeDispatch;
  onRefresh: () => void;
  unitM: boolean;
  setUnitM: (v: boolean | ((prev: boolean) => boolean)) => void;
  loading: boolean;
};

function ChargeTableSkeleton() {
  return (
    <>
      {Array.from({ length: 6 }).map((_, index) => (
        <TableRow
          key={index}
          className="pointer-events-none hover:bg-transparent"
        >
          <TableCell>
            <Skeleton className="h-5 w-10" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-7 w-32 rounded-full" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-40" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-8" />
          </TableCell>
          <TableCell>
            <div className="inline-flex flex-row flex-wrap gap-2">
              <Skeleton className="h-9 w-9" />
              <Skeleton className="h-9 w-9" />
            </div>
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

function ChargeTable({
  data,
  dispatch,
  onRefresh,
  unitM,
  setUnitM,
  loading,
}: ChargeTableProps) {
  const { t } = useTranslation();
  const copy = useClipboard();
  const multiplier = unitM ? 1000 : 1;
  const initialLoading = loading && data.length === 0;

  return (
    <div className={`charge-table`}>
      <Table classNameWrapper={`table`}>
        <TableHeader>
          <TableRow className={`select-none whitespace-nowrap`}>
            <TableCell>{t("admin.charge.id")}</TableCell>
            <TableCell>{t("admin.charge.type")}</TableCell>
            <TableCell>{t("admin.charge.model")}</TableCell>
            <TableCell>
              {t("admin.charge.input")}
              <span
                className={`ml-1 text-xs text-muted-foreground cursor-pointer hover:text-foreground transition-colors`}
                onClick={() => setUnitM((v) => !v)}
                title={unitM ? "切换到 1k tokens" : "切换到 1M tokens"}
              >
                /{unitM ? "M" : "k"}↕
              </span>
            </TableCell>
            <TableCell>
              {t("admin.charge.output")}
              <span
                className={`ml-1 text-xs text-muted-foreground cursor-pointer hover:text-foreground transition-colors`}
                onClick={() => setUnitM((v) => !v)}
                title={unitM ? "切换到 1k tokens" : "切换到 1M tokens"}
              >
                /{unitM ? "M" : "k"}↕
              </span>
            </TableCell>
            <TableCell>{t("admin.charge.support-anonymous")}</TableCell>
            <TableCell>{t("admin.charge.action")}</TableCell>
          </TableRow>
        </TableHeader>
        <TableBody>
          {initialLoading ? (
            <ChargeTableSkeleton />
          ) : (
            data.map((charge, idx) => (
              <TableRow key={idx}>
                <TableCell className={`charge-id`}>{charge.id}</TableCell>
                <TableCell>
                  <Badge className={`whitespace-nowrap`}>
                    {t(`admin.charge.${charge.type}`)}
                  </Badge>
                </TableCell>
                <TableCell>
                  {charge.models.map((model, index) => (
                    <p
                      key={index}
                      className={`whitespace-nowrap cursor-pointer`}
                      onClick={() => copy(model)}
                    >
                      {model}
                      <Copy className={`inline w-3 h-3 ml-1`} />
                    </p>
                  ))}
                </TableCell>
                <TableCell>
                  {charge.type === imageBilling ? (
                    t("admin.charge.image-reference-short", {
                      quota: formatDecimal(
                        normalizeImageChargeConfig(
                          charge.image,
                          charge.output,
                        ).reference || 0,
                      ),
                    })
                  ) : (
                    <>
                      {formatDecimal(
                        parseFloat((charge.input * multiplier).toPrecision(10)),
                      )}
                      {charge.type === tokenBilling &&
                        hasTokenCachePrices(charge) && (
                          <p className="mt-1 text-xs text-muted-foreground">
                            {t("admin.charge.cache-short", {
                              hit: formatDecimal(
                                parseFloat(
                                  (
                                    ((charge.cache_hit ?? 0) * multiplier) ||
                                    0
                                  ).toPrecision(10),
                                ),
                              ),
                              miss: formatDecimal(
                                parseFloat(
                                  (
                                    ((charge.cache_miss ?? 0) * multiplier) ||
                                    0
                                  ).toPrecision(10),
                                ),
                              ),
                            })}
                          </p>
                        )}
                    </>
                  )}
                </TableCell>
                <TableCell>
                  {charge.type === imageBilling
                    ? formatImageChargeSummary(charge)
                    : formatDecimal(
                        parseFloat(
                          (charge.output * multiplier).toPrecision(10),
                        ),
                      )}
                </TableCell>
                <TableCell>{t(String(charge.anonymous))}</TableCell>
                <TableCell>
                  <div className={`inline-flex flex-row flex-wrap gap-2`}>
                    <OperationAction
                      tooltip={t("admin.channels.edit")}
                      onClick={async () => {
                        const props: ChargeProps = { ...charge };
                        dispatch({ type: "set", payload: props });

                        // scroll to top
                        scrollUp(
                          getQuerySelector(
                            ".admin-content > .scrollarea-viewport",
                          )!,
                        );
                      }}
                    >
                      <Settings2 className={`h-4 w-4`} />
                    </OperationAction>
                    <OperationAction
                      tooltip={t("admin.channels.delete")}
                      variant={`destructive`}
                      onClick={async () => {
                        const resp = await deleteCharge(charge.id);
                        withNotify(t, resp, true);
                        onRefresh();
                      }}
                    >
                      <Trash className={`h-4 w-4`} />
                    </OperationAction>
                  </div>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function ChargeWidget() {
  const { t } = useTranslation();
  const [data, setData] = useState<ChargeProps[]>([]);
  const [form, dispatch] = useReducer(reducer, initialState);
  const [loading, setLoading] = useState(false);
  const [unitM, setUnitM] = useState(false);

  const { allModels, update } = useAllModels();

  const currentModels = useMemo(() => {
    return data.flatMap((charge) => charge.models);
  }, [data]);

  const usedModels = useMemo((): string[] => {
    return data.flatMap((charge) => charge.models);
  }, [data]);

  const unusedModels = useMemo(() => {
    if (loading) return [];
    return allModels.filter(
      (model) => !usedModels.includes(model) && model.trim() !== "",
    );
  }, [loading, allModels, usedModels]);

  async function refresh(ignoreUpdate?: boolean) {
    setLoading(true);
    try {
      const resp = await listCharge();
      if (!ignoreUpdate) await update();

      if (resp.status) {
        setData(resp.data);
      } else {
        withNotify(t, resp);
      }
    } finally {
      setLoading(false);
    }
  }

  useEffectAsync(async () => await refresh(true), []);

  return (
    <div className={`charge-widget`}>
      <ChargeAction
        loading={loading}
        onRefresh={refresh}
        currentModels={currentModels}
      />
      <ChargeAlert
        models={unusedModels}
        onClick={(model) => dispatch({ type: "toggle-model", payload: model })}
      />
      <ChargeEditor
        onRefresh={refresh}
        form={form}
        dispatch={dispatch}
        allModels={allModels}
        usedModels={usedModels}
        unitM={unitM}
        setUnitM={setUnitM}
      />
      <ChargeTable
        data={data}
        dispatch={dispatch}
        onRefresh={refresh}
        unitM={unitM}
        setUnitM={setUnitM}
        loading={loading}
      />
    </div>
  );
}

export default ChargeWidget;
