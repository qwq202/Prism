import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";
import { isEmailValid } from "@/utils/form.ts";
import { toast } from "sonner";
import type { TFunction } from "i18next";

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
  code: string;
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
      description: t("auth.send-code-failed-prompt", { reason: res.error }),
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
    return response.data as UserInfoResponse;
  } catch (e) {
    return {
      status: false,
      error: getErrorMessage(e),
      data: { ...initialUserInfo },
    };
  }
}
