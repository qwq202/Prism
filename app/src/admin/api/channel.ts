import { Channel } from "@/admin/channel.ts";
import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";
import { CommonResponse } from "@/api/common.ts";
import { asArray, asNumber, asRecord, asString } from "./normalize.ts";

const adminAnalyticsNoCacheConfig = {
  prismCache: false,
  headers: {
    "Cache-Control": "no-cache",
    Pragma: "no-cache",
  },
} as const;

export type ChannelListResponse = CommonResponse & {
  data: Channel[];
};

export type GetChannelResponse = CommonResponse & {
  data?: Channel;
};

export async function listChannel(): Promise<ChannelListResponse> {
  try {
    const response = await axios.get("/admin/channel/list", {
      prismCache: false,
    });
    const data = asRecord(response.data);
    return {
      status: data.status === true,
      error: asString(data.error),
      message: asString(data.message),
      data: asArray<Channel>(data.data),
    };
  } catch (e) {
    return { status: false, error: getErrorMessage(e), data: [] };
  }
}

export async function getChannel(id: number): Promise<GetChannelResponse> {
  try {
    const response = await axios.get(`/admin/channel/get/${id}`, {
      prismCache: false,
    });
    return response.data as GetChannelResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function createChannel(channel: Channel): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/channel/create", channel);
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function updateChannel(
  id: number,
  channel: Channel,
): Promise<CommonResponse> {
  try {
    const response = await axios.post(`/admin/channel/update/${id}`, channel);
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function deleteChannel(id: number): Promise<CommonResponse> {
  try {
    const response = await axios.get(`/admin/channel/delete/${id}`);
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function activateChannel(id: number): Promise<CommonResponse> {
  try {
    const response = await axios.get(`/admin/channel/activate/${id}`);
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function deactivateChannel(id: number): Promise<CommonResponse> {
  try {
    const response = await axios.get(`/admin/channel/deactivate/${id}`);
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export type ChannelStat = {
  channel_id: number;
  requests: number;
  errors: number;
  error_rate: number;
};

export type ChannelStatsResponse = {
  stats: ChannelStat[];
};

export async function getChannelStats(
  channelIds?: number[],
): Promise<ChannelStatsResponse> {
  try {
    const response = await axios.post(
      "/admin/analytics/channel",
      {
        channel_ids: channelIds ?? [],
      },
      adminAnalyticsNoCacheConfig,
    );
    const data = asRecord(response.data);
    return {
      stats: asArray<unknown>(data.stats).map((item) => {
        const stat = asRecord(item);
        return {
          channel_id: asNumber(stat.channel_id),
          requests: asNumber(stat.requests),
          errors: asNumber(stat.errors),
          error_rate: asNumber(stat.error_rate),
        };
      }),
    };
  } catch (e) {
    console.warn(e);
    return { stats: [] };
  }
}
