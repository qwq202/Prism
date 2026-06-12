import { CommonResponse } from "@/api/common.ts";
import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";

export type Record = {
  id: number;
  user_id: number;
  username: string;
  type: string;
  token_name: string;
  model: string;
  input_tokens: number;
  output_tokens: number;
  quota: number;
  duration: number;
  detail: string;
  prompts: string;
  response_prompts: string;
  channel?: number;
  channel_name?: string;
  created_at: string;
};
export type RecordData = {
  total: number;
  total_input_tokens?: number;
  total_output_tokens?: number;
  total_tokens?: number;
  records: Record[];
};

export type RecordStats = {
  billing_today: number;
  billing_month: number;
  request_today: number;
  request_month: number;
  rpm: number;
  tpm: number;
  total_tokens: number;
};

export type RecordUsageModel = {
  name: string;
  value: number;
  count: number;
};

export type RecordUsageSummary = {
  model_count: number;
  top_model: string;
  average_quota: number;
  max_quota: number;
  models: RecordUsageModel[];
};

export type RecordQuery = {
  user_id?: number;
  username?: string;
  start_time?: string;
  end_time?: string;
  token_name?: string;
  model?: string;
  type?: RecordType;
  show_channel?: boolean;
  self?: boolean;
};

type ListRecordsResponse = CommonResponse & {
  data?: RecordData;
};

type RecordStatsResponse = CommonResponse & {
  data?: RecordStats;
};

type RecordUsageSummaryResponse = CommonResponse & {
  data?: RecordUsageSummary;
};

export enum RecordType {
  All = "all",
  Topup = "topup",
  Consume = "consume",
  System = "system",
}

export const RecordTypes = [
  RecordType.All,
  RecordType.Topup,
  RecordType.Consume,
  RecordType.System,
];

export async function listRecords(
  page: number,
  options?: RecordQuery,
): Promise<ListRecordsResponse> {
  try {
    const payload: Partial<RecordQuery> = { ...options };
    if (options && options.show_channel === undefined) {
      delete payload.show_channel;
    }
    const resp = await axios.post(`/record/view?page=${page}`, payload);
    return resp.data as ListRecordsResponse;
  } catch (e) {
    return {
      status: false,
      message: getErrorMessage(e),
    };
  }
}

export async function getRecordStats(
  options?: Pick<RecordQuery, "self">,
): Promise<RecordStatsResponse> {
  try {
    const resp = await axios.post(`/record/stats`, options ?? {});
    return resp.data as RecordStatsResponse;
  } catch (e) {
    return {
      status: false,
      message: getErrorMessage(e),
    };
  }
}

export async function getRecordUsageSummary(
  options?: Omit<RecordQuery, "show_channel" | "type">,
): Promise<RecordUsageSummaryResponse> {
  try {
    const resp = await axios.post(`/record/usage-summary`, options ?? {});
    return resp.data as RecordUsageSummaryResponse;
  } catch (e) {
    return {
      status: false,
      message: getErrorMessage(e),
    };
  }
}
