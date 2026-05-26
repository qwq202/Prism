import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { Button } from "@/components/ui/button.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import {
  ArrowLeft,
  ArrowRight,
  BadgeCheck,
  CalendarClock,
  Cloud,
  HelpCircle,
  RefreshCcw,
  Route,
  ShieldCheck,
  Sparkles,
  WalletCards,
} from "lucide-react";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

type GuideCardProps = {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
  accent?: string;
};

const revealVariants = {
  hidden: { opacity: 0, y: 18 },
  visible: { opacity: 1, y: 0 },
};

const revealViewport = { once: true, amount: 0.18 };

function GuideCard({ icon, title, children, accent }: GuideCardProps) {
  return (
    <motion.div
      className="rounded-2xl border bg-background p-5"
      variants={revealVariants}
      initial="hidden"
      whileInView="visible"
      viewport={revealViewport}
      transition={{ duration: 0.38, ease: "easeOut" }}
    >
      <div
        className={cn(
          "mb-4 flex h-10 w-10 items-center justify-center rounded-xl",
          accent ?? "bg-muted text-muted-foreground",
        )}
      >
        {icon}
      </div>
      <h2 className="text-base font-semibold tracking-tight">{title}</h2>
      <p className="mt-2 text-sm leading-6 text-muted-foreground">{children}</p>
    </motion.div>
  );
}

type GuideRowProps = {
  index: number;
  title: string;
  children: React.ReactNode;
};

function GuideRow({ index, title, children }: GuideRowProps) {
  return (
    <motion.div
      className="grid gap-3 border-t py-5 first:border-t-0 first:pt-0 last:pb-0 md:grid-cols-[4rem_1fr]"
      variants={revealVariants}
      initial="hidden"
      whileInView="visible"
      viewport={revealViewport}
      transition={{ duration: 0.34, ease: "easeOut" }}
    >
      <div className="font-mono text-2xl font-semibold text-muted-foreground/40">
        {String(index).padStart(2, "0")}
      </div>
      <div>
        <h3 className="text-sm font-semibold">{title}</h3>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          {children}
        </p>
      </div>
    </motion.div>
  );
}

function SubscriptionGuide() {
  const { t } = useTranslation();

  return (
    <ScrollArea className="h-full w-full bg-muted/25">
      <div className="mx-auto w-full max-w-5xl px-4 py-8 md:py-12">
        <Link
          to="/subscription"
          className="mb-6 inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          {t("sub.guide-back")}
        </Link>

        <motion.div
          className="rounded-3xl border bg-background px-6 py-8 md:px-10 md:py-12"
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35 }}
        >
          <div className="mx-auto max-w-2xl text-center">
            <div className="mx-auto mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary">
              <Route className="h-6 w-6" />
            </div>
            <h1 className="text-3xl font-bold tracking-tight md:text-4xl">
              {t("sub.guide-title")}
            </h1>
            <p className="mt-4 text-sm leading-7 text-muted-foreground md:text-base">
              {t("sub.guide-desc")}
            </p>
            <div className="mt-6 flex flex-col justify-center gap-2 sm:flex-row">
              <Button asChild>
                <Link to="/subscription">
                  {t("sub.choose-plan")}
                  <ArrowRight className="ml-1.5 h-4 w-4" />
                </Link>
              </Button>
              <Button asChild variant="outline">
                <Link to="/wallet">{t("buy.title")}</Link>
              </Button>
            </div>
          </div>
        </motion.div>

        <motion.section
          className="mt-8"
          variants={revealVariants}
          initial="hidden"
          whileInView="visible"
          viewport={revealViewport}
          transition={{ duration: 0.35, ease: "easeOut" }}
        >
          <motion.div
            className="mb-4 flex items-end justify-between gap-4"
            variants={revealVariants}
          >
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {t("sub.guide-pay-eyebrow")}
              </p>
              <h2 className="mt-1 text-xl font-semibold tracking-tight">
                {t("sub.guide-pay-title")}
              </h2>
            </div>
          </motion.div>
          <div className="grid gap-4 md:grid-cols-3">
            <GuideCard
              icon={<Cloud className="h-5 w-5" />}
              title={t("sub.guide-points-title")}
              accent="bg-sky-100 text-sky-600 dark:bg-sky-950/40 dark:text-sky-300"
            >
              {t("sub.guide-points-desc")}
            </GuideCard>
            <GuideCard
              icon={<CalendarClock className="h-5 w-5" />}
              title={t("sub.guide-subscription-title")}
              accent="bg-emerald-100 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300"
            >
              {t("sub.guide-subscription-desc")}
            </GuideCard>
            <GuideCard
              icon={<WalletCards className="h-5 w-5" />}
              title={t("sub.guide-hybrid-title")}
              accent="bg-amber-100 text-amber-600 dark:bg-amber-950/40 dark:text-amber-300"
            >
              {t("sub.guide-hybrid-desc")}
            </GuideCard>
          </div>
        </motion.section>

        <section className="mt-8 grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
          <motion.div
            className="rounded-2xl border bg-background p-6"
            variants={revealVariants}
            initial="hidden"
            whileInView="visible"
            viewport={revealViewport}
            transition={{ duration: 0.38, ease: "easeOut" }}
          >
            <div className="mb-5 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-violet-100 text-violet-600 dark:bg-violet-950/40 dark:text-violet-300">
                <RefreshCcw className="h-5 w-5" />
              </div>
              <div>
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                  {t("sub.quota-manage")}
                </p>
                <h2 className="text-lg font-semibold tracking-tight">
                  {t("sub.guide-window-title")}
                </h2>
              </div>
            </div>
            <GuideRow index={1} title={t("sub.guide-short-window-title")}>
              {t("sub.guide-short-window-desc")}
            </GuideRow>
            <GuideRow index={2} title={t("sub.guide-weekly-window-title")}>
              {t("sub.guide-weekly-window-desc")}
            </GuideRow>
            <GuideRow index={3} title={t("sub.guide-reset-window-title")}>
              {t("sub.guide-reset-window-desc")}
            </GuideRow>
          </motion.div>

          <motion.div
            className="rounded-2xl border bg-background p-6"
            variants={revealVariants}
            initial="hidden"
            whileInView="visible"
            viewport={revealViewport}
            transition={{ duration: 0.38, ease: "easeOut", delay: 0.04 }}
          >
            <div className="mb-5 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
                <Sparkles className="h-5 w-5" />
              </div>
              <div>
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                  {t("sub.guide-choice-eyebrow")}
                </p>
                <h2 className="text-lg font-semibold tracking-tight">
                  {t("sub.guide-choice-title")}
                </h2>
              </div>
            </div>
            <div className="space-y-3">
              {[
                ["sub.guide-light-title", "sub.guide-light-desc"],
                ["sub.guide-steady-title", "sub.guide-steady-desc"],
                ["sub.guide-heavy-title", "sub.guide-heavy-desc"],
              ].map(([titleKey, descKey]) => (
                <motion.div
                  key={titleKey}
                  className="border-t py-4 first:border-t-0 first:pt-0 last:pb-0"
                  variants={revealVariants}
                  initial="hidden"
                  whileInView="visible"
                  viewport={revealViewport}
                  transition={{ duration: 0.32, ease: "easeOut" }}
                >
                  <div className="flex items-start gap-2">
                    <BadgeCheck className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />
                    <div>
                      <h3 className="text-sm font-semibold">{t(titleKey)}</h3>
                      <p className="mt-1 text-sm leading-6 text-muted-foreground">
                        {t(descKey)}
                      </p>
                    </div>
                  </div>
                </motion.div>
              ))}
            </div>
          </motion.div>
        </section>

        <motion.section
          className="mt-8"
          variants={revealVariants}
          initial="hidden"
          whileInView="visible"
          viewport={revealViewport}
          transition={{ duration: 0.35, ease: "easeOut" }}
        >
          <div className="mb-5 flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-muted text-muted-foreground">
              <HelpCircle className="h-5 w-5" />
            </div>
            <h2 className="text-lg font-semibold tracking-tight">
              {t("sub.guide-faq-title")}
            </h2>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            <GuideCard
              icon={<ShieldCheck className="h-5 w-5" />}
              title={t("sub.guide-faq-model-title")}
            >
              {t("sub.guide-faq-model-desc")}
            </GuideCard>
            <GuideCard
              icon={<RefreshCcw className="h-5 w-5" />}
              title={t("sub.guide-faq-change-title")}
            >
              {t("sub.guide-faq-change-desc")}
            </GuideCard>
            <GuideCard
              icon={<WalletCards className="h-5 w-5" />}
              title={t("sub.guide-faq-record-title")}
            >
              {t("sub.guide-faq-record-desc")}
            </GuideCard>
          </div>
        </motion.section>
      </div>
    </ScrollArea>
  );
}

export default SubscriptionGuide;
