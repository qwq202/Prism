import { ChargeBaseProps } from "@/admin/charge.ts";
import React from "react";
import { BotIcon, ServerIcon, UserIcon } from "lucide-react";

export const UserRole = "user";
export const AssistantRole = "assistant";
export const SystemRole = "system";
export const VirtualRolePrefix = "virtualRole::";
export const VirtualWebSearchRole = "virtualRole::websearch";
export type Role = typeof UserRole | typeof AssistantRole | typeof SystemRole;
export const Roles = [UserRole, AssistantRole, SystemRole];

export type MessageToolCall = {
  index: number;
  type: string;
  id: string;
  function: {
    name: string;
    arguments: string;
  };
  status?: "start" | "executing" | "pending" | "success" | "error";
  result?: string;
  error?: string;
};

export const GetRoleIcon = (role: string) => {
  switch (role) {
    case UserRole:
      return React.createElement(UserIcon);
    case AssistantRole:
      return React.createElement(BotIcon);
    case SystemRole:
      return React.createElement(ServerIcon);
    default:
      return React.createElement(UserIcon);
  }
};

export const getRoleIcon = GetRoleIcon;

export type Message = {
  role: string;
  content: string;
  model?: string;
  keyword?: string;
  quota?: number;
  end?: boolean;
  plan?: boolean;
  search_query?: {
    type: string;
    search_queries: string[];
  };
  search_result?: {
    type: string;
    search_results: Array<{
      url: string;
      title: string;
      snippet: string;
      published_at?: number;
      site_name?: string;
      site_icon?: string;
    }>;
  };
  search_index?: {
    type: string;
    search_indexes: Array<{
      url: string;
      cite_index: number;
    }>;
  };
  tool_calls?: MessageToolCall[];
  tool_call_id?: string;
  name?: string;
  response_type?: string;
};

export type Model = {
  id: string;
  name: string;
  channel_type?: string;
  description?: string;
  free: boolean;
  auth: boolean;
  default: boolean;
  high_context: boolean;
  drawing_model?: boolean;
  vision_model?: boolean;
  ocr_model?: boolean;
  reverse_model?: boolean;
  avatar: string;
  tag?: string[];

  price?: ChargeBaseProps;
};

export type Id = number;

export type ConversationInstance = {
  id: number;
  name: string;
  message: Message[];
  model?: string;
  shared?: boolean;
  favorite?: boolean;
  updated_at?: string;
  local_key?: string;
};

export type PlanItem = {
  id: string;
  name: string;
  value: number;
  icon: string;
  models: string[];
};

export type Plan = {
  level: number;
  price: number;
  sellable?: boolean;
  quota?: number;
  reset_interval?: number;
  weekly_quota?: number;
  items: PlanItem[];
  discounts?: Record<string, number>;
};

export type Plans = Plan[];

export function newModel(id: string, name?: string, avatar?: string): Model {
  return {
    id,
    name: name ?? id,
    avatar: avatar ?? "",
    free: false,
    auth: false,
    default: false,
    high_context: false,
  };
}
