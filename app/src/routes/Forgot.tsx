import { useTranslation } from "react-i18next";
import { useMemo, useReducer } from "react";
import { formReducer, isEmailValid, isTextInRange } from "@/utils/form.ts";
import { doReset, requestPasswordReset, ResetForm } from "@/api/auth.ts";
import router from "@/router.tsx";
import { Card, CardContent } from "@/components/ui/card.tsx";
import { Label } from "@/components/ui/label.tsx";
import Require, {
  EmailRequire,
  LengthRangeRequired,
  SameRequired,
} from "@/components/Require.tsx";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Input } from "@/components/ui/input.tsx";
import { Button } from "@/components/ui/button.tsx";
import { appLogo } from "@/conf/env.ts";
import { useDispatch, useSelector } from "react-redux";
import { infoMailSelector } from "@/store/info.ts";
import { AlertCircle } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { toast } from "sonner";
import { localizeError } from "@/utils/error.ts";
import { getQueryParam } from "@/utils/path.ts";
import { logout } from "@/store/auth.ts";

function Forgot() {
  const { t } = useTranslation();
  const dispatchAuth = useDispatch();
  const enabled = useSelector(infoMailSelector);
  const initialEmail = useMemo(() => getQueryParam("email"), []);
  const initialToken = useMemo(
    () => getQueryParam("token") || getQueryParam("code"),
    [],
  );
  const hasResetToken = initialToken.length > 0;

  const [form, dispatch] = useReducer(formReducer<ResetForm>(), {
    email: initialEmail,
    code: initialToken,
    password: "",
    repassword: "",
  });

  const onRequestResetLink = async () => {
    if (!isEmailValid(form.email)) {
      toast.error(t("error"), { description: t("auth.invalid-email") });
      return;
    }

    const res = await requestPasswordReset(form.email.trim());
    if (!res.status) {
      toast.error(t("auth.reset-link-failed"), {
        description: t("auth.reset-link-failed-prompt", {
          reason: localizeError(t, res.error),
        }),
      });
      return;
    }

    toast.info(t("auth.reset-link-sent"), {
      description: t("auth.reset-link-sent-prompt"),
    });
  };

  const onSubmit = async () => {
    if (
      !isEmailValid(form.email) ||
      !form.code.length ||
      !isTextInRange(form.password, 6, 36) ||
      form.password.trim() !== form.repassword.trim()
    )
      return;

    const res = await doReset(form);
    if (!res.status) {
      toast.error(t("error"), {
        description: localizeError(t, res.error),
      });
      return;
    }

    toast.info(t("auth.reset-success"), {
      description: t("auth.reset-success-prompt"),
    });

    sessionStorage.removeItem("username");
    sessionStorage.removeItem("password");
    dispatchAuth(logout());
    await router.navigate("/login");
  };

  return (
    <ScrollArea className={`w-full h-full grid place-items-center`}>
      <div className={`auth-container`}>
        <img className={`logo`} src={appLogo} alt="" />
        <div className={`title`}>{t("auth.reset-password")}</div>
        <Card className={`auth-card`}>
          <CardContent className={`pb-0`}>
            <div className={`auth-wrapper`}>
              {!enabled && !hasResetToken && (
                <Alert className={`p-4`}>
                  <AlertCircle className={`h-4 w-4`} />
                  <AlertDescription>{t("auth.disabled-mail")}</AlertDescription>
                </Alert>
              )}
              <Label>
                <Require />
                {t("auth.email")}
                <EmailRequire content={form.email} hideOnEmpty={true} />
              </Label>
              <Input
                type="email"
                placeholder={t("auth.email-placeholder")}
                value={form.email}
                autoComplete="email"
                disabled={hasResetToken}
                onChange={(e) =>
                  dispatch({
                    type: "update:email",
                    payload: e.target.value,
                  })
                }
              />

              {!hasResetToken ? (
                <Button
                  disabled={!enabled}
                  onClick={onRequestResetLink}
                  tapScale={0.975}
                  classNameWrapper={`mt-2`}
                  className={`w-full`}
                  loading={true}
                >
                  {t("auth.send-reset-link")}
                </Button>
              ) : (
                <>
                  <Alert className={`p-4`}>
                    <AlertCircle className={`h-4 w-4`} />
                    <AlertDescription>
                      {t("auth.reset-link-ready")}
                    </AlertDescription>
                  </Alert>

                  <Label>
                    <Require />
                    {t("auth.password")}
                    <LengthRangeRequired
                      content={form.password}
                      min={6}
                      max={36}
                      hideOnEmpty={true}
                    />
                  </Label>
                  <Input
                    placeholder={t("auth.password-placeholder")}
                    value={form.password}
                    type={"password"}
                    autoComplete="new-password"
                    onChange={(e) =>
                      dispatch({
                        type: "update:password",
                        payload: e.target.value,
                      })
                    }
                  />

                  <Label>
                    <Require />
                    {t("auth.check-password")}
                    <SameRequired
                      content={form.password}
                      compare={form.repassword}
                      hideOnEmpty={true}
                    />
                  </Label>
                  <Input
                    placeholder={t("auth.check-password-placeholder")}
                    value={form.repassword}
                    type={"password"}
                    autoComplete="new-password"
                    onChange={(e) =>
                      dispatch({
                        type: "update:repassword",
                        payload: e.target.value,
                      })
                    }
                  />

                  <Button
                    onClick={onSubmit}
                    tapScale={0.975}
                    classNameWrapper={`mt-2`}
                    className={`w-full`}
                    loading={true}
                  >
                    {t("reset")}
                  </Button>
                </>
              )}
            </div>
          </CardContent>
        </Card>
        <div className={`auth-card addition-wrapper`}>
          <div className={`row`}>
            {t("auth.no-account")}
            <a className={`link`} onClick={() => router.navigate("/register")}>
              {t("auth.register")}
            </a>
          </div>
          <div className={`row`}>
            {t("auth.have-account")}
            <a className={`link`} onClick={() => router.navigate("/login")}>
              {t("auth.login")}
            </a>
          </div>
        </div>
      </div>
    </ScrollArea>
  );
}

export default Forgot;
