import {
  ActiveUserChartResponse,
  BillingChartResponse,
  CommonResponse,
  ConversionFunnelResponse,
  ErrorChartResponse,
  InfoResponse,
  InvitationGenerateResponse,
  InvitationResponse,
  ModelChartResponse,
  RedeemBatchCodesResponse,
  RedeemBatchResponse,
  RedeemResponse,
  RegistrationChartResponse,
  RequestChartResponse,
  UserData,
  UserResponse,
  UserTypeChartResponse,
} from "@/admin/types.ts";
import axios, { type AxiosRequestConfig } from "axios";
import { getErrorMessage } from "@/utils/base.ts";
import {
  asArray,
  asNumber,
  asRecord,
  asString,
  asUserRecord,
  normalizeSubscriptionWindows,
} from "./normalize.ts";

export const initialAdminInfoState: InfoResponse = {
  subscription_count: 0,
  billing_today: 0,
  billing_month: 0,
  online_chats: 0,
  billing_yesterday: 0,
  billing_last_month: 0,
};

export type UserFilterProps = {
  plan: string;
  admin: string;
  ban: string;
  sort: string;
};

export const initialUserFilter: UserFilterProps = {
  plan: "all", // all/no/yes
  admin: "all", // all/no/yes
  ban: "all", // all/no/yes
  sort: "id-asc",
  // id-asc/id-desc
  // quota-asc/quota-desc
  // used-quota-asc/used-quota-desc
  // plan-desc/plan-asc
};

const adminAnalyticsNoCacheConfig: AxiosRequestConfig = {
  prismCache: false,
  headers: {
    "Cache-Control": "no-cache",
    Pragma: "no-cache",
  },
};

const initialUserTypeChartState: UserTypeChartResponse = {
  total: 0,
  normal: 0,
  api_paid: 0,
  basic_plan: 0,
  standard_plan: 0,
  pro_plan: 0,
};

function normalizeStringArray(value: unknown): string[] {
  return asArray<unknown>(value).map((item) => String(item ?? ""));
}

function normalizeNumberArray(value: unknown): number[] {
  return asArray<unknown>(value).map(asNumber);
}

function normalizeSeriesChart(value: unknown): RequestChartResponse {
  const data = asRecord(value);

  return {
    date: normalizeStringArray(data.date),
    value: normalizeNumberArray(data.value),
  };
}

function normalizeModelChart(value: unknown): ModelChartResponse {
  const data = asRecord(value);

  return {
    date: normalizeStringArray(data.date),
    value: asArray<unknown>(data.value)
      .map((item) => {
        const record = asRecord(item);
        const model = asString(record.model).trim();
        if (!model) return null;

        return {
          model,
          data: normalizeNumberArray(record.data),
        };
      })
      .filter(
        (item): item is ModelChartResponse["value"][number] => item !== null,
      ),
  };
}

function normalizeAdminInfo(value: unknown): InfoResponse {
  const data = asRecord(value);

  return {
    subscription_count: asNumber(data.subscription_count),
    billing_today: asNumber(data.billing_today),
    billing_month: asNumber(data.billing_month),
    online_chats: asNumber(data.online_chats),
    billing_yesterday: asNumber(data.billing_yesterday),
    billing_last_month: asNumber(data.billing_last_month),
  };
}

function normalizeUserTypeChart(value: unknown): UserTypeChartResponse {
  const data = asRecord(value);

  return {
    total: asNumber(data.total),
    normal: asNumber(data.normal),
    api_paid: asNumber(data.api_paid),
    basic_plan: asNumber(data.basic_plan),
    standard_plan: asNumber(data.standard_plan),
    pro_plan: asNumber(data.pro_plan),
  };
}

function normalizeFunnel(value: unknown): ConversionFunnelResponse {
  const data = asRecord(value);

  return {
    registered: asNumber(data.registered),
    ever_subscribed: asNumber(data.ever_subscribed),
    active_subscribed: asNumber(data.active_subscribed),
  };
}

function normalizeUser(value: unknown): UserData | null {
  const user = asUserRecord(value);
  if (!user) return null;
  const userData = user as Omit<UserData, "subscription_windows">;

  return {
    ...userData,
    subscription_windows: normalizeSubscriptionWindows(
      user.subscription_windows,
    ),
  };
}

function normalizeUserResponse(value: unknown): UserResponse {
  const data = asRecord(value);

  return {
    status: data.status === true,
    message: asString(data.message || data.error),
    data: asArray<unknown>(data.data)
      .map(normalizeUser)
      .filter((user): user is UserData => user !== null),
    total: asNumber(data.total),
  };
}

export async function getAdminInfo(): Promise<InfoResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/info",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeAdminInfo(response.data);
  } catch (e) {
    console.warn(e);
    return {
      ...initialAdminInfoState,
    };
  }
}

export async function getModelChart(): Promise<ModelChartResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/model",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeModelChart(response.data);
  } catch (e) {
    console.warn(e);
    return { date: [], value: [] };
  }
}

export async function getRequestChart(): Promise<RequestChartResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/request",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeSeriesChart(response.data);
  } catch (e) {
    console.warn(e);
    return { date: [], value: [] };
  }
}

export async function getBillingChart(): Promise<BillingChartResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/billing",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeSeriesChart(response.data);
  } catch (e) {
    console.warn(e);
    return { date: [], value: [] };
  }
}

export async function getErrorChart(): Promise<ErrorChartResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/error",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeSeriesChart(response.data);
  } catch (e) {
    console.warn(e);
    return { date: [], value: [] };
  }
}

export async function getUserTypeChart(): Promise<UserTypeChartResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/user",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeUserTypeChart(response.data);
  } catch (e) {
    console.warn(e);
    return { ...initialUserTypeChartState };
  }
}

export async function getInvitationList(
  page: number,
): Promise<InvitationResponse> {
  try {
    const response = await axios.get(`/admin/invitation/list?page=${page}`);
    const data = asRecord(response.data);
    return {
      status: data.status === true,
      message: asString(data.message || data.error),
      data: asArray<InvitationResponse["data"][number]>(data.data),
      total: asNumber(data.total),
    };
  } catch (e) {
    return {
      status: false,
      message: getErrorMessage(e),
      data: [],
      total: 0,
    };
  }
}

export async function deleteInvitation(code: string): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/invitation/delete", { code });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function generateInvitation(
  type: string,
  quota: number,
  number: number,
): Promise<InvitationGenerateResponse> {
  try {
    const response = await axios.post("/admin/invitation/generate", {
      type,
      quota,
      number,
    });
    return response.data as InvitationGenerateResponse;
  } catch (e) {
    return { status: false, data: [], message: getErrorMessage(e) };
  }
}

export async function getRedeemList(page: number): Promise<RedeemResponse> {
  try {
    const response = await axios.get(`/admin/redeem/list?page=${page}`);
    const data = asRecord(response.data);
    return {
      status: data.status === true,
      message: asString(data.message || data.error),
      data: asArray<RedeemResponse["data"][number]>(data.data),
      total: asNumber(data.total),
    };
  } catch (e) {
    console.warn(e);
    return { status: false, message: getErrorMessage(e), data: [], total: 0 };
  }
}

export async function deleteRedeem(code: string): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/redeem/delete", { code });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function generateRedeem(
  quota: number,
  number: number,
): Promise<InvitationGenerateResponse> {
  try {
    const response = await axios.post("/admin/redeem/generate", {
      quota,
      number,
    });
    return response.data as InvitationGenerateResponse;
  } catch (e) {
    return { status: false, data: [], message: getErrorMessage(e) };
  }
}

export async function getUserList(
  page: number,
  search: string,
  params: UserFilterProps,
): Promise<UserResponse> {
  try {
    const response = await axios.get(`/admin/user/list`, {
      params: {
        page,
        search,
        ...params,
      },
    });
    return normalizeUserResponse(response.data);
  } catch (e) {
    return {
      status: false,
      message: getErrorMessage(e),
      data: [],
      total: 0,
    };
  }
}

export async function createUserOperation(
  username: string,
  email: string,
  password: string,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/create", {
      username,
      email,
      password,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function updatePassword(
  id: number,
  password: string,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/password", {
      id,
      password,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function updateEmail(
  id: number,
  email: string,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/email", {
      id,
      email,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function quotaOperation(
  id: number,
  quota: number,
  override?: boolean,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/quota", {
      id,
      quota,
      override: override ?? false,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function subscriptionOperation(
  id: number,
  expired: string,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/subscription", {
      id,
      expired,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function banUserOperation(
  id: number,
  ban: boolean,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/ban", {
      id,
      ban,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function deleteUserOperation(id: number): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/delete", { id });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function setAdminOperation(
  id: number,
  admin: boolean,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/admin", {
      id,
      admin,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function subscriptionLevelOperation(
  id: number,
  level: number,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/level", { id, level });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export type ReleaseUsageType = "hour" | "week";

export async function releaseUsageOperation(
  id: number,
  type: ReleaseUsageType,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/release", { id, type });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function releaseAllUsageOperation(
  type: ReleaseUsageType,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/release", {
      all: true,
      type,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export type BatchUserAction = "ban" | "unban" | "add_quota";

export async function batchUserOperation(
  ids: number[],
  action: BatchUserAction,
  value?: number,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/user/batch", {
      ids,
      action,
      value,
    });
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function getRedeemBatchList(): Promise<RedeemBatchResponse> {
  try {
    const response = await axios.get("/admin/redeem/batch/list");
    const data = asRecord(response.data);
    return {
      status: data.status === true,
      message: asString(data.message || data.error),
      data: asArray<RedeemBatchResponse["data"][number]>(data.data),
    };
  } catch (e) {
    return { status: false, message: getErrorMessage(e), data: [] };
  }
}

export async function getRedeemBatchCodes(
  batchId: string,
): Promise<RedeemBatchCodesResponse> {
  try {
    const response = await axios.get(`/admin/redeem/batch/${batchId}`);
    const data = asRecord(response.data);
    return {
      status: data.status === true,
      message: asString(data.message || data.error),
      data: asArray<RedeemBatchCodesResponse["data"][number]>(data.data),
    };
  } catch (e) {
    return { status: false, message: getErrorMessage(e), data: [] };
  }
}

export async function getActiveUserChart(): Promise<ActiveUserChartResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/active-users",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeSeriesChart(response.data);
  } catch (e) {
    console.warn(e);
    return { date: [], value: [] };
  }
}

export async function getRegistrationChart(): Promise<RegistrationChartResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/registrations",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeSeriesChart(response.data);
  } catch (e) {
    console.warn(e);
    return { date: [], value: [] };
  }
}

export async function getConversionFunnel(): Promise<ConversionFunnelResponse> {
  try {
    const response = await axios.get(
      "/admin/analytics/funnel",
      adminAnalyticsNoCacheConfig,
    );
    return normalizeFunnel(response.data);
  } catch (e) {
    console.warn(e);
    return { registered: 0, ever_subscribed: 0, active_subscribed: 0 };
  }
}
