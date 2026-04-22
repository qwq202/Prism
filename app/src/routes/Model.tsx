import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input.tsx";
import {
  ArrowDownToDot,
  ArrowRightLeft,
  ArrowUpFromDot,
  Award,
  Bolt,
  Cloud,
  Cpu,
  DollarSign,
  EyeIcon,
  Gem,
  Github,
  Globe,
  Image,
  Link,
  Search,
  Snail,
  Sparkles,
  Star,
  Tag,
  X,
  Zap,
} from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import { splitList } from "@/utils/base.ts";
import type { Model } from "@/api/types.tsx";
import { useDispatch, useSelector } from "react-redux";
import {
  addModelList,
  removeModelList,
  selectModel,
  selectModelList,
  selectSupportModels,
  setModel,
} from "@/store/chat.ts";
import { levelSelector } from "@/store/subscription.ts";
import { teenagerSelector } from "@/store/package.ts";
import { selectAuthenticated } from "@/store/auth.ts";
import { docsEndpoint } from "@/conf/env.ts";
import { goAuth } from "@/utils/app.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { includingModelFromPlan } from "@/conf/subscription.tsx";
import { subscriptionDataSelector } from "@/store/globals.ts";
import { getResolvedModelTags } from "@/conf/model.ts";
import {
  ChargeBaseProps,
  nonBilling,
  timesBilling,
  tokenBilling,
} from "@/admin/charge.ts";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import router from "@/router.tsx";
import ModelAvatar from "@/components/ModelAvatar.tsx";
import { ToggleGroup } from "@radix-ui/react-toggle-group";
import { marketTags } from "@/admin/market.ts";
import { ToggleGroupItem } from "@/components/ui/toggle-group.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import { Label } from "@/components/ui/label.tsx";
import { toast } from "sonner";
import { motion, AnimatePresence } from "framer-motion";
import Tips from "@/components/Tips";
import Icon from "@/components/utils/Icon";

const tagIcons: { [key: string]: React.ReactNode } = {
  official: <Award />,
  "multi-modal": <EyeIcon />,
  web: <Globe />,
  "high-quality": <Sparkles />,
  "high-price": <DollarSign />,
  "open-source": <Github />,
  "image-generation": <Image />,
  fast: <Bolt />,
  unstable: <Snail />,
  "high-context": <Cpu />,
  free: <Zap />,
};

const notDisplayTags = ["official", "fast", "unstable", "free"];

type SearchBarProps = {
  text: string;
  onTextChange: (value: string) => void;
  tags: string[];
  onTagsChange: (value: string[]) => void;
  displayPricing: boolean;
  onDisplayPricingChange: (value: boolean) => void;
  show1mPricing: boolean;
  onShow1mPricingChange: (value: boolean) => void;
};

function getTags(model: Model): string[] {
  return getResolvedModelTags(model);
}

function SearchBar({
  text,
  onTextChange,
  tags,
  onTagsChange,
  displayPricing,
  onDisplayPricingChange,
  show1mPricing,
  onShow1mPricingChange,
}: SearchBarProps) {
  const { t } = useTranslation();

  const supportModels = useSelector(selectSupportModels);
  const availableTags = useMemo(
    () =>
      marketTags.filter((tag) =>
        supportModels.some((model) => getTags(model).includes(tag)),
      ),
    [supportModels],
  );

  return (
    <motion.div
      className={`flex flex-col search-bar-wrapper`}
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      <div className={`option-bar flex flex-row mb-2 items-center`}>
        <div className={`grow`} />
        <Label>{t("market.show-pricing")}</Label>
        <Switch
          checked={displayPricing}
          onCheckedChange={onDisplayPricingChange}
          className={`ml-1.5 scale-90`}
        />

        {displayPricing && (
          <>
            <Label className={`ml-2`}>K/M</Label>
            <Switch
              checked={show1mPricing}
              onCheckedChange={onShow1mPricingChange}
              className={`ml-1.5 scale-90`}
            />
          </>
        )}
      </div>
      <div className={`search-bar`}>
        <Search size={16} className={`search-icon`} />
        <Input
          placeholder={t("market.search")}
          className={`input-box`}
          value={text}
          onChange={(e) => onTextChange(e.target.value)}
        />
        <X
          size={16}
          className={cn("clear-icon", text.length > 0 && "active")}
          onClick={() => onTextChange("")}
        />
      </div>
      <motion.div
        className={`tags-search-area`}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.2, duration: 0.3 }}
      >
        <ToggleGroup
          type={`multiple`}
          value={tags}
          onValueChange={onTagsChange}
          className={`flex flex-row flex-wrap justify-center`}
        >
          {availableTags.map((tag, index) => (
            <motion.div
              key={index}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <ToggleGroupItem value={tag} variant={`outline`} size={`col`}>
                {tagIcons[tag] && (
                  <Icon icon={tagIcons[tag]} className={`w-3.5 h-3.5 mr-1`} />
                )}
                {t(`tag.${tag}`)}
              </ToggleGroupItem>
            </motion.div>
          ))}
        </ToggleGroup>
      </motion.div>
    </motion.div>
  );
}

type ModelProps = React.DetailedHTMLProps<
  React.HTMLAttributes<HTMLDivElement>,
  HTMLDivElement
> & {
  model: Model;
  className?: string;
  style?: React.CSSProperties;
  forwardRef?: React.Ref<HTMLDivElement>;
  showPricing?: boolean;
  show1mPricing?: boolean;
  index: number;
  onModelSelect: (model: Model) => void;
};

type PriceColumnProps = ChargeBaseProps & {
  pro: boolean;
  anonymous?: boolean;
  show1mPricing?: boolean;
};

function PriceColumn({
  type,
  input,
  output,
  pro,
  show1mPricing,
}: PriceColumnProps) {
  const { t } = useTranslation();

  const unitName = !show1mPricing ? "1K TOKENS" : "1M TOKENS";
  const unitValue = !show1mPricing ? 1 : 1000;

  const className = cn(
    "flex flex-row text-sm items-center px-2 pr-1 py-1 w-full rounded-md border transition-all",
    pro && "pro",
  );

  const iconClassName =
    "h-4 w-4 scale-110 mr-2 p-0.5 rounded-full bg-primary/5";

  switch (type) {
    case nonBilling:
      return (
        <motion.div
          className={cn(className, "bg-secondary/5 hover:bg-secondary/10")}
          whileHover={{ scale: 1.02 }}
          transition={{ type: "spring", stiffness: 300 }}
        >
          <Cloud className={iconClassName} />
          <span className="flex-grow">{t("tag.badges.non-billing")}</span>
          <span className="text-2xs ml-1 px-1.5 bg-input/40 select-none rounded-sm">
            FREE
          </span>
        </motion.div>
      );
    case timesBilling:
      return (
        <motion.div
          className={cn(className, "bg-secondary/5 hover:bg-secondary/10")}
          whileHover={{ scale: 1.02 }}
          transition={{ type: "spring", stiffness: 300 }}
        >
          <Cloud className={iconClassName} />
          <span className="flex-grow">
            {t("tag.badges.times-billing", { price: output })}
          </span>
          <span className="text-2xs ml-1 px-1.5 bg-input/40 select-none rounded-sm">
            TIME
          </span>
        </motion.div>
      );
    case tokenBilling:
      const inputValue = input * unitValue;
      const outputValue = output * unitValue;

      return (
        <div className="grid grid-cols-2 gap-1">
          <motion.div
            className={cn(className, "bg-secondary/5 hover:bg-secondary/10")}
            whileHover={{ scale: 1.02 }}
            transition={{ type: "spring", stiffness: 300 }}
          >
            <ArrowUpFromDot className={iconClassName} />
            <span className="flex-grow">{inputValue}</span>
            <span className="text-2xs ml-1 px-1.5 bg-input/40 select-none rounded-sm">
              {unitName}
            </span>
          </motion.div>
          <motion.div
            className={cn(className, "bg-secondary/5 hover:bg-secondary/10")}
            whileHover={{ scale: 1.02 }}
            transition={{ type: "spring", stiffness: 300 }}
          >
            <ArrowDownToDot className={iconClassName} />
            <span className="flex-grow">{outputValue}</span>
            <span className="text-2xs ml-1 px-1.5 bg-input/40 select-none rounded-sm">
              {unitName}
            </span>
          </motion.div>
        </div>
      );
  }
}

function ModelItem({
  model,
  className,
  style,
  forwardRef,
  showPricing,
  show1mPricing,
  index,
  onModelSelect,
  ...props
}: ModelProps) {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const list = useSelector(selectModelList);

  const level = useSelector(levelSelector);
  const student = useSelector(teenagerSelector);

  const subscriptionData = useSelector(subscriptionDataSelector);

  const state = useMemo(() => list.includes(model.id), [model, list]);

  const pro = useMemo(() => {
    return includingModelFromPlan(subscriptionData, level, model.id);
  }, [subscriptionData, model, level, student]);

  const tags = useMemo(
    (): string[] => getTags(model).filter((tag) => tag !== "free"),
    [model],
  );

  return (
    <motion.div
      className={cn("model-item rounded-md", className)}
      style={style} //@ts-ignore
      ref={forwardRef}
      {...props}
      onClick={() => onModelSelect(model)}
      initial={{ opacity: 0, x: -50 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.5, delay: index * 0.1 }}
      whileHover={{ scale: 1.05 }}
    >
      <motion.div
        className={`model-info-wrapper w-full h-max flex flex-row`}
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: index * 0.1 + 0.2 }}
      >
        <div
          className={`model-info flex flex-row items-center flex-wrap w-full mt-1 ml-1`}
        >
          <motion.div
            className={`model-avatar-wrapper mr-1.5 -translate-x-2 -translate-y-2 flex w-max h-max border rounded-full`}
            whileHover={{ scale: 1.1, rotate: 360 }}
            whileTap={{ scale: 0.9 }}
            initial={{ opacity: 0, rotate: -180 }}
            animate={{ opacity: 1, rotate: 0 }}
            transition={{ duration: 0.5, delay: index * 0.1 + 0.4 }}
          >
            <ModelAvatar className={`model-avatar`} model={model} size={24} />
          </motion.div>
          <div className={"flex flex-row items-center model-name mr-2"}>
            {model.name}
          </div>
          {/* <Tips
            content={model.id}
            trigger={<Tag className={`w-5 h-5 p-1 bg-primary/5 rounded-sm`} />}
          /> */}
          {pro && (
            <Tips
              content={t("tag.badges.plan-included-tip")}
              trigger={
                <Gem
                  className={`w-5 h-5 p-1 rounded-sm ml-1 text-amber-600 bg-amber-500/20`}
                />
              }
            />
          )}
          {tags
            .filter((tag) => !notDisplayTags.includes(tag))
            .map((tag, index) => (
              <Tips
                key={index}
                content={t(`tag.${tag}`)}
                trigger={
                  tagIcons[tag] ? (
                    <Icon
                      icon={tagIcons[tag]}
                      className={cn(
                        `w-5 h-5 p-1 rounded-sm ml-1 bg-primary/5`,
                        {
                          "text-amber-600 bg-amber-500/20": tag === "official",
                          "text-blue-600 bg-blue-500/20": tag === "multi-modal",
                          "text-green-600 bg-green-500/20": tag === "web",
                          "text-purple-600 bg-purple-500/20":
                            tag === "high-quality",
                          "text-red-600 bg-red-500/20": tag === "high-price",
                          "text-gray-600 bg-gray-500/20": tag === "open-source",
                          "text-indigo-600 bg-indigo-500/20":
                            tag === "image-generation",
                          "text-yellow-600 bg-yellow-500/20": tag === "fast",
                          "text-orange-600 bg-orange-500/20":
                            tag === "unstable",
                          "text-teal-600 bg-teal-500/20":
                            tag === "high-context",
                          "text-emerald-600 bg-emerald-500/20": tag === "free",
                        },
                      )}
                    />
                  ) : undefined
                }
              />
            ))}
        </div>
      </motion.div>
      <motion.p
        className={`model-description text-sm my-1.5 ml-1`}
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: index * 0.1 + 0.5 }}
      >
        <div className="px-1.5 py-0.5 bg-primary/5 border rounded-md inline-block mr-1 text-xs text-muted-foreground">
          <Tag className={`w-3 h-3 scale-90 mr-1 inline`} />
          {model.id}
        </div>
        {model.description}
      </motion.p>

      <div className={`flex-grow`} />
      {showPricing && model.price && (
        <motion.div
          className={`mt-2.5`}
          initial={{ opacity: 0, y: 0 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: index * 0.1 + 0.6 }}
        >
          <PriceColumn
            type={model.price.type}
            input={model.price.input}
            output={model.price.output}
            pro={pro}
            show1mPricing={show1mPricing}
            anonymous={true}
          />
        </motion.div>
      )}

      <div className="flex flex-row mt-1.5">
        <div className="flex-grow" />
        <motion.span
          className={`clickable w-fit h-fit p-1 border hover:border-hover transition-all rounded-md`}
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();

            dispatch(
              state ? removeModelList(model.id) : addModelList(model.id),
            );

            toast.info(t("market.switch-bookmark"), {
              description: (
                <div
                  className={`inline-flex flex-row items-center flex-wrap space-x-1 space-y-1`}
                >
                  <p className={`translate-y-[1px]`}>
                    {state
                      ? t("market.remove-bookmark")
                      : t("market.add-bookmark")}
                  </p>
                  <ModelAvatar size={20} model={model} />
                  <p>{model.name}</p>
                </div>
              ),
            });
          }}
        >
          {state ? (
            <Star className={`w-4 h-4 shrink-0 fill-current text-amber-500`} />
          ) : (
            <Star className={`w-4 h-4 shrink-0 text-muted-foreground`} />
          )}
        </motion.span>
      </div>
    </motion.div>
  );
}

type MarketPlaceProps = {
  search: string;
  showPricing: boolean;
  show1mPricing: boolean;
  onSelect: (model: Model) => void;
};

function MarketPlace({ search, showPricing, show1mPricing, onSelect}: MarketPlaceProps) {
  const { t } = useTranslation();
  const select = useSelector(selectModel);
  const supportModels = useSelector(selectSupportModels);

  const models = useMemo(() => {
    if (search.length === 0) return supportModels;
    // fuzzy search
    const raw = splitList(search.toLowerCase(), [" ", ",", ";", "-"]);
    return supportModels.filter((model) => {
      const name = model.name.toLowerCase();

      const tag = getTags(model);

      const tag_translated_name = tag
        .map((item) => t(`tag.${item}`))
        .join(" ")
        .toLowerCase();
      const id = model.id.toLowerCase();

      return raw.every(
        (item) =>
          name.includes(item) ||
          tag_translated_name.includes(item) ||
          id.includes(item),
      );
    });
  }, [supportModels, search]);

  return (
    <motion.div
      className={`model-list`}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5 }}
    >
      <AnimatePresence>
        {models.map((model, index) => (
          <ModelItem
            key={index}
            model={model}
            className={cn(select === model.id && "active")}
            showPricing={showPricing}
            show1mPricing={show1mPricing}
            index={index}
            onModelSelect={onSelect}
          />
        ))}
      </AnimatePresence>
    </motion.div>
  );
}

function ModelDetailPanel({
  model,
  onClose,
  show1mPricing,
}: {
  model: Model;
  onClose: () => void;
  show1mPricing: boolean;
}) {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const list = useSelector(selectModelList);
  const auth = useSelector(selectAuthenticated);
  const level = useSelector(levelSelector);
  const student = useSelector(teenagerSelector);
  const subscriptionData = useSelector(subscriptionDataSelector);

  const state = useMemo(() => list.includes(model.id), [model, list]);
  const pro = useMemo(
    () => includingModelFromPlan(subscriptionData, level, model.id),
    [subscriptionData, model, level, student],
  );
  const tags = useMemo(() => getTags(model), [model]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onClose]);

  const handleUse = () => {
    if (!auth && model.auth) {
      toast(t("login-require"), {
        action: { label: t("login"), onClick: goAuth },
      });
      return;
    }
    dispatch(setModel(model.id));
    router.navigate("/");
    onClose();
  };

  const unitLabel = show1mPricing ? "1M" : "1K";
  const unitValue = show1mPricing ? 1000 : 1;
  const quickEnter = 0.18;

  return (
    <motion.div
      className="fixed inset-0 z-40"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.12 }}
    >
      <div
        className="absolute inset-0 bg-black/30 backdrop-blur-[2px]"
        onClick={onClose}
      />
      <motion.div
        className="absolute right-0 top-0 h-full w-[420px] max-w-[92vw] bg-background flex flex-col shadow-2xl overflow-hidden"
        initial={{ x: "100%" }}
        animate={{ x: 0 }}
        exit={{ x: "100%" }}
        transition={{ type: "tween", duration: 0.24, ease: "easeOut" }}
      >
        {/* ── Hero header ── */}
        <div className="relative shrink-0 overflow-hidden">
          {/* gradient mesh background */}
          <div className="absolute inset-0 bg-gradient-to-br from-primary/8 via-primary/4 to-transparent" />
          <div className="absolute -top-8 -right-8 w-40 h-40 rounded-full bg-primary/6 blur-2xl" />
          <div className="absolute -bottom-4 -left-4 w-28 h-28 rounded-full bg-primary/4 blur-xl" />

          {/* close button */}
          <motion.button
            className="absolute top-4 right-4 z-10 p-1.5 rounded-full bg-background/60 backdrop-blur-sm text-muted-foreground hover:text-foreground hover:bg-background/90 transition-colors border border-border/40"
            whileHover={{ scale: 1.08 }}
            whileTap={{ scale: 0.92 }}
            onClick={onClose}
          >
            <X className="w-3.5 h-3.5" />
          </motion.button>

          <div className="relative flex flex-col items-center pt-10 pb-6 px-6">
            {/* avatar with animated ring */}
            <motion.div
              className="relative mb-4"
              initial={{ scale: 0.6, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              transition={{
                type: "spring",
                damping: 20,
                stiffness: 260,
                delay: 0.03,
              }}
            >
              <motion.div
                className="absolute inset-0 rounded-full border-2 border-primary/30"
                animate={{ scale: [1, 1.18, 1], opacity: [0.6, 0, 0.6] }}
                transition={{ duration: 2.4, repeat: Infinity, ease: "easeInOut" }}
              />
              <div className="relative z-10 rounded-full border-2 border-background shadow-lg overflow-hidden">
                <ModelAvatar model={model} size={64} />
              </div>
              {pro && (
                <div className="absolute -bottom-1 -right-1 z-20 w-5 h-5 rounded-full bg-amber-500 flex items-center justify-center shadow-md">
                  <Gem className="w-2.5 h-2.5 text-white" />
                </div>
              )}
            </motion.div>

            {/* name */}
            <motion.h2
              className="text-xl font-bold text-foreground text-center mb-1"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.06, duration: quickEnter }}
            >
              {model.name}
            </motion.h2>

            {/* model id */}
            <motion.div
              className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-primary/8 border border-primary/15 text-xs text-muted-foreground"
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.09, duration: quickEnter }}
            >
              <Tag className="w-2.5 h-2.5" />
              <span className="truncate max-w-[240px] font-mono">{model.id}</span>
            </motion.div>

            {/* tags row */}
            {tags.filter((tag) => tagIcons[tag]).length > 0 && (
              <motion.div
                className="flex flex-wrap justify-center gap-1.5 mt-3"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.12, duration: quickEnter }}
              >
                {tags.filter((tag) => tagIcons[tag]).map((tag, idx) => (
                  <motion.span
                    key={idx}
                    className={cn(
                      "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium",
                      {
                        "text-amber-700 bg-amber-100/80": tag === "official",
                        "text-blue-700 bg-blue-100/80": tag === "multi-modal",
                        "text-green-700 bg-green-100/80": tag === "web",
                        "text-purple-700 bg-purple-100/80": tag === "high-quality",
                        "text-red-700 bg-red-100/80": tag === "high-price",
                        "text-gray-700 bg-gray-100/80": tag === "open-source",
                        "text-indigo-700 bg-indigo-100/80": tag === "image-generation",
                        "text-yellow-700 bg-yellow-100/80": tag === "fast",
                        "text-orange-700 bg-orange-100/80": tag === "unstable",
                        "text-teal-700 bg-teal-100/80": tag === "high-context",
                        "text-emerald-700 bg-emerald-100/80": tag === "free",
                      }
                    )}
                    initial={{ opacity: 0, scale: 0.75 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ delay: 0.14 + idx * 0.03, duration: 0.16 }}
                  >
                    <Icon icon={tagIcons[tag]} className="w-2.5 h-2.5" />
                    {t(`tag.${tag}`)}
                  </motion.span>
                ))}
              </motion.div>
            )}
          </div>
        </div>

        {/* ── Scrollable body ── */}
        <ScrollArea className="flex-grow">
          <div className="px-5 py-4 space-y-4">
            {/* description */}
            {model.description && (
              <motion.p
                className="text-sm text-muted-foreground leading-relaxed"
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.12, duration: quickEnter }}
              >
                {model.description}
              </motion.p>
            )}

            {/* pricing cards */}
            {model.price && model.price.type === "token-billing" && (
              <motion.div
                className="grid grid-cols-2 gap-3"
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.16, duration: quickEnter }}
              >
                <div className="rounded-xl border bg-gradient-to-br from-blue-50/60 to-transparent p-3.5">
                  <div className="flex items-center gap-1.5 text-blue-600 mb-2">
                    <ArrowUpFromDot className="w-3.5 h-3.5" />
                    <span className="text-xs font-medium">输入</span>
                  </div>
                  <p className="text-2xl font-bold text-foreground leading-none">
                    {parseFloat((model.price.input * unitValue).toPrecision(8))}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">点数 / {unitLabel} tokens</p>
                </div>
                <div className="rounded-xl border bg-gradient-to-br from-violet-50/60 to-transparent p-3.5">
                  <div className="flex items-center gap-1.5 text-violet-600 mb-2">
                    <ArrowDownToDot className="w-3.5 h-3.5" />
                    <span className="text-xs font-medium">输出</span>
                  </div>
                  <p className="text-2xl font-bold text-foreground leading-none">
                    {parseFloat((model.price.output * unitValue).toPrecision(8))}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">点数 / {unitLabel} tokens</p>
                </div>
              </motion.div>
            )}

            {model.price && model.price.type !== "token-billing" && (
              <motion.div
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.16, duration: quickEnter }}
              >
                <PriceColumn
                  type={model.price.type}
                  input={model.price.input}
                  output={model.price.output}
                  pro={pro}
                  show1mPricing={show1mPricing}
                  anonymous
                />
              </motion.div>
            )}
          </div>
        </ScrollArea>

        {/* ── Footer ── */}
        <motion.div
          className="shrink-0 flex items-center gap-2 px-5 py-4 bg-background"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.14, duration: quickEnter }}
        >
          <motion.button
            className={cn(
              "shrink-0 w-10 h-10 rounded-xl border flex items-center justify-center transition-colors",
              state
                ? "text-amber-500 bg-amber-50 border-amber-200"
                : "text-muted-foreground border-input hover:bg-accent",
            )}
            whileHover={{ scale: 1.08 }}
            whileTap={{ scale: 0.92 }}
            onClick={() =>
              dispatch(state ? removeModelList(model.id) : addModelList(model.id))
            }
          >
            <Star className={cn("w-4 h-4", state && "fill-current")} />
          </motion.button>
          <motion.button
            className="flex-grow flex items-center justify-center gap-2 h-10 px-4 rounded-xl bg-primary text-primary-foreground text-sm font-semibold shadow-sm"
            whileHover={{ scale: 1.02, boxShadow: "0 4px 16px rgb(0 0 0 / 0.15)" }}
            whileTap={{ scale: 0.98 }}
            onClick={handleUse}
          >
            <ArrowRightLeft className="w-4 h-4" />
            {t("market.switch-model")}
          </motion.button>
        </motion.div>
      </motion.div>
    </motion.div>
  );
}

function MarketHeader() {
  const { t } = useTranslation();

  return (
    <motion.div
      className={`market-header`}
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <div
        className={`title select-none text-center text-primary font-bold flex flex-row items-center justify-center`}
      >
        <motion.div
          className={`header-bar`}
          initial={{ width: 0 }}
          animate={{ width: "0.75rem" }}
          transition={{ duration: 0.5, delay: 0.2 }}
        />
        {t("market.explore")}
        <motion.div
          className={`header-bar reverse`}
          initial={{ width: 0 }}
          animate={{ width: "0.75rem" }}
          transition={{ duration: 0.5, delay: 0.2 }}
        />
      </div>
    </motion.div>
  );
}

function MarketFooter() {
  const { t } = useTranslation();

  return (
    <motion.div
      className={`market-footer`}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <motion.a
        href={docsEndpoint}
        target={`_blank`}
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
      >
        <Link size={14} className={`mr-1`} />
        {t("pricing")}
      </motion.a>
    </motion.div>
  );
}

function Model() {
  const { t } = useTranslation();
  const [displayPricing, setDisplayPricing] = useState<boolean>(true);
  const [show1mPricing, setShow1mPricing] = useState<boolean>(false);
  const [searchText, setSearchText] = useState<string>("");
  const [searchTags, setSearchTags] = useState<string[]>([]);
  const [selectedModel, setSelectedModel] = useState<Model | null>(null);

  const search = useMemo(() => {
    return [
      searchText,
      ...searchTags.filter((tag) => tag !== "").map((v) => t(`tag.${v}`)),
    ].join(" ");
  }, [searchText, searchTags]);

  return (
    <>
    <ScrollArea className={`model-market`}>
      <motion.div
        className={`market-wrapper`}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        <motion.div
          className="absolute inset-0 overflow-hidden pointer-events-none"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 1 }}
        >
          {[...Array(50)].map((_, i) => (
            <motion.div
              key={i}
              className="absolute bg-primary/10 rounded-full"
              style={{
                width: Math.random() * 4 + 1 + "px",
                height: Math.random() * 4 + 1 + "px",
                top: Math.random() * 100 + "%",
                left: Math.random() * 100 + "%",
              }}
              animate={{
                y: [0, Math.random() * 100 - 50],
                x: [0, Math.random() * 100 - 50],
                opacity: [0.7, 0],
              }}
              transition={{
                duration: Math.random() * 10 + 10,
                repeat: Infinity,
                ease: "linear",
              }}
            />
          ))}
        </motion.div>
        <MarketHeader />
        <SearchBar
          text={searchText}
          onTextChange={setSearchText}
          tags={searchTags}
          onTagsChange={setSearchTags}
          displayPricing={displayPricing}
          onDisplayPricingChange={setDisplayPricing}
          show1mPricing={show1mPricing}
          onShow1mPricingChange={setShow1mPricing}
        />
        <MarketPlace
          search={search}
          showPricing={displayPricing}
          show1mPricing={show1mPricing}
          onSelect={setSelectedModel}
        />
        <MarketFooter />
      </motion.div>
    </ScrollArea>
    <AnimatePresence>
      {selectedModel && (
        <ModelDetailPanel
          key="model-detail"
          model={selectedModel}
          onClose={() => setSelectedModel(null)}
          show1mPricing={show1mPricing}
        />
      )}
    </AnimatePresence>
    </>
  );
}

export default Model;
