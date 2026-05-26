import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";

export type ModelUsageTrendPoint = {
  time: string;
  requests: number;
  successes: number;
  avg_latency: number;
  availability: number;
};

export type ModelUsageStats = {
  model: string;
  window_hours: number;
  request_count: number;
  success_count: number;
  failure_count: number;
  availability_failures: number;
  tps: number;
  avg_latency: number;
  success_rate: number;
  availability: number;
  latency_trend: ModelUsageTrendPoint[];
  availability_trend: ModelUsageTrendPoint[];
};

type ModelUsageStatsResponse = {
  status: boolean;
  message?: string;
  data?: ModelUsageStats;
};

type ModelUsageStatsBatchResponse = {
  status: boolean;
  message?: string;
  data?: Record<string, ModelUsageStats>;
};

export async function getModelUsageStats(
  model: string,
): Promise<ModelUsageStats | null> {
  try {
    const response = await axios.get<ModelUsageStatsResponse>(
      "/v1/model-metrics",
      {
        params: { model },
        prismCache: false,
      },
    );
    if (!response.data.status) {
      console.warn(response.data.message || "model metrics unavailable");
      return null;
    }

    return response.data.data ?? null;
  } catch (e) {
    console.warn(getErrorMessage(e));
    return null;
  }
}

export async function getModelsUsageStats(
  models: string[],
): Promise<Record<string, ModelUsageStats>> {
  const ids = Array.from(new Set(models.map((model) => model.trim()))).filter(
    Boolean,
  );
  if (ids.length === 0) return {};

  try {
    const response = await axios.get<ModelUsageStatsBatchResponse>(
      "/v1/model-metrics",
      {
        params: { models: ids.join(",") },
        prismCache: false,
      },
    );
    if (!response.data.status) {
      console.warn(response.data.message || "model metrics unavailable");
      return {};
    }

    return response.data.data ?? {};
  } catch (e) {
    console.warn(getErrorMessage(e));
    return {};
  }
}
