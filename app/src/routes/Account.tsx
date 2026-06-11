import "@/assets/pages/package.less";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import React, { useState } from "react";
import { useLocation } from "react-router-dom";
import { cn } from "@/components/ui/lib/utils.ts";
import Avatar from "@/components/Avatar.tsx";
import { useDispatch, useSelector } from "react-redux";
import {
  logout,
  selectAuthenticated,
  selectUsername,
  validateToken,
} from "@/store/auth.ts";
import type { AppDispatch } from "@/store";
import { Badge } from "@/components/ui/badge.tsx";
import { useClipboard } from "@/utils/dom.ts";
import { useGroup } from "@/utils/groups.ts";
import { useTranslation } from "react-i18next";
import Icon from "@/components/utils/Icon.tsx";
import {
  CalendarClock,
  Clock,
  Cloud,
  CloudRain,
  Coins,
  ExternalLink,
  Fingerprint,
  HandIcon,
  HelpCircle,
  KeyRound,
  Mail,
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
  createPasskeyRegistrationOptions,
  deletePasskey,
  listPasskeys,
  PasskeyCredentialInfo,
  registerPasskey,
  requestPasswordReset as requestPasswordResetEmail,
  sendCode,
  updateAccountEmail,
  updateAccountPassword,
  UserInfo,
} from "@/api/auth.ts";
import { withNotify } from "@/api/common.ts";
import { goAuth } from "@/utils/app.ts";
import {
  allowSubscriptionQuotaFallbackSelector,
  quotaSelector,
  refreshQuota,
  setAllowSubscriptionQuotaFallback,
  setQuota,
} from "@/store/quota.ts";
import {
  isSubscribedSelector,
  refreshSubscription,
} from "@/store/subscription.ts";
import Tips from "@/components/Tips.tsx";
import { getSharedLink, SharingPreviewForm } from "@/api/sharing.ts";
import { openWindow } from "@/utils/device.ts";
import { dataSelector, deleteData, setData, syncData } from "@/store/sharing.ts";
import { DeeptrainOnly } from "@/conf/deeptrain.tsx";
import { deeptrainEndpoint } from "@/conf/env.ts";
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
import { getErrorMessage } from "@/utils/base.ts";
import { localizeError } from "@/utils/error.ts";
import { Switch } from "@/components/ui/switch.tsx";
import { updateSubscriptionQuotaFallback } from "@/api/quota.ts";

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
      <div className={cn("p-4", className)}>{children}</div>
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

function base64urlToBuffer(value: string): ArrayBuffer {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(
    normalized.length + ((4 - (normalized.length % 4)) % 4),
    "=",
  );
  const binary = window.atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return window
    .btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

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
  const dispatch: AppDispatch = useDispatch();
  const location = useLocation();
  const username = useSelector(selectUsername);
  const auth = useSelector(selectAuthenticated);
  const quota = useSelector(quotaSelector);
  const subscription = useSelector(isSubscribedSelector);
  const allowSubscriptionQuotaFallback = useSelector(
    allowSubscriptionQuotaFallbackSelector,
  );
  const copy = useClipboard();
  const group = useGroup(true);

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

  const [info, setInfo] = React.useState<UserInfo>({
    ...initialUserInfo,
  });

  const sharingData = useSelector(dataSelector);

  useEffectAsync(async () => {
    if (!auth) {
      dispatch(setData([]));
      return;
    }

    const resp = await syncData(dispatch);
    if (resp) {
      toast.error(t("share.sync-error"), {
        description: resp,
      });
    }
  }, [auth, location.key]);

  const updateUserInfo = async () => {
    if (!auth) {
      setInfo({ ...initialUserInfo });
      return;
    }

    const resp = await getUserInfo();
    if (!resp.status) {
      withNotify(t, resp);
    }

    if (resp.status) {
      setInfo(resp.data);
    }
  };
  useEffectAsync(updateUserInfo, [auth, location.key]);

  useEffectAsync(async () => {
    if (!auth) {
      dispatch(setQuota(0));
      return;
    }

    await dispatch(refreshQuota());
    await dispatch(refreshSubscription());
  }, [auth, location.key]);

  const [emailDialogOpen, setEmailDialogOpen] = useState(false);
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false);
  const [passwordResetDialogOpen, setPasswordResetDialogOpen] =
    useState(false);
  const [passwordResetLoading, setPasswordResetLoading] = useState(false);
  const [savingFallback, setSavingFallback] = useState(false);
  const [emailForm, setEmailForm] = useState({ email: "", code: "" });
  const [passwordForm, setPasswordForm] = useState({
    oldPassword: "",
    password: "",
    repassword: "",
  });
  const [passkeyEnabled, setPasskeyEnabled] = useState(false);
  const [passkeys, setPasskeys] = useState<PasskeyCredentialInfo[]>([]);

  const refreshPasskeys = async () => {
    if (!auth) {
      setPasskeyEnabled(false);
      setPasskeys([]);
      return;
    }

    const resp = await listPasskeys();
    if (resp.status) {
      setPasskeyEnabled(resp.enabled);
      setPasskeys(resp.credentials ?? []);
    } else {
      withNotify(t, resp);
    }
  };
  useEffectAsync(refreshPasskeys, [auth, location.key]);

  const updateFallbackPreference = async (checked: boolean) => {
    const previous = allowSubscriptionQuotaFallback;
    dispatch(setAllowSubscriptionQuotaFallback(checked));
    setSavingFallback(true);

    const res = await updateSubscriptionQuotaFallback(checked);
    setSavingFallback(false);

    if (res.status) {
      dispatch(
        setAllowSubscriptionQuotaFallback(
          res.allow_subscription_quota_fallback ?? checked,
        ),
      );
      toast.success(t("buy.subscription-fallback-save-success"));
      return;
    }

    dispatch(setAllowSubscriptionQuotaFallback(previous));
    toast.error(t("buy.subscription-fallback-save-failed"), {
      description: res.error,
    });
  };

  async function sendEmailChangeCode() {
    const email = emailForm.email.trim();
    if (!isEmailValid(email)) {
      toast.error(t("error"), { description: t("auth.invalid-email") });
      return;
    }

    await sendCode(t, email, true);
  }

  async function requestPasswordReset() {
    if (passwordResetLoading) return;

    const email = info.email.trim();
    if (!isEmailValid(email)) {
      toast.error(t("error"), { description: t("account.email-not-bound") });
      return;
    }

    setPasswordResetLoading(true);
    try {
      const resp = await requestPasswordResetEmail(email);
      if (!resp.status) {
        toast.error(t("auth.reset-link-failed"), {
          description: t("auth.reset-link-failed-prompt", {
            reason: localizeError(t, resp.error),
          }),
        });
        return;
      }

      setPasswordDialogOpen(false);
      setPasswordResetDialogOpen(true);
    } finally {
      setPasswordResetLoading(false);
    }
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
    const oldPassword = passwordForm.oldPassword.trim();
    const password = passwordForm.password.trim();
    const repassword = passwordForm.repassword.trim();

    if (!isTextInRange(oldPassword, 6, 36)) {
      toast.error(t("error"), {
        description: t("account.old-password-invalid"),
      });
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

    const resp = await updateAccountPassword({
      old_password: oldPassword,
      password,
    });
    withNotify(t, resp, true, t("account.password-updated"));

    if (resp.status) {
      if (resp.token) {
        validateToken(dispatch, resp.token);
      }
      setPasswordDialogOpen(false);
      setPasswordForm({ oldPassword: "", password: "", repassword: "" });
    }
  }

  async function bindPasskey() {
    if (!window.PublicKeyCredential || !navigator.credentials?.create) {
      toast.error(t("error"), {
        description: t("account.passkey-unsupported"),
      });
      return;
    }

    try {
      const resp = await createPasskeyRegistrationOptions();
      if (!resp.status || !resp.data) {
        withNotify(t, resp);
        return;
      }

      const options = resp.data.publicKey;
      const authenticatorSelection = {
        ...options.authenticatorSelection,
      } as AuthenticatorSelectionCriteria;

      if (!authenticatorSelection.authenticatorAttachment) {
        delete authenticatorSelection.authenticatorAttachment;
      }

      const credential = (await navigator.credentials.create({
        publicKey: {
          ...options,
          challenge: base64urlToBuffer(options.challenge),
          user: {
            ...options.user,
            id: base64urlToBuffer(options.user.id),
          },
          excludeCredentials: options.excludeCredentials.map((item) => ({
            type: item.type,
            id: base64urlToBuffer(item.id),
          })),
          authenticatorSelection,
        },
      })) as PublicKeyCredential | null;

      if (!credential) {
        toast.warning(t("error"), {
          description: t("account.passkey-cancelled"),
        });
        return;
      }

      const response = credential.response as AuthenticatorAttestationResponse;
      const registerResp = await registerPasskey({
        name: t("account.passkey-default-name"),
        id: credential.id,
        raw_id: bufferToBase64url(credential.rawId),
        type: credential.type,
        client_data_json: bufferToBase64url(response.clientDataJSON),
        attestation_object: bufferToBase64url(response.attestationObject),
        transports: response.getTransports?.() ?? [],
      });

      withNotify(t, registerResp, true, t("account.passkey-bound"));
      if (registerResp.status) {
        await refreshPasskeys();
      }
    } catch (err) {
      console.debug(err);
      if (err instanceof DOMException && err.name === "NotAllowedError") {
        toast.warning(t("error"), {
          description: t("account.passkey-cancelled"),
        });
        return;
      }

      toast.error(t("error"), {
        description: getErrorMessage(err),
      });
    }
  }

  async function removePasskey(id: number) {
    const resp = await deletePasskey(id);
    withNotify(t, resp, true, t("account.passkey-removed"));
    if (resp.status) {
      await refreshPasskeys();
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
                          type="email"
                          placeholder={t("account.new-email")}
                          value={emailForm.email}
                          autoComplete="email"
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
                          <button
                            type="button"
                            className="ml-1 text-xs underline-offset-4 transition-colors hover:text-foreground hover:underline disabled:cursor-not-allowed disabled:opacity-60"
                            onClick={requestPasswordReset}
                            disabled={passwordResetLoading}
                          >
                            {t("account.forgot-password")}
                          </button>
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4">
                        <Input
                          type="password"
                          placeholder={t("account.old-password")}
                          value={passwordForm.oldPassword}
                          autoComplete="current-password"
                          onChange={(e) =>
                            setPasswordForm((prev) => ({
                              ...prev,
                              oldPassword: e.target.value,
                            }))
                          }
                        />
                        <Input
                          type="password"
                          placeholder={t("account.new-password")}
                          value={passwordForm.password}
                          autoComplete="new-password"
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
                          autoComplete="new-password"
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

                  <Dialog
                    open={passwordResetDialogOpen}
                    onOpenChange={setPasswordResetDialogOpen}
                  >
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>
                          {t("account.password-reset-sent")}
                        </DialogTitle>
                        <DialogDescription>
                          {t("account.password-reset-sent-description")}
                        </DialogDescription>
                      </DialogHeader>
                      <DialogFooter>
                        <DialogAction
                          onClick={() => setPasswordResetDialogOpen(false)}
                        >
                          {t("i-know")}
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

              <motion.div
                className="flex flex-wrap gap-2"
                variants={contentVariants}
              >
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
        {auth && subscription && (
          <motion.div variants={cardVariants}>
            <AccountCard
              title={"account.subscription-management"}
              description={"account.subscription-management-description"}
              icon={<Coins />}
            >
              <motion.div
                className="flex flex-col gap-3 rounded-lg border bg-muted/10 p-4 md:flex-row md:items-center md:justify-between"
                variants={contentVariants}
              >
                <div className="flex min-w-0 items-start gap-3">
                  <Coins className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground">
                      {t("buy.subscription-fallback-title")}
                    </p>
                    <p className="mt-1 text-xs leading-5 text-muted-foreground">
                      {t("buy.subscription-fallback-desc")}
                    </p>
                  </div>
                </div>
                <Switch
                  checked={allowSubscriptionQuotaFallback}
                  disabled={savingFallback}
                  onCheckedChange={updateFallbackPreference}
                  aria-label={t("buy.subscription-fallback-title")}
                  className="shrink-0"
                />
              </motion.div>
            </AccountCard>
          </motion.div>
        )}
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
                  <p className={`text-common text-sm font-bold`}>
                    DeepTrain SSO
                  </p>
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
            title={"account.security"}
            description={t("account.security-description")}
            icon={<Fingerprint />}
          >
            <motion.div className="space-y-3" variants={contentVariants}>
              <div className="flex flex-col gap-3 rounded-lg border bg-muted/10 p-4 md:flex-row md:items-center md:justify-between">
                <div className="flex min-w-0 items-start gap-3">
                  <Fingerprint className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium">
                      {t("account.passkey")}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {passkeyEnabled
                        ? t("account.passkey-description")
                        : t("account.passkey-disabled")}
                    </p>
                  </div>
                </div>
                {auth ? (
                  <Button
                    className="shrink-0"
                    size="default-sm"
                    loading
                    disabled={!passkeyEnabled}
                    onClick={bindPasskey}
                  >
                    <Fingerprint className="mr-1.5 h-3.5 w-3.5" />
                    {t("account.bind-passkey")}
                  </Button>
                ) : (
                  <Button
                    className="shrink-0"
                    size="default-sm"
                    onClick={goAuth}
                  >
                    <HandIcon className="mr-1.5 h-3.5 w-3.5" />
                    {t("login")}
                  </Button>
                )}
              </div>
              {auth && passkeys.length > 0 && (
                <div className="space-y-2">
                  {passkeys.map((item) => (
                    <div
                      key={item.id}
                      className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium">
                          {item.name || t("account.passkey")}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {t("account.passkey-created-at", {
                            time: item.created_at || "-",
                          })}
                        </p>
                      </div>
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button
                            variant="light-destructive"
                            size="icon-sm"
                            className="shrink-0"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>
                              {t("account.remove-passkey")}
                            </AlertDialogTitle>
                            <AlertDialogDescription>
                              {t("account.remove-passkey-description")}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => removePasskey(item.id)}
                            >
                              {t("confirm")}
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  ))}
                </div>
              )}
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
