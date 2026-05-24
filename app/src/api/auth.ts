import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";
import { localizeError } from "@/utils/error.ts";
import { isEmailValid } from "@/utils/form.ts";
import { toast } from "sonner";
import type { TFunction } from "i18next";
import { getClientCache, setClientCache } from "@/utils/client-cache.ts";

const userInfoCacheKey = "user-info";

function hashCacheScope(value: string): string {
  let hash = 5381;
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 33) ^ value.charCodeAt(index);
  }
  return (hash >>> 0).toString(36);
}

function getUserInfoCacheKey(): string {
  const endpoint = String(axios.defaults.baseURL ?? "default");
  const token = String(axios.defaults.headers.common["Authorization"] ?? "");
  return `${userInfoCacheKey}:${endpoint}:${hashCacheScope(token)}`;
}

export type LoginForm = {
  username: string;
  password: string;
};

export type DeepLoginForm = {
  token: string;
};

export type LoginResponse = {
  status: boolean;
  error: string;
  token: string;
};

export type StateResponse = {
  status: boolean;
  user: string;
  admin: boolean;
};

export type RegisterForm = {
  username: string;
  password: string;
  repassword: string;
  email: string;
  code: string;
};

export type RegisterResponse = {
  status: boolean;
  error: string;
  token: string;
};

export type VerifyForm = {
  email: string;
};

export type VerifyResponse = {
  status: boolean;
  error: string;
};

export type ResetForm = {
  email: string;
  code: string;
  password: string;
  repassword: string;
};

export type ResetResponse = {
  status: boolean;
  error: string;
};

export type AccountEmailForm = {
  email: string;
  code: string;
};

export type AccountPasswordForm = {
  old_password: string;
  password: string;
};

export type PasskeyCredentialInfo = {
  id: number;
  name: string;
  created_at: string;
};

export type PasskeyCredentialDescriptor = {
  type: "public-key";
  id: string;
};

export type PasskeyRegistrationOptions = {
  publicKey: {
    challenge: string;
    rp: {
      name: string;
      id?: string;
    };
    user: {
      id: string;
      name: string;
      displayName: string;
    };
    pubKeyCredParams: PublicKeyCredentialParameters[];
    timeout: number;
    authenticatorSelection: {
      authenticatorAttachment?: AuthenticatorAttachment;
      userVerification: UserVerificationRequirement;
      residentKey?: ResidentKeyRequirement;
      requireResidentKey?: boolean;
    };
    attestation: AttestationConveyancePreference;
    excludeCredentials: PasskeyCredentialDescriptor[];
  };
};

export type PasskeyListResponse = VerifyResponse & {
  enabled: boolean;
  credentials: PasskeyCredentialInfo[];
};

export type PasskeyRegistrationOptionsResponse = VerifyResponse & {
  data?: PasskeyRegistrationOptions;
};

export type PasskeyRegistrationForm = {
  name?: string;
  id: string;
  raw_id: string;
  type: string;
  client_data_json: string;
  attestation_object: string;
  transports: string[];
};

export type PasskeyLoginOptionsForm = Record<string, never>;

export type PasskeyAuthenticationOptions = {
  publicKey: {
    challenge: string;
    timeout: number;
    rpId?: string;
    allowCredentials?: Array<
      PasskeyCredentialDescriptor & {
        transports?: AuthenticatorTransport[];
      }
    >;
    userVerification: UserVerificationRequirement;
  };
};

export type PasskeyLoginOptionsResponse = VerifyResponse & {
  data?: PasskeyAuthenticationOptions;
};

export type PasskeyLoginForm = {
  id: string;
  raw_id: string;
  type: string;
  authenticator_data: string;
  client_data_json: string;
  signature: string;
  user_handle?: string;
};

export type UserInfo = {
  id: number;
  register_days: number;
  used_quota: number;
  plan_total_month: number;
  email: string;
};

export type UserInfoResponse = {
  status: boolean;
  error: string;
  data: UserInfo;
};

export async function doLogin(
  data: DeepLoginForm | LoginForm,
): Promise<LoginResponse> {
  const response = await axios.post("/login", data);
  return response.data as LoginResponse;
}

export async function createPasskeyLoginOptions(
  data: PasskeyLoginOptionsForm = {},
): Promise<PasskeyLoginOptionsResponse> {
  try {
    const response = await axios.post("/login/passkey/options", data);
    return response.data as PasskeyLoginOptionsResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function verifyPasskeyLogin(
  data: PasskeyLoginForm,
): Promise<LoginResponse> {
  try {
    const response = await axios.post("/login/passkey/verify", data);
    return response.data as LoginResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
      token: "",
    };
  }
}

export async function doState(): Promise<StateResponse> {
  const response = await axios.post("/state");
  return response.data as StateResponse;
}

export async function doRegister(
  data: RegisterForm,
): Promise<RegisterResponse> {
  try {
    const response = await axios.post("/register", data);
    return response.data as RegisterResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
      token: "",
    };
  }
}

export async function doVerify(
  email: string,
  checkout?: boolean,
): Promise<VerifyResponse> {
  try {
    const response = await axios.post("/verify", {
      email,
      checkout,
    } as VerifyForm);
    return response.data as VerifyResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function doReset(data: ResetForm): Promise<ResetResponse> {
  try {
    const response = await axios.post("/reset", data);
    return response.data as ResetResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function updateAccountEmail(
  data: AccountEmailForm,
): Promise<VerifyResponse> {
  try {
    const response = await axios.post("/account/email", data);
    return response.data as VerifyResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function updateAccountPassword(
  data: AccountPasswordForm,
): Promise<VerifyResponse> {
  try {
    const response = await axios.post("/account/password", data);
    return response.data as VerifyResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function listPasskeys(): Promise<PasskeyListResponse> {
  try {
    const response = await axios.get("/account/passkeys");
    return response.data as PasskeyListResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
      enabled: false,
      credentials: [],
    };
  }
}

export async function createPasskeyRegistrationOptions(): Promise<PasskeyRegistrationOptionsResponse> {
  try {
    const response = await axios.post("/account/passkeys/options");
    return response.data as PasskeyRegistrationOptionsResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function registerPasskey(
  data: PasskeyRegistrationForm,
): Promise<VerifyResponse> {
  try {
    const response = await axios.post("/account/passkeys/register", data);
    return response.data as VerifyResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function deletePasskey(id: number): Promise<VerifyResponse> {
  try {
    const response = await axios.delete(`/account/passkeys/${id}`);
    return response.data as VerifyResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function sendCode(
  t: TFunction,
  email: string,
  checkout?: boolean,
): Promise<boolean> {
  if (email.trim().length === 0 || !isEmailValid(email)) return false;

  const res = await doVerify(email, checkout);
  if (!res.status)
    toast.error(t("auth.send-code-failed"), {
      description: t("auth.send-code-failed-prompt", {
        reason: localizeError(t, res.error),
      }),
    });
  else
    toast.info(t("auth.send-code-success"), {
      description: t("auth.send-code-success-prompt"),
    });

  return res.status;
}

export const initialUserInfo: UserInfo = {
  id: 0,
  register_days: 0,
  used_quota: 0,
  plan_total_month: 0,
  email: "",
};

export async function getUserInfo(): Promise<UserInfoResponse> {
  try {
    const response = await axios.get("/userinfo");
    const data = response.data as UserInfoResponse;
    if (data.status) void setClientCache(getUserInfoCacheKey(), data.data);
    return data;
  } catch (e) {
    const cached = await getCachedUserInfo();
    return {
      status: Boolean(cached),
      error: getErrorMessage(e),
      data: cached ?? { ...initialUserInfo },
    };
  }
}

export async function getCachedUserInfo(): Promise<UserInfo | undefined> {
  return await getClientCache<UserInfo>(getUserInfoCacheKey());
}
