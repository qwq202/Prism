import { useTranslation } from "react-i18next";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import Paragraph, {
  ParagraphDescription,
  ParagraphFooter,
  ParagraphItem,
  ParagraphSpace,
} from "@/components/Paragraph.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { Label } from "@/components/ui/label.tsx";
import { Input } from "@/components/ui/input.tsx";
import { useMemo, useReducer, useState } from "react";
import { formReducer } from "@/utils/form.ts";
import { NumberInput } from "@/components/ui/number-input.tsx";
import {
  AuthenticationState,
  CommonState,
  commonWhiteList,
  GeneralState,
  getConfig,
  initialSystemState,
  MailState,
  SearchState,
  TaskState,
  setConfig,
  SiteState,
  SystemProps,
  testStorageConfig,
  testWebSearching,
  updateRootPassword,
} from "@/admin/api/system.ts";
import { useEffectAsync } from "@/utils/hook.ts";
import { withNotify } from "@/api/common.ts";
import { doVerify } from "@/api/auth.ts";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTrigger,
} from "@/components/ui/dialog.tsx";
import { DialogTitle } from "@radix-ui/react-dialog";
import Require from "@/components/Require.tsx";
import { Loader2, PencilLine, RotateCw, Save, Settings2 } from "lucide-react";
import { FlexibleTextarea, Textarea } from "@/components/ui/textarea.tsx";
import Tips from "@/components/Tips.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import { Switch } from "@/components/ui/switch.tsx";
import { MultiCombobox } from "@/components/ui/multi-combobox.tsx";
import { useAllModels, useChannelModels } from "@/admin/hook.tsx";
import { useDispatch, useSelector } from "react-redux";
import { selectSupportModels } from "@/store/chat.ts";
import { JSONEditorProvider } from "@/components/EditorProvider.tsx";
import { Combobox } from "@/components/ui/combo-box.tsx";
import { validateToken } from "@/store/auth.ts";

type FormAction = {
  type: string;
  payload?: unknown;
  value?: unknown;
};

type CompProps<T> = {
  data: T;
  form: SystemProps;
  dispatch: (action: FormAction) => void;
  onChange: (doToast?: boolean) => Promise<void>;
};

const systemTimeZoneSuggestions = [
  "Asia/Shanghai",
  "Asia/Hong_Kong",
  "Asia/Taipei",
  "Asia/Tokyo",
  "Asia/Seoul",
  "UTC",
  "Europe/London",
  "Europe/Paris",
  "America/New_York",
  "America/Los_Angeles",
];

function RootDialog() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const [open, setOpen] = useState<boolean>(false);
  const [password, setPassword] = useState<string>("");
  const [repeat, setRepeat] = useState<string>("");

  const onPost = async () => {
    const res = await updateRootPassword(password);
    withNotify(t, res, true);
    if (res.status) {
      setPassword("");
      setRepeat("");
      setOpen(false);
      if (res.token) {
        validateToken(dispatch, res.token);
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant={`outline`} size={`sm`}>
          {t("admin.system.updateRoot")}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("admin.system.updateRoot")}</DialogTitle>
          <DialogDescription>
            <div className={`mb-4 select-none`}>
              {t("admin.system.updateRootTip")}
            </div>
            <Input
              className={`mb-2`}
              type={`password`}
              placeholder={t("admin.system.updateRootPlaceholder")}
              value={password}
              autoComplete="new-password"
              onChange={(e) => setPassword(e.target.value)}
            />
            <Input
              type={`password`}
              placeholder={t("admin.system.updateRootRepeatPlaceholder")}
              value={repeat}
              autoComplete="new-password"
              onChange={(e) => setRepeat(e.target.value)}
            />
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant={`outline`}
            onClick={() => {
              setPassword("");
              setRepeat("");
              setOpen(false);
            }}
          >
            {t("admin.cancel")}
          </Button>
          <Button
            variant={`default`}
            loading={true}
            onClick={onPost}
            disabled={
              password.trim().length === 0 || password.trim() !== repeat.trim()
            }
          >
            {t("admin.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function General({ data, dispatch, onChange }: CompProps<GeneralState>) {
  const { t } = useTranslation();

  return (
    <Paragraph
      title={t("admin.system.general")}
      configParagraph={true}
      isCollapsed={true}
    >
      <ParagraphItem>
        <Label>{t("admin.system.title")}</Label>
        <Input
          value={data.title}
          onChange={(e) =>
            dispatch({
              type: "update:general.title",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.titleTip")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.docs")}</Label>
        <Input
          value={data.docs}
          onChange={(e) =>
            dispatch({
              type: "update:general.docs",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.docsTip")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.logo")}</Label>
        <Input
          value={data.logo}
          onChange={(e) =>
            dispatch({
              type: "update:general.logo",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.logoTip", {
            logo: `${window.location.protocol}//${window.location.host}/favicon.svg`,
          })}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.backend")}</Label>
        <Input
          value={data.backend}
          onChange={(e) =>
            dispatch({
              type: "update:general.backend",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.backendPlaceholder")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          {t("admin.system.timezone")}
          <Tips
            className={`inline-block`}
            content={t("admin.system.timezoneTip")}
          />
        </Label>
        <Input
          list="system-timezone-options"
          value={data.timezone || "Asia/Shanghai"}
          onChange={(e) =>
            dispatch({
              type: "update:general.timezone",
              value: e.target.value,
            })
          }
          placeholder="Asia/Shanghai"
        />
        <datalist id="system-timezone-options">
          {systemTimeZoneSuggestions.map((timezone) => (
            <option key={timezone} value={timezone} />
          ))}
        </datalist>
      </ParagraphItem>
      <ParagraphDescription border>
        {t("admin.system.backendTip", {
          backend: `${window.location.protocol}//${window.location.host}/api`,
        })}
      </ParagraphDescription>
      <ParagraphItem>
        <Label>PWA Manifest</Label>
        <JSONEditorProvider
          value={data.pwa_manifest ?? ""}
          onChange={(value) =>
            dispatch({ type: "update:general.pwa_manifest", value })
          }
        >
          <Button variant={`outline`}>
            <PencilLine className={`h-4 w-4 mr-1`} />
            {t("edit")}
          </Button>
        </JSONEditorProvider>
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          {t("admin.system.debugMode")}
          <Tips
            className={`inline-block`}
            content={t("admin.system.debugModeTip")}
          />
        </Label>
        <Switch
          checked={data.debug_mode}
          onCheckedChange={(value) => {
            dispatch({ type: "update:general.debug_mode", value });
          }}
        />
      </ParagraphItem>
      <ParagraphSpace />
      <ParagraphFooter>
        <div className={`grow`} />
        <RootDialog />
        <Button
          size={`sm`}
          loading={true}
          onClick={async () => await onChange()}
        >
          {t("admin.system.save")}
        </Button>
      </ParagraphFooter>
    </Paragraph>
  );
}

function Mail({ data, dispatch, onChange }: CompProps<MailState>) {
  const { t } = useTranslation();
  const [email, setEmail] = useState<string>("");

  const [mailDialog, setMailDialog] = useState<boolean>(false);

  const valid = useMemo((): boolean => {
    return (
      data.host.length > 0 &&
      data.port > 0 &&
      data.port < 65535 &&
      data.username.length > 0 &&
      data.password.length > 0 &&
      data.from.length > 0
    );
  }, [data]);

  const onTest = async () => {
    if (!email.trim()) return;
    await onChange(false);
    const res = await doVerify(email);
    withNotify(t, res, true);

    if (res.status) setMailDialog(false);
  };

  const white_list = useMemo(() => {
    const raw = data.white_list.custom
      .split(",")
      .map((item) => item.trim())
      .filter((item) => item.length > 0);

    return [...commonWhiteList, ...raw];
  }, [data]);

  return (
    <Paragraph
      title={t("admin.system.mail")}
      configParagraph={true}
      isCollapsed={true}
    >
      {!valid && (
        <ParagraphDescription border={true}>
          {t("admin.system.mailConfNotValid")}
        </ParagraphDescription>
      )}
      <ParagraphItem>
        <Label>
          <Require /> {t("admin.system.mailHost")}
        </Label>
        <Input
          value={data.host}
          onChange={(e) =>
            dispatch({
              type: "update:mail.host",
              value: e.target.value,
            })
          }
          placeholder={`smtp.qcloudmail.com`}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          <Require /> {t("admin.system.mailProtocol")}
        </Label>
        <Select
          value={data.protocol ? "true" : "false"}
          onValueChange={(value: string) => {
            dispatch({
              type: "update:mail.protocol",
              value: value === "true",
            });
          }}
        >
          <SelectTrigger className={`select`}>
            <SelectValue
              placeholder={
                data.protocol
                  ? t("admin.system.mailProtocolTLS")
                  : t("admin.system.mailProtocolSSL")
              }
            />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="true">TLS</SelectItem>
            <SelectItem value="false">SSL</SelectItem>
          </SelectContent>
        </Select>
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          <Require /> {t("admin.system.mailPort")}
        </Label>
        <NumberInput
          value={data.port}
          onValueChange={(value) =>
            dispatch({ type: "update:mail.port", value })
          }
          placeholder={`465`}
          min={0}
          max={65535}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          <Require /> {t("admin.system.mailUser")}
        </Label>
        <Input
          value={data.username}
          onChange={(e) =>
            dispatch({
              type: "update:mail.username",
              value: e.target.value,
            })
          }
          className={cn("transition-all duration-300")}
          placeholder={t("admin.system.mailUser")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          <Require /> {t("admin.system.mailPass")}
        </Label>
        <Input
          value={data.password}
          onChange={(e) =>
            dispatch({
              type: "update:mail.password",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.mailPass")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          <Require /> {t("admin.system.mailFrom")}
        </Label>
        <Input
          value={data.from}
          onChange={(e) =>
            dispatch({
              type: "update:mail.from",
              value: e.target.value,
            })
          }
          placeholder={`${t("admin.system.mailFrom")} <${data.username}@${
            location.hostname
          }>`}
          className={cn("transition-all duration-300")}
        />
      </ParagraphItem>
      <ParagraphSpace />
      <ParagraphItem>
        <Label>{t("admin.system.mailEnableWhitelist")}</Label>
        <Switch
          checked={data.white_list.enabled}
          onCheckedChange={(value) => {
            dispatch({
              type: "update:mail.white_list.enabled",
              value,
            });
          }}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.mailWhitelist")}</Label>
        <MultiCombobox
          value={data.white_list.white_list}
          list={white_list}
          disabled={!data.white_list.enabled}
          onChange={(value) => {
            dispatch({
              type: "update:mail.white_list.white_list",
              value,
            });
          }}
          placeholder={t("admin.system.mailWhitelistSelected", {
            length: data.white_list.white_list.length,
          })}
          searchPlaceholder={t("admin.system.mailWhitelistSearchPlaceholder")}
        />
      </ParagraphItem>
      <Input
        className={`mb-2`}
        value={data.white_list.custom}
        onChange={(e) =>
          dispatch({
            type: "update:mail.white_list.custom",
            value: e.target.value,
          })
        }
        disabled={!data.white_list.enabled}
        placeholder={t("admin.system.customWhitelistPlaceholder")}
      />
      <ParagraphFooter>
        <div className={`grow`} />
        <Dialog open={mailDialog} onOpenChange={setMailDialog}>
          <DialogTrigger asChild>
            <Button variant={`outline`} size={`sm`}>
              {t("admin.system.test")}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("admin.system.test")}</DialogTitle>
              <DialogDescription className={`pt-2`}>
                <Input
                  placeholder={t("auth.email-placeholder")}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button
                variant={`outline`}
                onClick={() => {
                  setEmail("");
                  setMailDialog(false);
                }}
              >
                {t("admin.cancel")}
              </Button>
              <Button variant={`default`} loading={true} onClick={onTest}>
                {t("admin.confirm")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <Button
          size={`sm`}
          loading={true}
          onClick={async () => await onChange()}
        >
          {t("admin.system.save")}
        </Button>
      </ParagraphFooter>
    </Paragraph>
  );
}

function Site({ data, dispatch, onChange }: CompProps<SiteState>) {
  const { t } = useTranslation();

  return (
    <Paragraph
      title={t("admin.system.site")}
      configParagraph={true}
      isCollapsed={true}
    >
      <ParagraphItem>
        <Label>
          {t("admin.system.closeRegistration")}
          <Tips
            className={`inline-block`}
            content={t("admin.system.closeRegistrationTip")}
          />
        </Label>
        <Switch
          checked={data.close_register}
          onCheckedChange={(value) => {
            dispatch({ type: "update:site.close_register", value });
          }}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.quota")}
          <Tips content={t("admin.system.quotaTip")} />
        </Label>
        <NumberInput
          value={data.quota}
          onValueChange={(value) =>
            dispatch({ type: "update:site.quota", value })
          }
          placeholder={`5`}
          min={0}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.buyLink")}</Label>
        <Input
          value={data.buy_link}
          onChange={(e) =>
            dispatch({
              type: "update:site.buy_link",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.buyLinkPlaceholder")}
        />
      </ParagraphItem>
      <ParagraphItem rowLayout={true}>
        <Label>{t("admin.system.announcement")}</Label>
        <FlexibleTextarea
          value={data.announcement}
          rows={12}
          onChange={(e) =>
            dispatch({
              type: "update:site.announcement",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.announcementPlaceholder")}
        />
      </ParagraphItem>
      <ParagraphItem rowLayout={true}>
        <Label>{t("admin.system.contact")}</Label>
        <FlexibleTextarea
          value={data.contact}
          rows={6}
          onChange={(e) =>
            dispatch({
              type: "update:site.contact",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.contactPlaceholder")}
        />
      </ParagraphItem>
      <ParagraphSpace />
      <ParagraphItem rowLayout={true}>
        <Label>{t("admin.system.footer")}</Label>
        <FlexibleTextarea
          rows={6}
          value={data.footer}
          onChange={(e) =>
            dispatch({
              type: "update:site.footer",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.footerPlaceholder")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.authFooter")}</Label>
        <Switch
          checked={data.auth_footer}
          onCheckedChange={(value) => {
            dispatch({ type: "update:site.auth_footer", value });
          }}
        />
      </ParagraphItem>
      <ParagraphFooter>
        <div className={`grow`} />
        <Button
          size={`sm`}
          loading={true}
          onClick={async () => await onChange()}
        >
          {t("admin.system.save")}
        </Button>
      </ParagraphFooter>
    </Paragraph>
  );
}

function Authentication({
  data,
  dispatch,
  onChange,
}: CompProps<AuthenticationState>) {
  const { t } = useTranslation();
  const passkey = data.passkey;

  return (
    <Paragraph
      title={t("admin.system.authentication")}
      configParagraph={true}
      isCollapsed={true}
    >
      <ParagraphDescription border>
        {t("admin.system.passkeyTip")}
      </ParagraphDescription>
      <ParagraphItem>
        <Label>{t("admin.system.passkeyEnabled")}</Label>
        <Switch
          checked={passkey.enabled}
          onCheckedChange={(value) => {
            dispatch({ type: "update:auth.passkey.enabled", value });
          }}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.passkeyRpDisplayName")}</Label>
        <Input
          value={passkey.rp_display_name}
          onChange={(e) =>
            dispatch({
              type: "update:auth.passkey.rp_display_name",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.passkeyRpDisplayNamePlaceholder")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.passkeyRpId")}</Label>
        <Input
          value={passkey.rp_id}
          onChange={(e) =>
            dispatch({
              type: "update:auth.passkey.rp_id",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.passkeyRpIdPlaceholder")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.passkeyUserVerification")}
          <Tips content={t("admin.system.passkeyUserVerificationTip")} />
        </Label>
        <Select
          value={passkey.user_verification}
          onValueChange={(value) => {
            dispatch({
              type: "update:auth.passkey.user_verification",
              value,
            });
          }}
        >
          <SelectTrigger className={`select`}>
            <SelectValue
              placeholder={t(
                `admin.system.passkeyUserVerificationModes.${passkey.user_verification}`,
              )}
            />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="required">
              {t("admin.system.passkeyUserVerificationModes.required")}
            </SelectItem>
            <SelectItem value="preferred">
              {t("admin.system.passkeyUserVerificationModes.preferred")}
            </SelectItem>
            <SelectItem value="discouraged">
              {t("admin.system.passkeyUserVerificationModes.discouraged")}
            </SelectItem>
          </SelectContent>
        </Select>
      </ParagraphItem>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.passkeyAuthenticatorAttachment")}
          <Tips content={t("admin.system.passkeyAuthenticatorAttachmentTip")} />
        </Label>
        <Select
          value={passkey.authenticator_attachment}
          onValueChange={(value) => {
            dispatch({
              type: "update:auth.passkey.authenticator_attachment",
              value,
            });
          }}
        >
          <SelectTrigger className={`select`}>
            <SelectValue
              placeholder={t(
                `admin.system.passkeyAuthenticatorAttachments.${passkey.authenticator_attachment}`,
              )}
            />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="any">
              {t("admin.system.passkeyAuthenticatorAttachments.any")}
            </SelectItem>
            <SelectItem value="platform">
              {t("admin.system.passkeyAuthenticatorAttachments.platform")}
            </SelectItem>
            <SelectItem value="cross-platform">
              {t("admin.system.passkeyAuthenticatorAttachments.cross-platform")}
            </SelectItem>
          </SelectContent>
        </Select>
      </ParagraphItem>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.passkeyAllowInsecureOrigin")}
          <Tips content={t("admin.system.passkeyAllowInsecureOriginTip")} />
        </Label>
        <Switch
          checked={passkey.allow_insecure_origin}
          onCheckedChange={(value) => {
            dispatch({
              type: "update:auth.passkey.allow_insecure_origin",
              value,
            });
          }}
        />
      </ParagraphItem>
      <ParagraphItem rowLayout={true}>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.passkeyOrigins")}
          <Tips content={t("admin.system.passkeyOriginsTip")} />
        </Label>
        <Textarea
          rows={5}
          value={passkey.origins}
          onChange={(e) =>
            dispatch({
              type: "update:auth.passkey.origins",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.passkeyOriginsPlaceholder")}
        />
      </ParagraphItem>
      <ParagraphFooter>
        <div className={`grow`} />
        <Button
          size={`sm`}
          loading={true}
          onClick={async () => await onChange()}
        >
          {t("admin.system.save")}
        </Button>
      </ParagraphFooter>
    </Paragraph>
  );
}

function Common({ data, dispatch, onChange }: CompProps<CommonState>) {
  const { t } = useTranslation();

  const { channelModels } = useChannelModels();
  const supportModels = useSelector(selectSupportModels);

  return (
    <Paragraph
      title={t("admin.system.common")}
      configParagraph={true}
      isCollapsed={true}
    >
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.cache")}
          <Tips content={t("admin.system.cacheTip")} />
        </Label>
        <MultiCombobox
          value={data.cache}
          onChange={(value) => {
            dispatch({ type: "update:common.cache", value });
          }}
          list={channelModels}
          placeholder={t("admin.system.cachePlaceholder", {
            length: (data.cache ?? []).length,
          })}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          {t("admin.system.cacheExpired")}
          <Tips
            className={`inline-block`}
            content={t("admin.system.cacheExpiredTip")}
          />
        </Label>
        <NumberInput
          value={data.expire}
          onValueChange={(value) =>
            dispatch({ type: "update:common.expire", value })
          }
          min={0}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>
          {t("admin.system.cacheSize")}
          <Tips
            className={`inline-block`}
            content={t("admin.system.cacheSizeTip")}
          />
        </Label>
        <NumberInput
          value={data.size}
          onValueChange={(value) =>
            dispatch({ type: "update:common.size", value })
          }
          min={0}
        />
      </ParagraphItem>
      <ParagraphItem>
        <div className={`flex flex-row flex-wrap gap-2 ml-auto`}>
          <Button
            variant={`outline`}
            onClick={() => dispatch({ type: "update:common.cache", value: [] })}
          >
            <Settings2
              className={`inline-flex h-4 w-4 mr-2 translate-y-[1px]`}
            />
            {t("admin.system.cacheNone")}
          </Button>
          <Button
            variant={`outline`}
            onClick={() =>
              dispatch({
                type: "update:common.cache",
                value: supportModels
                  .filter((item) => item.free)
                  .map((item) => item.id),
              })
            }
          >
            <Settings2
              className={`inline-flex h-4 w-4 mr-2 translate-y-[1px]`}
            />
            {t("admin.system.cacheFree")}
          </Button>
          <Button
            variant={`outline`}
            onClick={() =>
              dispatch({ type: "update:common.cache", value: channelModels })
            }
          >
            <Settings2
              className={`inline-flex h-4 w-4 mr-2 translate-y-[1px]`}
            />
            {t("admin.system.cacheAll")}
          </Button>
        </div>
      </ParagraphItem>
      <ParagraphSpace />
      <ParagraphFooter>
        <div className={`grow`} />
        <Button
          size={`sm`}
          loading={true}
          onClick={async () => await onChange()}
        >
          {t("admin.system.save")}
        </Button>
      </ParagraphFooter>
    </Paragraph>
  );
}

function StorageSettings({
  form,
  data,
  dispatch,
  onChange,
}: CompProps<CommonState>) {
  const { t } = useTranslation();
  const [testing, setTesting] = useState<boolean>(false);

  return (
    <Paragraph
      title={t("admin.system.storage")}
      configParagraph={true}
      isCollapsed={true}
    >
      <ParagraphDescription border={true}>
        {t("admin.system.storageTip")}
      </ParagraphDescription>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.image_store")}
          <Tips content={t("admin.system.image_storeTip")} />
        </Label>
        <Switch
          checked={data.image_store}
          onCheckedChange={(value) => {
            dispatch({ type: "update:common.image_store", value });
          }}
        />
      </ParagraphItem>
      {data.image_store && form.general.backend.length === 0 && (
        <ParagraphDescription border={true}>
          {t("admin.system.image_storeNoBackend")}
        </ParagraphDescription>
      )}
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.orphanCleanupEnabled")}
          <Tips content={t("admin.system.orphanCleanupEnabledTip")} />
        </Label>
        <Switch
          checked={data.orphan_cleanup_enabled}
          onCheckedChange={(value) => {
            dispatch({ type: "update:common.orphan_cleanup_enabled", value });
          }}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.orphanCleanupInterval")}
          <Tips content={t("admin.system.orphanCleanupIntervalTip")} />
        </Label>
        <NumberInput
          min={5}
          max={10080}
          value={data.orphan_cleanup_interval}
          onValueChange={(value) => {
            dispatch({
              type: "update:common.orphan_cleanup_interval",
              value: Number(value) || 60,
            });
          }}
          placeholder={`60`}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.storageMode")}
          <Tips content={t("admin.system.storageModeTip")} />
        </Label>
        <Select
          value={data.storage_mode}
          onValueChange={(value: string) => {
            dispatch({
              type: "update:common.storage_mode",
              value: value === "s3" ? "s3" : value === "r2" ? "r2" : "local",
            });
          }}
        >
          <SelectTrigger className={`select`}>
            <SelectValue
              placeholder={t(`admin.system.storageMode_${data.storage_mode}`)}
            />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="local">
              {t("admin.system.storageMode_local")}
            </SelectItem>
            <SelectItem value="s3">
              {t("admin.system.storageMode_s3")}
            </SelectItem>
            <SelectItem value="r2">
              {t("admin.system.storageMode_r2")}
            </SelectItem>
          </SelectContent>
        </Select>
      </ParagraphItem>
      {data.storage_mode === "s3" && (
        <>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageEndpoint")}
              <Tips content={t("admin.system.storageEndpointTip")} />
            </Label>
            <Input
              value={data.s3.endpoint}
              onChange={(e) =>
                dispatch({
                  type: "update:common.s3.endpoint",
                  value: e.target.value,
                })
              }
              placeholder={`https://s3.amazonaws.com`}
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageRegion")}
              <Tips content={t("admin.system.storageRegionTip")} />
            </Label>
            <Input
              value={data.s3.region}
              onChange={(e) =>
                dispatch({
                  type: "update:common.s3.region",
                  value: e.target.value,
                })
              }
              placeholder={`us-east-1`}
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageBucket")}
              <Tips content={t("admin.system.storageBucketTip")} />
            </Label>
            <Input
              value={data.s3.bucket}
              onChange={(e) =>
                dispatch({
                  type: "update:common.s3.bucket",
                  value: e.target.value,
                })
              }
              placeholder={`your-bucket`}
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageAccessKey")}
              <Tips content={t("admin.system.storageAccessKeyTip")} />
            </Label>
            <Input
              value={data.s3.access_key}
              onChange={(e) =>
                dispatch({
                  type: "update:common.s3.access_key",
                  value: e.target.value,
                })
              }
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageSecretKey")}
              <Tips content={t("admin.system.storageSecretKeyTip")} />
            </Label>
            <Input
              value={data.s3.secret_key}
              onChange={(e) =>
                dispatch({
                  type: "update:common.s3.secret_key",
                  value: e.target.value,
                })
              }
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storagePublicBaseUrl")}
              <Tips content={t("admin.system.storagePublicBaseUrlTip")} />
            </Label>
            <Input
              value={data.s3.public_base_url}
              onChange={(e) =>
                dispatch({
                  type: "update:common.s3.public_base_url",
                  value: e.target.value,
                })
              }
              placeholder={`https://cdn.example.com`}
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageForcePathStyle")}
              <Tips content={t("admin.system.storageForcePathStyleTip")} />
            </Label>
            <Switch
              checked={data.s3.force_path_style}
              onCheckedChange={(value) => {
                dispatch({
                  type: "update:common.s3.force_path_style",
                  value,
                });
              }}
            />
          </ParagraphItem>
        </>
      )}
      {data.storage_mode === "r2" && (
        <>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageR2AccountId")}
              <Tips content={t("admin.system.storageR2AccountIdTip")} />
            </Label>
            <Input
              value={data.r2.account_id}
              onChange={(e) =>
                dispatch({
                  type: "update:common.r2.account_id",
                  value: e.target.value,
                })
              }
              placeholder={`0123456789abcdef0123456789abcdef`}
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageR2Jurisdiction")}
              <Tips content={t("admin.system.storageR2JurisdictionTip")} />
            </Label>
            <Input
              value={data.r2.jurisdiction}
              onChange={(e) =>
                dispatch({
                  type: "update:common.r2.jurisdiction",
                  value: e.target.value,
                })
              }
              placeholder={`eu`}
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageBucket")}
              <Tips content={t("admin.system.storageBucketTip")} />
            </Label>
            <Input
              value={data.r2.bucket}
              onChange={(e) =>
                dispatch({
                  type: "update:common.r2.bucket",
                  value: e.target.value,
                })
              }
              placeholder={`your-bucket`}
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageAccessKey")}
              <Tips content={t("admin.system.storageAccessKeyTip")} />
            </Label>
            <Input
              value={data.r2.access_key}
              onChange={(e) =>
                dispatch({
                  type: "update:common.r2.access_key",
                  value: e.target.value,
                })
              }
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storageSecretKey")}
              <Tips content={t("admin.system.storageSecretKeyTip")} />
            </Label>
            <Input
              value={data.r2.secret_key}
              onChange={(e) =>
                dispatch({
                  type: "update:common.r2.secret_key",
                  value: e.target.value,
                })
              }
            />
          </ParagraphItem>
          <ParagraphItem>
            <Label className={`flex flex-row items-center`}>
              {t("admin.system.storagePublicBaseUrl")}
              <Tips content={t("admin.system.storagePublicBaseUrlTip")} />
            </Label>
            <Input
              value={data.r2.public_base_url}
              onChange={(e) =>
                dispatch({
                  type: "update:common.r2.public_base_url",
                  value: e.target.value,
                })
              }
              placeholder={`https://pub-xxxxxxxx.r2.dev`}
            />
          </ParagraphItem>
        </>
      )}
      <ParagraphFooter>
        <div className={`grow`} />
        <Button
          variant={`outline`}
          size={`sm`}
          loading={true}
          disabled={testing}
          onClick={async () => {
            if (testing) return;
            setTesting(true);
            try {
              const res = await testStorageConfig(form);
              withNotify(
                t,
                res,
                true,
                res.message || t("admin.system.storageTestSuccess"),
              );
            } finally {
              setTesting(false);
            }
          }}
        >
          {t("admin.system.storageTest")}
        </Button>
        <Button
          size={`sm`}
          loading={true}
          onClick={async () => await onChange()}
        >
          {t("admin.system.save")}
        </Button>
      </ParagraphFooter>
    </Paragraph>
  );
}

function Search({ data, dispatch, onChange }: CompProps<SearchState>) {
  const { t } = useTranslation();

  const [search, setSearch] = useState<string>("");
  const [searchDialog, setSearchDialog] = useState<boolean>(false);
  const [searchResult, setSearchResult] = useState<string>("");
  const [searchLoading, setSearchLoading] = useState<boolean>(false);

  return (
    <Paragraph
      title={t("admin.system.search")}
      configParagraph={true}
      isCollapsed={true}
    >
      <ParagraphDescription border>
        {t("admin.system.searchTip")}
      </ParagraphDescription>
      <ParagraphItem>
        <Label>{t("admin.system.searchApiKey")}</Label>
        <Input
          value={data.api_key}
          onChange={(e) =>
            dispatch({
              type: "update:search.api_key",
              value: e.target.value,
            })
          }
          placeholder={t("admin.system.searchApiKeyPlaceholder")}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.searchTopic")}</Label>
        <Combobox
          value={data.topic}
          onChange={(value) => {
            dispatch({ type: "update:search.topic", value });
          }}
          list={["general", "news", "finance"]}
          listTranslated={`admin.system.searchTopics`}
          hideSearchBar
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.searchDepth")}</Label>
        <Combobox
          value={data.depth}
          onChange={(value) => {
            dispatch({ type: "update:search.depth", value });
          }}
          list={["basic", "advanced", "fast", "ultra-fast"]}
          listTranslated={`admin.system.searchDepthModes`}
          hideSearchBar
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.searchMaxResults")}</Label>
        <NumberInput
          value={data.max_results}
          onValueChange={(value) =>
            dispatch({ type: "update:search.max_results", value })
          }
          min={1}
          max={20}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.searchCrop")}
          <Tips content={t("admin.system.searchCropTip")} />
        </Label>
        <Switch
          checked={data.crop}
          onCheckedChange={(value) => {
            dispatch({ type: "update:search.crop", value });
          }}
        />
      </ParagraphItem>
      <ParagraphItem>
        <Label>{t("admin.system.searchCropLen")}</Label>
        <NumberInput
          value={data.crop_len}
          onValueChange={(value) =>
            dispatch({ type: "update:search.crop_len", value })
          }
          min={1}
          disabled={!data.crop}
        />
      </ParagraphItem>
      <ParagraphFooter>
        <div className={`grow`} />
        <Dialog open={searchDialog} onOpenChange={setSearchDialog}>
          <DialogTrigger asChild>
            <Button variant={`outline`} size={`sm`}>
              {t("admin.system.searchTest")}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("admin.system.searchTest")}</DialogTitle>
              <FlexibleTextarea
                placeholder={t("admin.system.searchTestTip")}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
              {(searchLoading || searchResult) && (
                <div
                  className={`mt-2 border rounded-md p-4 flex items-center justify-center flex-col`}
                >
                  {searchLoading ? (
                    <Loader2 className={`h-4 w-4 animate-spin`} />
                  ) : (
                    <>
                      <p className={`text-sm mb-1`}>Tavily Result</p>
                      <Textarea value={searchResult} rows={5} readOnly />
                    </>
                  )}
                </div>
              )}
            </DialogHeader>
            <DialogFooter>
              <Button
                variant={`outline`}
                onClick={() => {
                  setSearch("");
                  setSearchDialog(false);
                }}
              >
                {t("admin.cancel")}
              </Button>
              <Button
                variant={`default`}
                loading={true}
                onClick={async () => {
                  await onChange();

                  setSearchResult("");
                  setSearchLoading(true);
                  const res = await testWebSearching(search);
                  if (res.status) setSearchResult(res.result);

                  withNotify(t, res, true);
                  setSearchLoading(false);
                }}
              >
                {t("admin.confirm")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <Button
          size={`sm`}
          loading={true}
          onClick={async () => await onChange()}
        >
          {t("admin.system.save")}
        </Button>
      </ParagraphFooter>
    </Paragraph>
  );
}

function Task({ data, dispatch, onChange }: CompProps<TaskState>) {
  const { t } = useTranslation();
  const { allModels } = useAllModels();

  return (
    <Paragraph
      title={t("admin.system.task")}
      configParagraph={true}
      isCollapsed={true}
    >
      <ParagraphDescription border>
        {t("admin.system.taskTip")}
      </ParagraphDescription>
      <ParagraphItem>
        <Label className={`flex flex-row items-center`}>
          {t("admin.system.taskModel")}
          <Tips content={t("admin.system.taskModelTip")} />
        </Label>
        <div className={`flex flex-row gap-2 items-center`}>
          <Combobox
            value={data.model}
            onChange={(value) =>
              dispatch({ type: "update:task.model", value: value || "" })
            }
            list={allModels}
            placeholder={t("admin.system.taskModelPlaceholder")}
          />
          <Button
            variant={`outline`}
            onClick={() => dispatch({ type: "update:task.model", value: "" })}
          >
            {t("admin.system.taskModelClear")}
          </Button>
        </div>
      </ParagraphItem>
      <ParagraphFooter>
        <div className={`grow`} />
        <Button
          size={`sm`}
          loading={true}
          onClick={async () => await onChange()}
        >
          {t("admin.system.save")}
        </Button>
      </ParagraphFooter>
    </Paragraph>
  );
}

function System() {
  const { t } = useTranslation();
  const [data, setData] = useReducer(
    formReducer<SystemProps>(),
    initialSystemState,
  );

  const [loading, setLoading] = useState<boolean>(false);

  const doSaving = async (doToast?: boolean) => {
    const res = await setConfig(data);

    if (doToast !== false) withNotify(t, res, true);
  };

  const doRefresh = async () => {
    setLoading(true);
    const res = await getConfig();
    setLoading(false);
    withNotify(t, res);
    if (res.status) {
      setData({ type: "set", value: res.data });
    }
  };

  useEffectAsync(doRefresh, []);

  return (
    <div className={`system`}>
      <Card className={`admin-card system-card`}>
        <CardHeader className={`select-none`}>
          <CardTitle>{t("admin.settings")}</CardTitle>
        </CardHeader>
        <CardContent className={`flex flex-col gap-1`}>
          <div className={`system-actions flex flex-row`}>
            <div className={`grow`} />
            <Button
              size={`icon`}
              variant={`outline`}
              loading={true}
              className={`mr-2`}
              onClick={async () => await doRefresh()}
            >
              <RotateCw className={cn(loading && `animate-spin`, `h-4 w-4`)} />
            </Button>
            <Button
              size={`icon`}
              loading={true}
              onClick={async () => await doSaving()}
            >
              <Save className={`h-4 w-4`} />
            </Button>
          </div>
          <General
            form={data}
            data={data.general}
            dispatch={setData}
            onChange={doSaving}
          />
          <Site
            form={data}
            data={data.site}
            dispatch={setData}
            onChange={doSaving}
          />
          <Authentication
            form={data}
            data={data.auth}
            dispatch={setData}
            onChange={doSaving}
          />
          <Mail
            form={data}
            data={data.mail}
            dispatch={setData}
            onChange={doSaving}
          />
          <Search
            form={data}
            data={data.search}
            dispatch={setData}
            onChange={doSaving}
          />
          <Task
            form={data}
            data={data.task}
            dispatch={setData}
            onChange={doSaving}
          />
          <StorageSettings
            form={data}
            data={data.common}
            dispatch={setData}
            onChange={doSaving}
          />
          <Common
            form={data}
            data={data.common}
            dispatch={setData}
            onChange={doSaving}
          />
        </CardContent>
      </Card>
    </div>
  );
}

export default System;
