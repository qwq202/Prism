import {
  CalendarPlus,
  Cloud,
  CloudCog,
  Cloudy,
  Gauge,
  Package,
  Percent,
  Plus,
} from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import React from "react";
import { useSelector } from "react-redux";
import { subscriptionDataSelector } from "@/store/globals.ts";
import { levelSelector } from "@/store/subscription.ts";
import { useTranslation } from "react-i18next";
import router from "@/router.tsx";
import Emoji from "../Emoji";
import { cn } from "../ui/lib/utils";
import ModelAvatar from "../ModelAvatar";
import { selectSupportModels } from "@/store/chat";
import { includingModelFromPlan } from "@/conf/subscription.tsx";

type QuotaExceededFormProps = {
  model: string;
  required: string;
  quota: string;
  plan: boolean;
  requiredLabel: string;
  quotaLabel?: string;
  title: string;
  tip: string;
};

function QuotaExceededForm({
  model,
  required,
  quota,
  plan,
  requiredLabel,
  quotaLabel,
  title,
  tip,
}: QuotaExceededFormProps) {
  const { t } = useTranslation();
  const supportModels = useSelector(selectSupportModels);
  const modelInfo = supportModels.find((m) => m.id === model);
  const displayModelName = modelInfo?.name ?? model;

  return (
    <div className={`flex flex-col items-center pt-4 pb-1`}>
      <Emoji emoji={"1f915"} className={`w-16 h-16 m-6 mb-4`} />
      <p className={`text-lg font-semibold !mb-1`}>{title}</p>
      <p className={`text-sm text-secondary px-2.5 text-center`}>{tip}</p>
      <div className="w-full h-fit border bg-muted/50 rounded-lg p-1.5 flex flex-col space-y-2.5 py-2.5">
        <div
          className={`flex flex-row w-full items-center justify-center px-4`}
        >
          <Package className={`h-4 w-4 mr-1`} />
          {t("model")}
          <div className={`grow`} />
          <div
            className={`!mb-0 flex min-w-0 max-w-[58%] flex-row items-center space-x-1`}
            title={displayModelName}
          >
            <ModelAvatar
              size={24}
              model={
                modelInfo ?? {
                  id: model,
                  name: model,
                }
              }
            />
            <p className={`!mb-0 min-w-0 truncate`}>{displayModelName}</p>
          </div>
        </div>
        <div
          className={`flex flex-row w-full items-center justify-center px-4`}
        >
          <Cloudy className={`h-4 w-4 mr-1`} />
          {quotaLabel ?? t("your-quota")}
          <div className={`grow`} />
          <p className={`flex flex-row items-center font-medium !mb-0`}>
            {quota}
            <Cloud className={`h-4 w-4 ml-1`} />
          </p>
        </div>
        <div
          className={`flex flex-row w-full items-center justify-center px-4`}
        >
          <CloudCog className={`h-4 w-4 mr-1`} />
          {requiredLabel}
          <div className={`grow`} />
          <p className={`flex flex-row items-center font-medium !mb-0`}>
            {required}
            <Cloud className={`h-4 w-4 ml-1`} />
          </p>
        </div>
      </div>

      <div
        className={cn(
          `mt-4 w-full h-fit grid grid-cols-1 gap-2`,
          plan && "md:grid-cols-2",
        )}
      >
        <Button
          classNameWrapper={`w-full`}
          className={`w-full`}
          onClick={() => router.navigate("/wallet")}
        >
          <Plus className={`h-4 w-4 mr-1`} />
          {t("buy.dialog-title")}
        </Button>
        {plan && (
          <Button
            variant={`outline`}
            classNameWrapper={`w-full`}
            className={`w-full`}
            onClick={() => router.navigate("/wallet#plan")}
          >
            <CalendarPlus className={`h-4 w-4 mr-1`} />
            {t("sub.dialog-title")}
          </Button>
        )}
      </div>
    </div>
  );
}

type SubscriptionQuotaExceededFormProps = {
  model: string;
  remainingPercent: string;
  minimumPercent: string;
};

function SubscriptionQuotaExceededForm({
  model,
  remainingPercent,
  minimumPercent,
}: SubscriptionQuotaExceededFormProps) {
  const { t } = useTranslation();
  const supportModels = useSelector(selectSupportModels);
  const modelInfo = supportModels.find((m) => m.id === model);
  const displayModelName = modelInfo?.name ?? model;

  return (
    <div className={`flex flex-col items-center pt-4 pb-1`}>
      <Emoji emoji={"1f915"} className={`w-16 h-16 m-6 mb-4`} />
      <p className={`text-lg font-semibold !mb-1`}>
        {t("oops-subscription-quota-exceeded")}
      </p>
      <p className={`text-sm text-secondary px-2.5 text-center`}>
        {t("oops-subscription-quota-exceeded-tip")}
      </p>
      <div className="w-full h-fit border bg-muted/50 rounded-lg p-1.5 flex flex-col space-y-2.5 py-2.5">
        <div
          className={`flex flex-row w-full items-center justify-center px-4`}
        >
          <Package className={`h-4 w-4 mr-1`} />
          {t("model")}
          <div className={`grow`} />
          <div
            className={`!mb-0 flex min-w-0 max-w-[58%] flex-row items-center space-x-1`}
            title={displayModelName}
          >
            <ModelAvatar
              size={24}
              model={
                modelInfo ?? {
                  id: model,
                  name: model,
                }
              }
            />
            <p className={`!mb-0 min-w-0 truncate`}>{displayModelName}</p>
          </div>
        </div>
        <div
          className={`flex flex-row w-full items-center justify-center px-4`}
        >
          <Percent className={`h-4 w-4 mr-1`} />
          {t("subscription-5h-remaining")}
          <div className={`grow`} />
          <p className={`flex flex-row items-center font-medium !mb-0`}>
            {remainingPercent}%
          </p>
        </div>
        <div
          className={`flex flex-row w-full items-center justify-center px-4`}
        >
          <Gauge className={`h-4 w-4 mr-1`} />
          {t("subscription-5h-minimum")}
          <div className={`grow`} />
          <p className={`flex flex-row items-center font-medium !mb-0`}>
            {minimumPercent}%
          </p>
        </div>
      </div>

      <div className="mt-4 w-full h-fit grid grid-cols-1 gap-2 md:grid-cols-2">
        <Button
          classNameWrapper={`w-full`}
          className={`w-full`}
          onClick={() => router.navigate("/wallet#plan")}
        >
          <CalendarPlus className={`h-4 w-4 mr-1`} />
          {t("sub.dialog-title")}
        </Button>
        <Button
          variant={`outline`}
          classNameWrapper={`w-full`}
          className={`w-full`}
          onClick={() => router.navigate("/wallet")}
        >
          <Plus className={`h-4 w-4 mr-1`} />
          {t("buy.title")}
        </Button>
      </div>
    </div>
  );
}

type LabelProps = {
  children: React.ReactNode;
};

function Label({ children }: LabelProps) {
  const { t } = useTranslation();
  const subscription = useSelector(subscriptionDataSelector);
  const level = useSelector(levelSelector);
  const content = (children ?? "").toString();

  if (content.startsWith("user quota")) {
    // if the format is `user quota is not enough error (model: gpt-3.5-turbo-1106, minimum quota: 0.01, your quota: -77.77)`, return special component

    const match = content.match(
      /user quota is not enough error \(model: (.*), minimum quota: (.*), your quota: (.*)\)/,
    );
    if (match) {
      const [, model, minimum, quota] = match;
      const plan = includingModelFromPlan(subscription, level, model);

      return (
        <QuotaExceededForm
          model={model}
          required={minimum}
          quota={quota}
          plan={plan}
          requiredLabel={t("min-quota")}
          title={t("oops-user-quota-exceeded")}
          tip={t("oops-user-quota-exceeded-tip")}
        />
      );
    }
  }

  if (content.startsWith("estimated cost exceeds user quota")) {
    const match = content.match(
      /estimated cost exceeds user quota \(model: (.*), estimated cost: (.*), your quota: (.*)\)/,
    );

    if (match) {
      const [, model, estimatedCost, quota] = match;
      const plan = includingModelFromPlan(subscription, level, model);

      return (
        <QuotaExceededForm
          model={model}
          required={estimatedCost}
          quota={quota}
          plan={plan}
          requiredLabel={t("estimated-cost")}
          title={t("oops-user-quota-exceeded")}
          tip={t("oops-user-quota-exceeded-tip")}
        />
      );
    }
  }

  if (content.startsWith("subscription quota")) {
    const match = content.match(
      /subscription quota is not enough error \(model: (.*), remaining quota: (.*), minimum quota: (.*), remaining percent: (.*), minimum percent: (.*), credit fallback: disabled\)/,
    );

    if (match) {
      const [, model, , , remainingPercent, minimumPercent] = match;
      return (
        <SubscriptionQuotaExceededForm
          model={model}
          remainingPercent={remainingPercent}
          minimumPercent={minimumPercent}
        />
      );
    }
  }

  return <p>{children}</p>;
}

export default Label;
