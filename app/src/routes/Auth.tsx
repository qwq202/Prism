import { useCallback, useEffect, useReducer } from "react";
import Loader from "@/components/Loader.tsx";
import "@/assets/pages/auth.less";
import { validateToken } from "@/store/auth.ts";
import { useDispatch } from "react-redux";
import router from "@/router.tsx";
import { useTranslation } from "react-i18next";
import { getQueryParam } from "@/utils/path.ts";
import { appLogo, appName, useDeeptrain } from "@/conf/env.ts";
import PrismLogo from "@/components/PrismLogo.tsx";
import { Card, CardContent } from "@/components/ui/card.tsx";
import { goAuth } from "@/utils/app.ts";
import { Label } from "@/components/ui/label.tsx";
import { Input } from "@/components/ui/input.tsx";
import Require, { LengthRangeRequired } from "@/components/Require.tsx";
import { Button } from "@/components/ui/button.tsx";
import { formReducer, isTextInRange } from "@/utils/form.ts";
import {
  createPasskeyLoginOptions,
  doLogin,
  LoginForm,
  verifyPasskeyLogin,
} from "@/api/auth.ts";
import { getErrorMessage, isEnter } from "@/utils/base.ts";
import { localizeError } from "@/utils/error.ts";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { toast } from "sonner";
import { Fingerprint } from "lucide-react";
import { useLocation } from "react-router-dom";

function getPostLoginRedirect(state: unknown): string {
  const from = (state as { from?: unknown } | null)?.from;
  if (
    typeof from !== "string" ||
    !from.startsWith("/") ||
    from.startsWith("//") ||
    from.startsWith("/login")
  ) {
    return "/";
  }

  return from;
}

const knownUnsafeRootPasswords = new Set(["chatnio123456", "coai123456"]);

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

function DeepAuth() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const token = getQueryParam("token").trim();

  useEffect(() => {
    if (!token.length) {
      toast.warning(t("invalid-token"), {
        description: t("invalid-token-prompt"),
        action: {
          label: t("try-again"),
          onClick: goAuth,
        },
      });

      setTimeout(goAuth, 2500);
      return;
    }

    doLogin({ token })
      .then((data) => {
        if (!data.status) {
          toast.error(t("login-failed"), {
            description: t("login-failed-prompt", {
              reason: localizeError(t, data.error),
            }),
            action: {
              label: t("try-again"),
              onClick: goAuth,
            },
          });
        } else
          validateToken(dispatch, data.token, async () => {
            toast.success(t("login-success"), {
              description: t("login-success-prompt"),
            });

            await router.navigate("/");
          });
      })
      .catch((err) => {
        console.debug(err);

        toast.error(t("server-error"), {
          description: `${t("server-error-prompt")}\n${err.message}`,
          action: {
            label: t("try-again"),
            onClick: goAuth,
          },
        });
      });
  }, [dispatch, t, token]);

  return (
    <div className={`auth`}>
      <Loader prompt={t("login")} />
    </div>
  );
}

function Login() {
  const { t } = useTranslation();
  const globalDispatch = useDispatch();
  const location = useLocation();
  const redirectPath = getPostLoginRedirect(location.state);
  const [form, dispatch] = useReducer(formReducer<LoginForm>(), {
    username: "",
    password: "",
  });

  useEffect(() => {
    sessionStorage.removeItem("username");
    sessionStorage.removeItem("password");
  }, []);

  const onSubmit = useCallback(async () => {
    if (
      !isTextInRange(form.username, 1, 255) ||
      !isTextInRange(form.password, 6, 36)
    )
      return;

    try {
      const resp = await doLogin(form);
      if (!resp.status) {
        toast.warning(t("login-failed"), {
          description: t("login-failed-prompt", {
            reason: localizeError(t, resp.error),
          }),
        });
        return;
      }

      toast.success(t("login-success"), {
        description: t("login-success-prompt"),
      });

      if (
        form.username.trim() === "root" &&
        knownUnsafeRootPasswords.has(form.password.trim())
      ) {
        toast.warning(t("admin.default-password"), {
          description: t("admin.default-password-prompt"),
          duration: 15000,
        });
      }

      validateToken(globalDispatch, resp.token, async () => {
        await router.navigate(redirectPath);
      });
    } catch (err) {
      console.debug(err);
      toast.error(t("server-error"), {
        description: `${t("server-error-prompt")}\n${getErrorMessage(err)}`,
      });
    }
  }, [form, globalDispatch, redirectPath, t]);

  const onPasskeyLogin = useCallback(async () => {
    if (!window.PublicKeyCredential || !navigator.credentials?.get) {
      toast.warning(t("login-failed"), {
        description: t("auth.passkey-unsupported"),
      });
      return;
    }

    try {
      const optionsResp = await createPasskeyLoginOptions();
      if (!optionsResp.status || !optionsResp.data) {
        toast.warning(t("login-failed"), {
          description: t("login-failed-prompt", {
            reason: localizeError(t, optionsResp.error),
          }),
        });
        return;
      }

      const options = optionsResp.data.publicKey;
      const publicKey: PublicKeyCredentialRequestOptions = {
        ...options,
        challenge: base64urlToBuffer(options.challenge),
        allowCredentials: options.allowCredentials?.map((item) => ({
          type: item.type,
          id: base64urlToBuffer(item.id),
          transports: item.transports,
        })),
      };
      if (!publicKey.allowCredentials?.length) {
        delete publicKey.allowCredentials;
      }

      const credential = (await navigator.credentials.get({
        publicKey,
      })) as PublicKeyCredential | null;

      if (!credential) {
        toast.warning(t("login-failed"), {
          description: t("auth.passkey-login-unavailable"),
        });
        return;
      }

      const response = credential.response as AuthenticatorAssertionResponse;
      const resp = await verifyPasskeyLogin({
        id: credential.id,
        raw_id: bufferToBase64url(credential.rawId),
        type: credential.type,
        authenticator_data: bufferToBase64url(response.authenticatorData),
        client_data_json: bufferToBase64url(response.clientDataJSON),
        signature: bufferToBase64url(response.signature),
        user_handle: response.userHandle
          ? bufferToBase64url(response.userHandle)
          : undefined,
      });

      if (!resp.status) {
        toast.warning(t("login-failed"), {
          description: t("login-failed-prompt", {
            reason: localizeError(t, resp.error),
          }),
        });
        return;
      }

      toast.success(t("login-success"), {
        description: t("login-success-prompt"),
      });

      validateToken(globalDispatch, resp.token, async () => {
        await router.navigate(redirectPath);
      });
    } catch (err) {
      console.debug(err);
      if (err instanceof DOMException && err.name === "NotAllowedError") {
        toast.warning(t("login-failed"), {
          description: t("auth.passkey-login-unavailable"),
        });
        return;
      }

      toast.error(t("server-error"), {
        description: `${t("server-error-prompt")}\n${getErrorMessage(err)}`,
      });
    }
  }, [globalDispatch, redirectPath, t]);

  useEffect(() => {
    // listen to enter key and auto submit
    const listener = async (e: KeyboardEvent) => {
      if (isEnter(e)) await onSubmit();
    };

    document.addEventListener("keydown", listener);
    return () => document.removeEventListener("keydown", listener);
  }, [onSubmit]);

  return (
    <ScrollArea className={`w-full h-full grid place-items-center`}>
      <div className={`auth-container`}>
        {appLogo === "/favicon.svg" ? (
          <PrismLogo className={`logo`} />
        ) : (
          <img className={`logo`} src={appLogo} alt="" />
        )}
        <div className={`title`}>
          {t("login")} {appName}
        </div>
        <Card className={`auth-card`}>
          <CardContent className={`pb-0`}>
            <div className={`auth-wrapper`}>
              <Label>
                <Require />
                {t("auth.username-or-email")}
                <LengthRangeRequired
                  content={form.username}
                  min={1}
                  max={255}
                  hideOnEmpty={true}
                />
              </Label>
              <Input
                placeholder={t("auth.username-or-email-placeholder")}
                value={form.username}
                onChange={(e) =>
                  dispatch({ type: "update:username", payload: e.target.value })
                }
              />

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
                onChange={(e) =>
                  dispatch({ type: "update:password", payload: e.target.value })
                }
              />

              <Button
                tapScale={0.975}
                classNameWrapper={`mt-2`}
                onClick={onSubmit}
                className={`w-full`}
                loading={true}
              >
                {t("login")}
              </Button>
              <Button
                tapScale={0.975}
                onClick={onPasskeyLogin}
                className={`w-full`}
                variant={`outline`}
                loading={true}
              >
                <Fingerprint className={`mr-2 h-4 w-4`} />
                {t("auth.passkey-login")}
              </Button>
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
            {t("auth.forgot-password")}
            <a className={`link`} onClick={() => router.navigate("/forgot")}>
              {t("auth.reset-password")}
            </a>
          </div>
        </div>
      </div>
    </ScrollArea>
  );
}

function Auth() {
  return useDeeptrain ? <DeepAuth /> : <Login />;
}

export default Auth;
