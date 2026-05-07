import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { useTranslation } from "react-i18next";
import { useEffect, useMemo, useReducer, useState } from "react";
import {
  getExternalPlanConfig,
  getPlanConfig,
  type PlanConfig,
  setPlanConfig,
} from "@/admin/api/plan.ts";
import { useEffectAsync } from "@/utils/hook.ts";
import { Switch } from "@/components/ui/switch.tsx";
import {
  Activity,
  ChevronDown,
  ChevronUp,
  Coins,
  GripVertical,
  Plus,
  RotateCw,
  Save,
  Trash2,
} from "lucide-react";
import { getPlanName } from "@/conf/subscription.tsx";
import { Plan, PlanItem } from "@/api/types.tsx";
import Tips from "@/components/Tips.tsx";
import { NumberInput } from "@/components/ui/number-input.tsx";
import { Input } from "@/components/ui/input.tsx";
import { MultiCombobox } from "@/components/ui/multi-combobox.tsx";
import { Button } from "@/components/ui/button.tsx";
import { withNotify } from "@/api/common.ts";
import { dispatchSubscriptionData } from "@/store/globals.ts";
import { useDispatch } from "react-redux";
import { cn } from "@/components/ui/lib/utils.ts";
import { useAllModels } from "@/admin/hook.tsx";
import PopupDialog, { popupTypes } from "@/components/PopupDialog.tsx";
import { PopupAlertDialog } from "@/components/PopupDialogComponent.tsx";
import { getUniqueList } from "@/utils/base.ts";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs.tsx";
import { Label } from "@/components/ui/label.tsx";

const planInitialConfig: PlanConfig = {
  enabled: false,
  plans: [],
};

const defaultPointQuota = 20;
const defaultWeeklyQuota = 200;
const defaultWindowResetInterval = 5 * 60 * 60;

function hasPlanPointPool(plan: Plan): boolean {
  return (plan.quota ?? 0) > 0 || plan.quota === -1;
}

function getFinitePointQuota(
  value?: number,
  fallback = defaultPointQuota,
): number {
  return value && value > 0 ? value : fallback;
}

function normalizeShortWindowQuota(
  value?: number,
  fallback = defaultPointQuota,
): number {
  return value === -1 ? -1 : getFinitePointQuota(value, fallback);
}

function getLegacyQuotaFallback(plan: Plan): number {
  if (plan.items.some((item) => item.value === -1)) return -1;
  const legacyValues = plan.items
    .map((item) => item.value)
    .filter((value) => value > 0);
  if (legacyValues.length === 0) return defaultPointQuota;
  return Math.max(defaultPointQuota, ...legacyValues);
}

function getMigratedWeeklyQuota(plan: Plan, quota: number): number {
  if (quota === -1 || plan.items.some((item) => item.value === -1)) return -1;
  return getFinitePointQuota(
    plan.weekly_quota,
    Math.max(defaultWeeklyQuota, quota * 10),
  );
}

function normalizeWindowQuotaPlan(plan: Plan): Plan {
  const wasWindowQuotaPlan = hasPlanPointPool(plan);
  const quota = normalizeShortWindowQuota(
    plan.quota,
    getLegacyQuotaFallback(plan),
  );
  const resetInterval = defaultWindowResetInterval;
  const weeklyQuota = wasWindowQuotaPlan
    ? plan.weekly_quota ?? 0
    : getMigratedWeeklyQuota(plan, quota);

  if (
    plan.quota === quota &&
    plan.reset_interval === resetInterval &&
    (plan.weekly_quota ?? 0) === weeklyQuota
  )
    return plan;

  return {
    ...plan,
    quota,
    reset_interval: resetInterval,
    weekly_quota: weeklyQuota,
  };
}

function normalizeWindowQuotaConfig(config: PlanConfig): PlanConfig {
  let changed = false;
  const plans = config.plans.map((plan: Plan) => {
    const normalized = normalizeWindowQuotaPlan(plan);
    if (normalized !== plan) changed = true;
    return normalized;
  });
  if (!changed) return config;
  return { ...config, plans };
}

type PlanLevelPayload = { level: number };
type PlanItemPayload = PlanLevelPayload & { index: number };

type PlanConfigAction =
  | { type: "set"; payload: PlanConfig }
  | { type: "set-enabled"; payload: boolean }
  | {
      type: "set-plan-sellable";
      payload: PlanLevelPayload & { sellable: boolean };
    }
  | { type: "set-price"; payload: PlanLevelPayload & { price: number } }
  | { type: "set-plan-quota"; payload: PlanLevelPayload & { quota: number } }
  | {
      type: "set-weekly-quota";
      payload: PlanLevelPayload & { weeklyQuota: number };
    }
  | { type: "set-item-id"; payload: PlanItemPayload & { id: string } }
  | { type: "set-item-name"; payload: PlanItemPayload & { name: string } }
  | { type: "set-item-icon"; payload: PlanItemPayload & { icon: string } }
  | { type: "add-item"; payload: PlanLevelPayload }
  | { type: "set-item-models"; payload: PlanItemPayload & { models: string[] } }
  | { type: "remove-item"; payload: PlanItemPayload }
  | { type: "upward-item"; payload: PlanItemPayload }
  | { type: "downward-item"; payload: PlanItemPayload }
  | {
      type: "set-discount";
      payload: PlanLevelPayload & { month: string; value: number };
    }
  | { type: "remove-discount"; payload: PlanLevelPayload & { month: string } };

function sanitizePlanConfigModels(
  config: PlanConfig,
  availableModels: string[],
): PlanConfig {
  if (availableModels.length === 0) return config;
  const availableSet = new Set(availableModels);
  let changed = false;
  const plans = config.plans.map((plan: Plan) => {
    let planChanged = false;
    const items = plan.items.map((item: PlanItem) => {
      const rawModels = item.models ?? [];
      const filteredModels = getUniqueList(
        rawModels.filter((model) => availableSet.has(model)),
      );
      const sameModels =
        filteredModels.length === rawModels.length &&
        filteredModels.every((model, index) => model === rawModels[index]);
      if (sameModels) return item;
      changed = true;
      planChanged = true;
      return { ...item, models: filteredModels };
    });
    if (!planChanged) return plan;
    return { ...plan, items };
  });
  if (!changed) return config;
  return { ...config, plans };
}

function reducer(state: PlanConfig, action: PlanConfigAction): PlanConfig {
  switch (action.type) {
    case "set":
      return normalizeWindowQuotaConfig(action.payload);
    case "set-enabled":
      return { ...state, enabled: action.payload };
    case "set-plan-sellable":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? { ...plan, sellable: action.payload.sellable }
            : plan,
        ),
      };
    case "set-price":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? { ...plan, price: action.payload.price }
            : plan,
        ),
      };
    case "set-plan-quota":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? {
                ...plan,
                quota: normalizeShortWindowQuota(action.payload.quota),
              }
            : plan,
        ),
      };
    case "set-weekly-quota":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? { ...plan, weekly_quota: action.payload.weeklyQuota }
            : plan,
        ),
      };
    case "set-item-id":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? {
                ...plan,
                items: plan.items.map((item: PlanItem, index: number) =>
                  index === action.payload.index
                    ? { ...item, id: action.payload.id }
                    : item,
                ),
              }
            : plan,
        ),
      };
    case "set-item-name":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? {
                ...plan,
                items: plan.items.map((item: PlanItem, index: number) =>
                  index === action.payload.index
                    ? { ...item, name: action.payload.name }
                    : item,
                ),
              }
            : plan,
        ),
      };
    case "set-item-icon":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? {
                ...plan,
                items: plan.items.map((item: PlanItem, index: number) =>
                  index === action.payload.index
                    ? { ...item, icon: action.payload.icon }
                    : item,
                ),
              }
            : plan,
        ),
      };
    case "add-item":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? {
                ...plan,
                items: [
                  ...plan.items,
                  { id: "", name: "", value: 0, icon: "", models: [] },
                ],
              }
            : plan,
        ),
      };
    case "set-item-models":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? {
                ...plan,
                items: plan.items.map((item: PlanItem, index: number) =>
                  index === action.payload.index
                    ? { ...item, models: action.payload.models }
                    : item,
                ),
              }
            : plan,
        ),
      };
    case "remove-item":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) =>
          plan.level === action.payload.level
            ? {
                ...plan,
                items: plan.items.filter(
                  (_: PlanItem, index: number) =>
                    index !== action.payload.index,
                ),
              }
            : plan,
        ),
      };
    case "upward-item":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) => {
          if (plan.level !== action.payload.level) return plan;
          const items = [...plan.items];
          const index = action.payload.index;
          if (index > 0) {
            const tmp = items[index];
            items[index] = items[index - 1];
            items[index - 1] = tmp;
          }
          return { ...plan, items };
        }),
      };
    case "downward-item":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) => {
          if (plan.level !== action.payload.level) return plan;
          const items = [...plan.items];
          const index = action.payload.index;
          if (index < items.length - 1) {
            const tmp = items[index];
            items[index] = items[index + 1];
            items[index + 1] = tmp;
          }
          return { ...plan, items };
        }),
      };
    case "set-discount":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) => {
          if (plan.level !== action.payload.level) return plan;
          const discounts = {
            ...(plan.discounts || {}),
            [action.payload.month]: action.payload.value,
          };
          return { ...plan, discounts };
        }),
      };
    case "remove-discount":
      return {
        ...state,
        plans: state.plans.map((plan: Plan) => {
          if (plan.level !== action.payload.level || !plan.discounts)
            return plan;
          const discounts = { ...plan.discounts };
          delete discounts[action.payload.month];
          return { ...plan, discounts };
        }),
      };
    default:
      throw new Error();
  }
}

const DISCOUNT_MONTHS = [1, 3, 6, 12, 36];
const DEFAULT_DISCOUNTS: Record<number, number> = { 6: 10, 12: 20, 36: 30 };

// ─── Credits pool settings ────────────────────────────────────────────────────
function CreditsPoolSettings({
  plan,
  formDispatch,
}: {
  plan: Plan;
  formDispatch: React.Dispatch<PlanConfigAction>;
}) {
  const { t } = useTranslation();
  const hasWeeklyPool =
    (plan.weekly_quota ?? 0) > 0 || plan.weekly_quota === -1;

  return (
    <div className="rounded-lg border border-amber-200/60 dark:border-amber-800/40 bg-amber-50/20 dark:bg-amber-950/10 overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-2.5 bg-amber-100/40 dark:bg-amber-900/20 border-b border-amber-200/40 dark:border-amber-800/30">
        <Coins className="h-3.5 w-3.5 text-amber-500" />
        <span className="text-xs font-medium text-amber-700 dark:text-amber-400">
          {t("admin.plan.mode-points")}
        </span>
      </div>

      <div className="p-4 space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <div className="flex items-center justify-between h-5">
              <Label className="text-xs text-muted-foreground flex items-center gap-1">
                {t("admin.plan.plan-quota")}
                <Tips
                  className="inline-block"
                  content={t("admin.plan.plan-quota-tip")}
                />
              </Label>
            </div>
            <NumberInput
              value={plan.quota ?? defaultPointQuota}
              min={-1}
              acceptNegative={true}
              onValueChange={(value) =>
                formDispatch({
                  type: "set-plan-quota",
                  payload: {
                    level: plan.level,
                    quota: normalizeShortWindowQuota(value),
                  },
                })
              }
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.plan.plan-quota-hint")}
            </p>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between h-5">
              <Label className="text-xs text-muted-foreground flex items-center gap-1">
                {t("admin.plan.weekly-quota")}
                <Tips
                  className="inline-block"
                  content={t("admin.plan.weekly-quota-tip")}
                />
              </Label>
              <Switch
                checked={hasWeeklyPool}
                onCheckedChange={(checked) =>
                  formDispatch({
                    type: "set-weekly-quota",
                    payload: {
                      level: plan.level,
                      weeklyQuota: checked
                        ? getFinitePointQuota(
                            plan.weekly_quota,
                            defaultWeeklyQuota,
                          )
                        : 0,
                    },
                  })
                }
              />
            </div>
            {hasWeeklyPool && (
              <>
                <NumberInput
                  value={plan.weekly_quota ?? -1}
                  min={-1}
                  acceptNegative={true}
                  onValueChange={(value) =>
                    formDispatch({
                      type: "set-weekly-quota",
                      payload: { level: plan.level, weeklyQuota: value },
                    })
                  }
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.plan.weekly-quota-hint")}
                </p>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Single plan editor ───────────────────────────────────────────────────────
function PlanEditor({
  plan,
  availableModels,
  formDispatch,
}: {
  plan: Plan;
  availableModels: string[];
  formDispatch: React.Dispatch<PlanConfigAction>;
}) {
  const { t } = useTranslation();
  const colTemplate = "1.5rem 1fr 1fr minmax(0,1.2fr) auto";

  return (
    <div className="space-y-5">
      {/* ── Section 1: Sale Status ── */}
      <div className="flex items-start justify-between gap-4 rounded-lg border bg-muted/20 p-4">
        <div className="space-y-1">
          <Label className="text-sm font-medium">
            {t("admin.plan.sellable")}
          </Label>
          <p className="text-xs text-muted-foreground">
            {t("admin.plan.sellable-tip")}
          </p>
        </div>
        <Switch
          checked={plan.sellable !== false}
          onCheckedChange={(checked) =>
            formDispatch({
              type: "set-plan-sellable",
              payload: { level: plan.level, sellable: checked },
            })
          }
        />
      </div>

      {/* ── Section 2: Pricing & Quotas ── */}
      <div className="space-y-3">
        <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {t("admin.plan.price")}
        </h3>

        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground flex items-center gap-1">
              {t("admin.plan.price")}
              <Tips
                className="inline-block"
                content={t("admin.plan.price-tip")}
              />
            </Label>
            <NumberInput
              value={plan.price}
              onValueChange={(value) =>
                formDispatch({
                  type: "set-price",
                  payload: { level: plan.level, price: value },
                })
              }
            />
          </div>
          <CreditsPoolSettings plan={plan} formDispatch={formDispatch} />
        </div>
      </div>

      {/* ── Section 3: Model Groups ── */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t("admin.plan.items-points-title")}
            </h3>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t("admin.plan.items-points-desc")}
            </p>
          </div>
          <Button
            size="sm"
            variant="outline"
            onClick={() =>
              formDispatch({ type: "add-item", payload: { level: plan.level } })
            }
          >
            <Plus className="h-3.5 w-3.5 mr-1" />
            {t("admin.plan.add-item")}
          </Button>
        </div>

        {plan.items.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 border border-dashed rounded-lg text-muted-foreground text-sm gap-1.5">
            <Plus className="h-6 w-6 opacity-30" />
            <span>{t("admin.plan.no-items-points")}</span>
          </div>
        ) : (
          <div className="rounded-lg border overflow-hidden">
            <div
              className="grid items-center bg-muted/50 px-3 py-2 text-xs font-medium text-muted-foreground border-b select-none"
              style={{ gridTemplateColumns: colTemplate }}
            >
              <span />
              <span className="flex items-center gap-1">
                ID
                <Tips content={t("admin.plan.item-id-placeholder")} />
              </span>
              <span>{t("admin.plan.item-name")}</span>
              <span className="flex items-center gap-1 pl-2">
                {t("admin.plan.item-models")}
                <Tips content={t("admin.plan.item-models-tip")} />
              </span>
              <span className="text-right pr-1">{t("admin.action")}</span>
            </div>

            <div className="divide-y">
              {plan.items.map((item: PlanItem, index: number) => (
                <div
                  key={index}
                  className="group grid items-start gap-2 px-3 py-2 hover:bg-muted/20 transition-colors"
                  style={{ gridTemplateColumns: colTemplate }}
                >
                  <div className="flex items-center justify-center h-9 text-muted-foreground/40">
                    <GripVertical className="h-3.5 w-3.5" />
                  </div>

                  <Input
                    value={item.id}
                    onChange={(e) =>
                      formDispatch({
                        type: "set-item-id",
                        payload: {
                          level: plan.level,
                          id: e.target.value,
                          index,
                        },
                      })
                    }
                    placeholder={t("admin.plan.item-id-placeholder")}
                    className="h-9 text-sm"
                  />

                  <Input
                    value={item.name}
                    onChange={(e) =>
                      formDispatch({
                        type: "set-item-name",
                        payload: {
                          level: plan.level,
                          name: e.target.value,
                          index,
                        },
                      })
                    }
                    placeholder={t("admin.plan.item-name-placeholder")}
                    className="h-9 text-sm"
                  />

                  <MultiCombobox
                    align="start"
                    value={item.models}
                    onChange={(value: string[]) =>
                      formDispatch({
                        type: "set-item-models",
                        payload: { level: plan.level, models: value, index },
                      })
                    }
                    placeholder={t("admin.plan.item-models-placeholder", {
                      length: item.models.length,
                    })}
                    searchPlaceholder={t(
                      "admin.plan.item-models-search-placeholder",
                    )}
                    list={availableModels}
                    className="w-full max-w-full h-9 text-sm"
                  />

                  <div className="flex items-center gap-0.5 pl-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-muted-foreground hover:text-foreground"
                      onClick={() =>
                        formDispatch({
                          type: "upward-item",
                          payload: { level: plan.level, index },
                        })
                      }
                      disabled={index === 0}
                      title={t("upward")}
                    >
                      <ChevronUp className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-muted-foreground hover:text-foreground"
                      onClick={() =>
                        formDispatch({
                          type: "downward-item",
                          payload: { level: plan.level, index },
                        })
                      }
                      disabled={index === plan.items.length - 1}
                      title={t("downward")}
                    >
                      <ChevronDown className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-destructive/70 hover:text-destructive hover:bg-destructive/10"
                      onClick={() =>
                        formDispatch({
                          type: "remove-item",
                          payload: { level: plan.level, index },
                        })
                      }
                      title={t("remove")}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* ── Section 4: Discounts ── */}
      <div className="space-y-3">
        <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
          {t("admin.plan.discounts")}
          <Tips content={t("admin.plan.discounts-tip")} />
        </h3>
        <div className="grid grid-cols-5 gap-2">
          {DISCOUNT_MONTHS.map((month) => {
            const key = month.toString();
            const hasDiscount = plan.discounts?.[key] !== undefined;
            const discountValue = hasDiscount ? plan.discounts![key] : null;
            const pct =
              discountValue !== null
                ? Math.round((1 - discountValue) * 100)
                : 0;

            return (
              <div
                key={month}
                className={cn(
                  "rounded-lg border p-2.5 transition-colors",
                  hasDiscount
                    ? "border-primary/30 bg-primary/5"
                    : "bg-muted/10",
                )}
              >
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-xs font-medium">
                    {t(`sub.time.${month}`)}
                  </span>
                  <Switch
                    checked={hasDiscount}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        const defaultPct = DEFAULT_DISCOUNTS[month] ?? 0;
                        formDispatch({
                          type: "set-discount",
                          payload: {
                            level: plan.level,
                            month: key,
                            value: 1 - defaultPct / 100,
                          },
                        });
                      } else {
                        formDispatch({
                          type: "remove-discount",
                          payload: { level: plan.level, month: key },
                        });
                      }
                    }}
                  />
                </div>
                {hasDiscount && (
                  <div className="space-y-1">
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      <span>{t("admin.plan.discount-off")}</span>
                      <span className="font-semibold text-primary">{pct}%</span>
                    </div>
                    <NumberInput
                      value={pct}
                      min={0}
                      max={90}
                      step={5}
                      onValueChange={(value) =>
                        formDispatch({
                          type: "set-discount",
                          payload: {
                            level: plan.level,
                            month: key,
                            value: 1 - value / 100,
                          },
                        })
                      }
                      className="h-7 text-xs"
                    />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────
function PlanConfig() {
  const { t } = useTranslation();
  const [form, formDispatch] = useReducer(reducer, planInitialConfig);
  const [loading, setLoading] = useState(false);
  const dispatch = useDispatch();

  const { allModels, update } = useAllModels();
  const availableModels = useMemo(() => getUniqueList(allModels), [allModels]);

  const [open, setOpen] = useState(false);
  const [syncOpen, setSyncOpen] = useState(false);
  const [conf, setConf] = useState<PlanConfig | null>(null);

  const confRules = useMemo(
    () => (conf ? conf.plans.flatMap((p: Plan) => p.items) : []),
    [conf],
  );
  const confAllModelPlanCount = useMemo(
    () =>
      conf
        ? conf.plans.filter(
            (plan) => hasPlanPointPool(plan) && plan.items.length === 0,
          ).length
        : 0,
    [conf],
  );
  const confIncluding = useMemo(
    () => getUniqueList(confRules.flatMap((i: PlanItem) => i.models)),
    [confRules],
  );
  const confRuleCount = confRules.length + confAllModelPlanCount;
  const confModelCount =
    confAllModelPlanCount > 0 ? availableModels.length : confIncluding.length;

  const refresh = async (ignoreUpdate?: boolean) => {
    setLoading(true);
    const res = await getPlanConfig();
    if (!ignoreUpdate) await update();
    formDispatch({ type: "set", payload: res });
    setLoading(false);
  };

  const save = async (data?: PlanConfig) => {
    const source = data ?? form;
    const normalized = normalizeWindowQuotaConfig(source);
    const payload = sanitizePlanConfigModels(normalized, availableModels);
    if (payload !== source) formDispatch({ type: "set", payload });
    const res = await setPlanConfig(payload);
    withNotify(t, res, true);
    if (res.status)
      dispatchSubscriptionData(dispatch, payload.enabled ? payload.plans : []);
  };

  useEffectAsync(async () => await refresh(true), []);

  useEffect(() => {
    const normalized = normalizeWindowQuotaConfig(form);
    const sanitized = sanitizePlanConfigModels(normalized, availableModels);
    if (sanitized !== form) formDispatch({ type: "set", payload: sanitized });
  }, [availableModels, form]);

  const activePlans = form.plans.filter((p) => p.level > 0);
  const defaultTab = activePlans[0]?.level.toString() ?? "1";

  return (
    <>
      <PopupDialog
        type={popupTypes.Text}
        title={t("admin.plan.sync")}
        name={t("admin.plan.sync-site")}
        placeholder={t("admin.plan.sync-placeholder")}
        open={open}
        setOpen={setOpen}
        defaultValue={"https://api.chatnio.net"}
        alert={t("admin.format-only")}
        onSubmit={async (endpoint): Promise<boolean> => {
          const conf = normalizeWindowQuotaConfig(
            await getExternalPlanConfig(endpoint),
          );
          setConf(conf);
          setSyncOpen(true);
          return true;
        }}
      />
      <PopupAlertDialog
        title={t("admin.plan.sync")}
        description={t("admin.plan.sync-result", {
          length: confRuleCount,
          models: confModelCount,
        })}
        open={syncOpen}
        setOpen={setSyncOpen}
        destructive={true}
        onSubmit={async () => {
          if (!conf) return false;
          formDispatch({ type: "set", payload: conf });
          await save(conf);
          return true;
        }}
      />

      <div className="space-y-5">
        {/* ── Top toolbar ── */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Switch
              id="plan-enable"
              checked={form.enabled}
              onCheckedChange={(checked) =>
                formDispatch({ type: "set-enabled", payload: checked })
              }
            />
            <Label
              htmlFor="plan-enable"
              className="text-sm font-medium cursor-pointer select-none"
            >
              {t("admin.plan.enable")}
            </Label>
          </div>

          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setOpen(true)}>
              <Activity className="h-3.5 w-3.5 mr-1.5" />
              {t("admin.plan.sync")}
            </Button>

            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => refresh()}
              title={t("admin.plan.sync")}
            >
              <RotateCw
                className={cn("h-3.5 w-3.5", loading && "animate-spin")}
              />
            </Button>

            <Button size="sm" onClick={() => save()} loading={true}>
              <Save className="h-3.5 w-3.5 mr-1.5" />
              {t("save")}
            </Button>
          </div>
        </div>

        {/* ── Plan tabs ── */}
        {form.enabled && activePlans.length > 0 && (
          <Tabs defaultValue={defaultTab}>
            <TabsList>
              {activePlans.map((plan) => (
                <TabsTrigger
                  key={plan.level}
                  value={plan.level.toString()}
                  className="gap-1.5"
                >
                  <span>{t(`sub.${getPlanName(plan.level)}`)}</span>
                  {plan.sellable === false && (
                    <span className="text-[10px] font-normal px-1.5 py-0.5 rounded text-muted-foreground bg-muted">
                      {t("admin.plan.sellable-off")}
                    </span>
                  )}
                  <span className="text-[10px] font-normal px-1.5 py-0.5 rounded text-amber-600 dark:text-amber-400 bg-amber-100 dark:bg-amber-900/30">
                    {t("admin.plan.mode-points-short")}
                  </span>
                </TabsTrigger>
              ))}
            </TabsList>

            {activePlans.map((plan) => (
              <TabsContent
                key={plan.level}
                value={plan.level.toString()}
                className="mt-4"
              >
                <PlanEditor
                  plan={plan}
                  availableModels={availableModels}
                  formDispatch={formDispatch}
                />
              </TabsContent>
            ))}
          </Tabs>
        )}

        {form.enabled && activePlans.length === 0 && (
          <div className="flex items-center justify-center py-12 text-muted-foreground text-sm border border-dashed rounded-lg">
            {t("admin.plan.no-plans")}
          </div>
        )}

        {!form.enabled && (
          <div className="flex items-center justify-center py-12 text-muted-foreground text-sm border border-dashed rounded-lg">
            {t("admin.plan.disabled-hint")}
          </div>
        )}
      </div>
    </>
  );
}

function Subscription() {
  const { t } = useTranslation();
  return (
    <div className="admin-subscription">
      <Card className="admin-card sub-card">
        <CardHeader className="select-none pb-4">
          <CardTitle>{t("admin.subscription")}</CardTitle>
        </CardHeader>
        <CardContent>
          <PlanConfig />
        </CardContent>
      </Card>
    </div>
  );
}

export default Subscription;
