import "@/assets/pages/package.less";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import React, { useState } from "react";
import { cn } from "@/components/ui/lib/utils.ts";
import Avatar from "@/components/Avatar.tsx";
import { useDispatch, useSelector } from "react-redux";
import {
  logout,
  selectAuthenticated,
  selectInit,
  selectUsername,
} from "@/store/auth.ts";
import { Badge } from "@/components/ui/badge.tsx";
import { copyClipboard, useClipboard } from "@/utils/dom.ts";
import { useGroup } from "@/utils/groups.ts";
import { useTranslation } from "react-i18next";
import Icon from "@/components/utils/Icon.tsx";
import {
  CalendarClock,
  Clock,
  Cloud,
  CloudRain,
  Copy,
  ExternalLink,
  HandIcon,
  HelpCircle,
  KeyRound,
  Mail,
  Plug,
  Power,
  RotateCw,
  Share2,
  Trash2,
  Undo2,
  UserRoundCog,
  UserRoundIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { useEffectAsync } from "@/utils/hook.ts";
import {
  getUserInfo,
  initialUserInfo,
  sendCode,
  updateAccountEmail,
  updateAccountPassword,
  UserInfo,
} from "@/api/auth.ts";
import { CommonResponse, withNotify } from "@/api/common.ts";
import { goAuth } from "@/utils/app.ts";
import { quotaSelector } from "@/store/quota.ts";
import Tips from "@/components/Tips.tsx";
import { getSharedLink, SharingPreviewForm } from "@/api/sharing.ts";
import { openWindow } from "@/utils/device.ts";
import { dataSelector, deleteData, syncData } from "@/store/sharing.ts";
import { DeeptrainOnly } from "@/conf/deeptrain.tsx";
import { deeptrainEndpoint, docsEndpoint } from "@/conf/env.ts";
import { getApiKey, keySelector, regenerateApiKey } from "@/store/api.ts";
import { Input } from "@/components/ui/input.tsx";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog.tsx";
import {
  Dialog,
  DialogAction,
  DialogCancel,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog.tsx";
import { toast } from "sonner";
import Emoji from "@/components/Emoji";
import { motion } from "framer-motion";
import { isEmailValid, isTextInRange } from "@/utils/form.ts";

type AccountCardProps = {
  title: string;
  description: string;
  icon?: React.ReactElement;
  children: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
  classNameWrapper?: string;
};

function AccountCard({
  title,
  description,
  icon,
  children,
  footer,
  className,
  classNameWrapper,
}: AccountCardProps) {
  const { t } = useTranslation();

  return (
    <div
      className={cn(
        `flex flex-col bg-background rounded-lg shadow border overflow-hidden`,
        classNameWrapper,
      )}
    >
      <div
        className={`select-none inline-flex flex-row items-center h-fit w-full border-b px-4 py-2.5 bg-muted/20`}
      >
        <div className="flex items-center mr-2.5">
          {icon && (
            <Icon
              icon={icon}
              className="w-8 h-8 p-2 rounded-lg bg-muted text-secondary"
            />
          )}
        </div>
        <div className="flex flex-col">
          <p className="text-sm font-medium">{t(title)}</p>
          {description && (
            <p className="text-xs text-secondary">{t(description)}</p>
          )}
        </div>
      </div>
      <div className={cn("p-4", className)}>
        {children}
      </div>
      {footer && (
        <div className={`flex flex-row items-center px-4 pb-4 pt-2`}>
          {footer}
        </div>
      )}
    </div>
  );
}

type ShareContentProps = {
  data: SharingPreviewForm[];
};

function ShareContent({ data }: ShareContentProps) {
  const { t } = useTranslation();
  const dispatch = useDispatch();

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return `${date.getMonth() + 1}-${date.getDate()} ${date
      .getHours()
      .toString()
      .padStart(2, "0")}:${date.getMinutes().toString().padStart(2, "0")}`;
  };

  return (
    <div className="space-y-3 pt-2 pb-6">
      {data.map((row) => (
        <motion.div
          key={row.conversation_id}
          onClick={() => openWindow(getSharedLink(row.hash), "_blank")}
          className="flex items-center justify-between w-full border border-input p-4 rounded-lg hover:bg-muted/20 duration-200 cursor-pointer transition-colors"
          whileHover={{ y: -2 }}
          transition={{ type: "spring", stiffness: 320, damping: 24 }}
        >
          <div className="flex-grow mr-4">
            <div className="flex items-center mb-1">
              <h3 className="text-sm font-medium line-clamp-1">{row.name}</h3>
            </div>
            <div className="flex items-center text-xs text-muted-foreground">
              <Clock className="h-3 w-3 mr-1" />
              {formatTime(row.time)}
            </div>
          </div>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="light-destructive"
                size="icon"
                onClick={(e) => e.stopPropagation()}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t("account.share-delete")}</AlertDialogTitle>
                <AlertDialogDescription>
                  {t("account.share-delete-description")}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
                <AlertDialogAction
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteData(dispatch, row.hash);
                  }}
                >
                  {t("confirm")}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </motion.div>
      ))}
    </div>
  );
}

function Account() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const init = useSelector(selectInit);
  const username = useSelector(selectUsername);
  const auth = useSelector(selectAuthenticated);
  const quota = useSelector(quotaSelector);
  const copy = useClipboard();
  const group = useGroup(true);

  const apiKey = useSelector(keySelector);
  const [loadingApiKey, setLoadingApiKey] = useState(false);
  const [openResetApiKey, setOpenResetApiKey] = useState(false);

  const pageVariants = {
    hidden: { opacity: 0, y: 18 },
    visible: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.35,
        ease: "easeOut",
        when: "beforeChildren",
        staggerChildren: 0.08,
      },
    },
  };

  const cardVariants = {
    hidden: { opacity: 0, y: 22 },
    visible: {
      opacity: 1,
      y: 0,
      transition: { duration: 0.4, ease: "easeOut" },
    },
  };

  const contentVariants = {
    hidden: { opacity: 0, y: 14 },
    visible: {
      opacity: 1,
      y: 0,
      transition: { duration: 0.3, ease: "easeOut" },
    },
  };

  const getSystemKey = async () => {
    if (!init) return;

    setLoadingApiKey(true);
    await getApiKey(dispatch);
    setLoadingApiKey(false);
  };

  useEffectAsync(getSystemKey, [init]);

  async function copySystemKey() {
    await copyClipboard(apiKey);
    toast.success(t("api.copied"), {
      description: t("api.copied-description"),
    });
  }

  async function resetSystemKey() {
    const resp = await regenerateApiKey(dispatch);
    withNotify(t, resp as CommonResponse, true);

    if (resp.status) {
      setOpenResetApiKey(false);
    }
  }

  const [info, setInfo] = React.useState<UserInfo>({
    ...initialUserInfo,
  });

  const sharingData = useSelector(dataSelector);

  useEffectAsync(async () => {
    if (auth) {
      if (sharingData.length > 0) return;
      const resp = await syncData(dispatch);
      if (resp) {
        toast.error(t("share.sync-error"), {
          description: resp,
        });
      }
    }
  }, [auth]);

  const updateUserInfo = async () => {
    if (!auth) {
      return;
    }

    const resp = await getUserInfo();
    console.log(`[account api] get user info:`, resp);
    withNotify(t, resp);

    if (resp.status) {
      setInfo(resp.data);
    }
  };
  useEffectAsync(updateUserInfo, [auth]);

  const [emailDialogOpen, setEmailDialogOpen] = useState(false);
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false);
  const [emailForm, setEmailForm] = useState({ email: "", code: "" });
  const [passwordForm, setPasswordForm] = useState({
    code: "",
    password: "",
    repassword: "",
  });

  async function sendEmailChangeCode() {
    const email = emailForm.email.trim();
    if (!isEmailValid(email)) {
      toast.error(t("error"), { description: t("auth.invalid-email") });
      return;
    }

    await sendCode(t, email, true);
  }

  async function sendPasswordChangeCode() {
    const email = info.email.trim();
    if (!isEmailValid(email)) {
      toast.error(t("error"), {
        description: t("account.email-not-bound"),
      });
      return;
    }

    await sendCode(t, email);
  }

  async function submitEmailChange() {
    const email = emailForm.email.trim();
    const code = emailForm.code.trim();

    if (!isEmailValid(email)) {
      toast.error(t("error"), { description: t("auth.invalid-email") });
      return;
    }

    if (code.length === 0) {
      toast.error(t("error"), { description: t("account.code-required") });
      return;
    }

    const resp = await updateAccountEmail({ email, code });
    withNotify(t, resp, true, t("account.email-updated"));

    if (resp.status) {
      setEmailDialogOpen(false);
      setEmailForm({ email: "", code: "" });
      await updateUserInfo();
    }
  }

  async function submitPasswordChange() {
    const code = passwordForm.code.trim();
    const password = passwordForm.password.trim();
    const repassword = passwordForm.repassword.trim();

    if (code.length === 0) {
      toast.error(t("error"), { description: t("account.code-required") });
      return;
    }

    if (!isTextInRange(password, 6, 36)) {
      toast.error(t("error"), {
        description: t("account.password-invalid"),
      });
      return;
    }

    if (password !== repassword) {
      toast.error(t("error"), {
        description: t("account.password-mismatch"),
      });
      return;
    }

    const resp = await updateAccountPassword({ code, password });
    withNotify(t, resp, true, t("account.password-updated"));

    if (resp.status) {
      setPasswordDialogOpen(false);
      setPasswordForm({ code: "", password: "", repassword: "" });
    }
  }

  return (
    <ScrollArea
      className={`relative w-full h-full flex flex-col bg-background`}
    >
      <motion.div
        className={`px-4 py-6 md:py-12 lg:py-16 h-full flex flex-col w-full max-w-3xl mx-auto space-y-4`}
        variants={pageVariants}
        initial="hidden"
        animate="visible"
      >
        <motion.div variants={cardVariants}>
          <AccountCard
            icon={<UserRoundIcon />}
            title={"account.my-account"}
            description={t("account.my-account-description")}
            footer={
              !auth ? (
                <Button
                  classNameWrapper={`ml-auto`}
                  className={`flex flex-row items-center`}
                  onClick={goAuth}
                >
                  <HandIcon className={`h-4 w-4 mr-1.5`} />
                  {t("login")}
                </Button>
              ) : (
                <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
                  <Dialog
                    open={emailDialogOpen}
                    onOpenChange={setEmailDialogOpen}
                  >
                    <DialogTrigger asChild>
                      <Button
                        variant="outline"
                        className="flex flex-row items-center"
                      >
                        <Mail className="h-4 w-4 mr-1.5" />
                        {t("account.change-email")}
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>{t("account.change-email")}</DialogTitle>
                        <DialogDescription>
                          {t("account.change-email-description")}
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4">
                        <div className="rounded-lg border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                          {t("account.current-email")}:{" "}
                          <span className="text-foreground">
                            {info.email || "-"}
                          </span>
                        </div>
                        <Input
                          placeholder={t("account.new-email")}
                          value={emailForm.email}
                          onChange={(e) =>
                            setEmailForm((prev) => ({
                              ...prev,
                              email: e.target.value,
                            }))
                          }
                        />
                        <div className="flex gap-2">
                          <Input
                            placeholder={t("account.verification-code")}
                            value={emailForm.code}
                            onChange={(e) =>
                              setEmailForm((prev) => ({
                                ...prev,
                                code: e.target.value,
                              }))
                            }
                          />
                          <Button
                            variant="outline"
                            className="shrink-0 min-w-20 whitespace-nowrap"
                            loading
                            onClick={sendEmailChangeCode}
                          >
                            {t("auth.send-code")}
                          </Button>
                        </div>
                      </div>
                      <DialogFooter>
                        <DialogCancel>{t("cancel")}</DialogCancel>
                        <DialogAction loading onClick={submitEmailChange}>
                          {t("confirm")}
                        </DialogAction>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>

                  <Dialog
                    open={passwordDialogOpen}
                    onOpenChange={setPasswordDialogOpen}
                  >
                    <DialogTrigger asChild>
                      <Button
                        variant="outline"
                        className="flex flex-row items-center"
                      >
                        <KeyRound className="h-4 w-4 mr-1.5" />
                        {t("account.change-password")}
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>
                          {t("account.change-password")}
                        </DialogTitle>
                        <DialogDescription>
                          {t("account.change-password-description")}
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4">
                        <div className="rounded-lg border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                          {t("account.send-code-to-current-email")}:{" "}
                          <span className="text-foreground">
                            {info.email || "-"}
                          </span>
                        </div>
                        <div className="flex gap-2">
                          <Input
                            placeholder={t("account.verification-code")}
                            value={passwordForm.code}
                            onChange={(e) =>
                              setPasswordForm((prev) => ({
                                ...prev,
                                code: e.target.value,
                              }))
                            }
                          />
                          <Button
                            variant="outline"
                            className="shrink-0 min-w-20 whitespace-nowrap"
                            loading
                            onClick={sendPasswordChangeCode}
                          >
                            {t("auth.send-code")}
                          </Button>
                        </div>
                        <Input
                          type="password"
                          placeholder={t("account.new-password")}
                          value={passwordForm.password}
                          onChange={(e) =>
                            setPasswordForm((prev) => ({
                              ...prev,
                              password: e.target.value,
                            }))
                          }
                        />
                        <Input
                          type="password"
                          placeholder={t("account.confirm-new-password")}
                          value={passwordForm.repassword}
                          onChange={(e) =>
                            setPasswordForm((prev) => ({
                              ...prev,
                              repassword: e.target.value,
                            }))
                          }
                        />
                      </div>
                      <DialogFooter>
                        <DialogCancel>{t("cancel")}</DialogCancel>
                        <DialogAction loading onClick={submitPasswordChange}>
                          {t("confirm")}
                        </DialogAction>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>

                  <Button
                    className={`flex flex-row items-center`}
                    onClick={() => dispatch(logout())}
                  >
                    <Undo2 className={`h-4 w-4 mr-1.5`} />
                    {t("logout")}
                  </Button>
                </div>
              )
            }
          >
            <div className="flex flex-col space-y-4">
              <motion.div
                className="flex items-center space-x-4"
                variants={contentVariants}
              >
                <Avatar
                  username={username}
                  className="w-16 h-16 shrink-0 shadow text-lg rounded-full"
                />
                <div className="flex flex-row w-full">
                  <div className="flex flex-col w-fit">
                    <p
                      className="text-xl font-semibold cursor-pointer select-none"
                      onClick={() => copy(username)}
                    >
                      {auth ? username : t("anonymous")}
                    </p>
                    <p className="text-sm text-muted-foreground">#{info.id}</p>
                  </div>
                </div>
              </motion.div>

              <motion.div className="flex flex-wrap gap-2" variants={contentVariants}>
                <Badge className="px-3 py-1 text-sm font-medium">
                  {t(`admin.channels.groups.${group}`)}
                </Badge>
                <Badge
                  variant="outline"
                  className="px-3 py-1 text-sm font-medium"
                >
                  {t(`account.registerDays`, {
                    days: Math.ceil(info.register_days),
                  })}
                </Badge>
              </motion.div>
            </div>
            <motion.div
              className="mt-6 grid grid-cols-1 gap-4 md:grid-cols-3"
              variants={contentVariants}
            >
              <motion.div
                className="bg-card shadow-sm rounded-lg p-4 transition-all border"
                variants={contentVariants}
                whileHover={{ scale: 1.01 }}
                transition={{ type: "spring", stiffness: 320, damping: 24 }}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-muted-foreground">
                    {t("account.current-quota")}
                  </span>
                  <Cloud className="w-10 h-10 p-2 rounded-lg bg-muted/40 text-secondary stroke-[1]" />
                </div>
                <p className="text-md">{quota.toFixed(2)}</p>
              </motion.div>
              <motion.div
                className="bg-card shadow-sm rounded-lg p-4 transition-all border"
                variants={contentVariants}
                whileHover={{ scale: 1.01 }}
                transition={{ type: "spring", stiffness: 320, damping: 24 }}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-muted-foreground">
                    {t("account.used-quota")}
                  </span>
                  <CloudRain className="w-10 h-10 p-2 rounded-lg bg-muted/40 text-secondary stroke-[1]" />
                </div>
                <p className="text-md">{info.used_quota.toFixed(2)}</p>
              </motion.div>
              <motion.div
                className="bg-card shadow-sm rounded-lg p-4 transition-all border"
                variants={contentVariants}
                whileHover={{ scale: 1.01 }}
                transition={{ type: "spring", stiffness: 320, damping: 24 }}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-muted-foreground">
                    {t("account.plan-total-month")}
                  </span>
                  <CalendarClock className="w-10 h-10 p-2 rounded-lg bg-muted/40 text-secondary stroke-[1]" />
                </div>
                <div className="flex items-center">
                  <p className="text-md mr-2">{info.plan_total_month}</p>
                  <Tips
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    content={t("account.plan-total-month-tips")}
                  />
                </div>
              </motion.div>
            </motion.div>
          </AccountCard>
        </motion.div>
        <DeeptrainOnly>
          <motion.div variants={cardVariants}>
            <AccountCard
              title={"account.deeptrain"}
              description={t("account.deeptrain-description")}
              icon={<UserRoundCog />}
              footer={
                auth ? (
                  <Button
                    className={`flex flex-row items-center`}
                    classNameWrapper={`ml-auto`}
                    onClick={() => openWindow(`${deeptrainEndpoint}/home`)}
                  >
                    <ExternalLink className={`h-4 w-4 mr-1.5`} />
                    {t("manage")}
                  </Button>
                ) : (
                  <Button classNameWrapper={`ml-auto`} onClick={goAuth}>
                    <HandIcon className={`h-4 w-4 mr-1.5`} />
                    {t("login")}
                  </Button>
                )
              }
            >
              <motion.div
                className={`flex flex-row items-center space-x-2`}
                variants={contentVariants}
              >
                <img
                  src={`${deeptrainEndpoint}/favicon.ico`}
                  alt={``}
                  className={`w-12 h-12 select-none cursor-pointer`}
                  onClick={() => openWindow(`${deeptrainEndpoint}/home`)}
                />
                <div className={`inline-flex flex-col`}>
                  <p className={`text-common text-sm font-bold`}>DeepTrain SSO</p>
                  <p className={`text-secondary text-xs`}>
                    {t("account.deeptrain-description")}
                  </p>
                </div>
              </motion.div>
            </AccountCard>
          </motion.div>
        </DeeptrainOnly>
        <motion.div variants={cardVariants}>
          <AccountCard
            title={"api.title"}
            description={t("account.api-description")}
            icon={<Plug />}
          >
            <motion.div className={`api-dialog`} variants={contentVariants}>
              <div className={`api-wrapper flex flex-row space-x-1`}>
                <Button
                  variant={`outline`}
                  size={`icon-sm`}
                  className={`shrink-0`}
                  onClick={getSystemKey}
                >
                  <RotateCw
                    className={cn("h-3.5 w-3.5", loadingApiKey && "animate-spin")}
                  />
                </Button>
                <Input
                  type={`password`}
                  value={apiKey}
                  readOnly={true}
                  classNameWrapper={`grow`}
                  className={`text-xs h-8`}
                />
                <Button
                  variant={`default`}
                  className={`shrink-0`}
                  size={`icon-sm`}
                  onClick={copySystemKey}
                >
                  <Copy className={`h-3.5 w-3.5`} />
                </Button>
              </div>
              <div className={`flex flex-row mt-2 items-center justify-center`}>
                <AlertDialog
                  open={openResetApiKey}
                  onOpenChange={setOpenResetApiKey}
                >
                  <AlertDialogTrigger asChild>
                    <Button
                      variant={`destructive`}
                      size={`default-sm`}
                      className={`text-xs mr-2`}
                    >
                      <Power className={`h-3.5 w-3.5 mr-2`} />
                      {t("api.reset")}
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>{t("api.reset")}</AlertDialogTitle>
                      <AlertDialogDescription>
                        {t("api.reset-description")}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <Button
                        variant={`destructive`}
                        loading={true}
                        onClick={resetSystemKey}
                        unClickable
                      >
                        {t("confirm")}
                      </Button>
                      <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>

                <Button
                  variant={`outline`}
                  size={`default-sm`}
                  className={`text-xs`}
                  asChild
                >
                  <a href={docsEndpoint} target={`_blank`}>
                    <ExternalLink className={`h-3.5 w-3.5 mr-2`} />
                    {t("api.learn-more")}
                  </a>
                </Button>
              </div>
            </motion.div>
          </AccountCard>
        </motion.div>
        <motion.div variants={cardVariants}>
          <AccountCard
            icon={<Share2 />}
            title={"share.manage"}
            description={t("account.share-description")}
            className={`bg-background px-1`}
          >
            {sharingData.length > 0 ? (
              <ScrollArea className={`h-48 md:h-64 px-4`}>
                <div className={`w-full`}>
                  <ShareContent data={sharingData} />
                </div>
              </ScrollArea>
            ) : (
              <motion.div
                className={`flex flex-col items-center text-sm select-none py-8`}
                variants={contentVariants}
              >
                <Emoji
                  emoji={`1f4c2`}
                  className="w-12 h-12 p-2 rounded-md bg-muted/80 mb-4"
                />
                <p>{t("share.empty")}</p>

                <p
                  className={`flex flex-row items-center text-xs text-secondary mt-1.5`}
                >
                  <HelpCircle className={`h-3 w-3 mr-1`} />
                  {t("share.share-tip")}
                </p>
              </motion.div>
            )}
          </AccountCard>
        </motion.div>
      </motion.div>
    </ScrollArea>
  );
}

export default Account;
