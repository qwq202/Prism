import { CommonResponse } from "@/api/common.ts";
import { getErrorMessage } from "@/utils/base.ts";
import axios from "axios";
import { backendEndpoint } from "@/conf/env.ts";

export type TestWebSearchResponse = CommonResponse & {
  result: string;
};

export type whiteList = {
  enabled: boolean;
  custom: string;
  white_list: string[];
};

export type GeneralState = {
  title: string;
  logo: string;
  description: string;
  backend: string;
  docs: string;
  timezone: string;
  pwa_manifest: string;
  gravatar: string;
  debug_mode: boolean;
  realtime?: {
    ws?: {
      buffer_size?: number;
      aggregate?: boolean;
      aggregate_window_ms?: number;
    };
  };
};

export type MailState = {
  host: string;
  protocol: boolean;
  port: number;
  username: string;
  password: string;
  from: string;
  white_list: whiteList;
};

export type SearchState = {
  api_key: string;
  engines: string[];
  crop: boolean;
  crop_len: number;
  max_results: number;
  topic: string;
  depth: string;
};

export type TaskState = {
  model: string;
};

export type PasskeyUserVerification = "required" | "preferred" | "discouraged";

export type PasskeyAuthenticatorAttachment =
  | "any"
  | "platform"
  | "cross-platform";

export type PasskeyState = {
  enabled: boolean;
  rp_display_name: string;
  rp_id: string;
  user_verification: PasskeyUserVerification;
  authenticator_attachment: PasskeyAuthenticatorAttachment;
  allow_insecure_origin: boolean;
  origins: string;
};

export type AuthenticationState = {
  passkey: PasskeyState;
};

export type SecurityState = {
  check_type: string;
  check_models?: string[];

  text_database: string;
  regex_database: string;

  baidu_api_key: string;
  baidu_secret_key: string;

  custom_endpoint: string;
  custom_audit_token: string;

  blacklist_ips: string[];
  whitelist_ips: string[];
};

export type PaymentState = {
  stripe: {
    enabled: boolean;
    public_key: string;
    secret_key: string;
    webhook_secret: string;
  };
  epay: {
    domain: string;
    business_id: string;
    business_key: string;
    enabled: boolean;
    methods: string[];
    aggregation: boolean;
  };
  wechatpay?: {
    enabled: boolean;
    app_id: string;
    mch_id: string;
    serial_no: string;
    apiv3_key: string;
    wechatcertificate: string;
  };
  xunhupay?: {
    wechat_enabled: boolean;
    alipay_enabled: boolean;
    wechat_app_id: string;
    wechat_app_secret: string;
    alipay_app_id: string;
    alipay_app_secret: string;
    endpoint: string;
  };
  affiliate?: {
    enabled: boolean;
    commission_rate: number;
    min_withdraw: number;
    allow_existing_bind: boolean;
  };
};

export type SiteState = {
  close_register: boolean;
  currency: string;
  quota: number;
  buy_link: string;
  announcement: string;
  contact: string;
  footer: string;
  auth_footer: boolean;
  pre_deduct_quota: boolean;
  hide_key_docs: boolean;
};

export type CustomState = {
  custom_js: string;
  custom_css: string;
  custom_html: string;
  ga_tracking_id: string;
};

export type AutoTitleState = {
  enabled: boolean;
  model: string;
  max_len: number;
  min_msgs: number;
  overwrite: boolean;
  prompt: string;
};

export type S3StorageState = {
  endpoint: string;
  region: string;
  bucket: string;
  access_key: string;
  secret_key: string;
  public_base_url: string;
  force_path_style: boolean;
};

export type R2StorageState = {
  account_id: string;
  jurisdiction: string;
  bucket: string;
  access_key: string;
  secret_key: string;
  public_base_url: string;
};

export type GroupPricingState = {
  buy_price: number;
  consume_price: number;
  description: string;
};

export type CommonState = {
  cache: string[];
  expire: number;
  size: number;

  prompt_store: boolean;
  image_store: boolean;
  orphan_cleanup_enabled: boolean;
  orphan_cleanup_interval: number;
  storage_mode: "local" | "s3" | "r2";
  s3: S3StorageState;
  r2: R2StorageState;
  group: Record<string, GroupPricingState>;
};

export type SystemProps = {
  general: GeneralState;
  site: SiteState;
  mail: MailState;
  auth: AuthenticationState;
  search: SearchState;
  task: TaskState;
  common: CommonState;
  payment: PaymentState;
  security: SecurityState;
  custom: CustomState;
  auto_title?: AutoTitleState;
};

export type SystemResponse = CommonResponse & {
  data?: SystemProps;
};

export const initialSystemState: SystemProps = {
  general: {
    logo: "",
    description: "",
    title: "",
    backend: "",
    docs: "",
    timezone: "Asia/Shanghai",
    pwa_manifest: "",
    gravatar: "",
    debug_mode: false,
    realtime: {
      ws: {
        buffer_size: 24,
        aggregate: true,
        aggregate_window_ms: 20,
      },
    },
  },
  site: {
    close_register: false,
    currency: "cny",
    quota: 0,
    buy_link: "",
    announcement: "",
    contact: "",
    footer: "",
    auth_footer: false,
    pre_deduct_quota: true,
    hide_key_docs: false,
  },
  mail: {
    host: "",
    protocol: false,
    port: 465,
    username: "",
    password: "",
    from: "",
    white_list: {
      enabled: false,
      custom: "",
      white_list: [],
    },
  },
  auth: {
    passkey: {
      enabled: false,
      rp_display_name: "Prism",
      rp_id: "",
      user_verification: "preferred",
      authenticator_attachment: "any",
      allow_insecure_origin: false,
      origins: "",
    },
  },
  search: {
    api_key: "",
    engines: [],
    crop: false,
    crop_len: 1000,
    max_results: 5,
    topic: "general",
    depth: "basic",
  },
  task: {
    model: "",
  },
  common: {
    cache: [],
    expire: 3600,
    size: 1,
    prompt_store: false,
    image_store: false,
    orphan_cleanup_enabled: false,
    orphan_cleanup_interval: 60,
    storage_mode: "local",
    s3: {
      endpoint: "",
      region: "",
      bucket: "",
      access_key: "",
      secret_key: "",
      public_base_url: "",
      force_path_style: false,
    },
    r2: {
      account_id: "",
      jurisdiction: "",
      bucket: "",
      access_key: "",
      secret_key: "",
      public_base_url: "",
    },
    group: {
      anonymous: {
        buy_price: 1,
        consume_price: 1,
        description: "",
      },
      normal: {
        buy_price: 1,
        consume_price: 1,
        description: "",
      },
      basic: {
        buy_price: 1,
        consume_price: 1,
        description: "",
      },
      standard: {
        buy_price: 1,
        consume_price: 1,
        description: "",
      },
      pro: {
        buy_price: 1,
        consume_price: 1,
        description: "",
      },
      admin: {
        buy_price: 1,
        consume_price: 1,
        description: "",
      },
    },
  },
  payment: {
    stripe: {
      enabled: false,
      public_key: "",
      secret_key: "",
      webhook_secret: "",
    },
    epay: {
      domain: "",
      business_id: "",
      business_key: "",
      enabled: false,
      methods: [],
      aggregation: false,
    },
    affiliate: {
      enabled: false,
      commission_rate: 0.1,
      min_withdraw: 10,
      allow_existing_bind: false,
    },
  },
  security: {
    check_type: "",
    check_models: [],
    text_database: "",
    regex_database: "",
    baidu_api_key: "",
    baidu_secret_key: "",
    custom_endpoint: "",
    custom_audit_token: "",
    blacklist_ips: [],
    whitelist_ips: [],
  },
  custom: {
    custom_js: "",
    custom_css: "",
    custom_html: "",
    ga_tracking_id: "",
  },
  auto_title: {
    enabled: false,
    model: "",
    max_len: 50,
    min_msgs: 6,
    overwrite: false,
    prompt: "",
  },
};

export async function getConfig(): Promise<SystemResponse> {
  try {
    const response = await axios.get("/admin/config/view");
    const data = response.data as SystemResponse;
    if (data.status && data.data) {
      // init system data pre-format

      data.data.general.timezone =
        data.data.general.timezone || "Asia/Shanghai";
      data.data.mail.white_list.white_list =
        data.data.mail.white_list.white_list || commonWhiteList;
      data.data.search.engines = data.data.search.engines || [];
      data.data.search.crop_len =
        data.data.search.crop_len && data.data.search.crop_len > 0
          ? data.data.search.crop_len
          : 1000;
      data.data.search.topic =
        data.data.search.topic &&
        ["general", "news", "finance"].includes(data.data.search.topic)
          ? data.data.search.topic
          : "general";
      data.data.search.depth =
        data.data.search.depth &&
        ["basic", "advanced", "fast", "ultra-fast"].includes(
          data.data.search.depth,
        )
          ? data.data.search.depth
          : "basic";

      data.data.site.currency = data.data.site.currency || "cny";
      const auth = (data.data.auth = data.data.auth || {
        passkey: {
          enabled: false,
          rp_display_name: "Prism",
          rp_id: "",
          user_verification: "preferred",
          authenticator_attachment: "any",
          allow_insecure_origin: false,
          origins: "",
        },
      });
      const passkey = (auth.passkey = auth.passkey || {
        enabled: false,
        rp_display_name: "Prism",
        rp_id: "",
        user_verification: "preferred",
        authenticator_attachment: "any",
        allow_insecure_origin: false,
        origins: "",
      });
      passkey.enabled = !!passkey.enabled;
      passkey.rp_display_name = passkey.rp_display_name || "Prism";
      passkey.rp_id = passkey.rp_id || "";
      passkey.user_verification = [
        "required",
        "preferred",
        "discouraged",
      ].includes(passkey.user_verification)
        ? passkey.user_verification
        : "preferred";
      passkey.authenticator_attachment = [
        "any",
        "platform",
        "cross-platform",
      ].includes(passkey.authenticator_attachment)
        ? passkey.authenticator_attachment
        : "any";
      passkey.allow_insecure_origin = !!passkey.allow_insecure_origin;
      passkey.origins = passkey.origins || "";

      data.data.common.storage_mode =
        data.data.common.storage_mode === "s3"
          ? "s3"
          : data.data.common.storage_mode === "r2"
          ? "r2"
          : "local";
      const s3 = (data.data.common.s3 = data.data.common.s3 || {
        endpoint: "",
        region: "",
        bucket: "",
        access_key: "",
        secret_key: "",
        public_base_url: "",
        force_path_style: false,
      });
      s3.endpoint = s3.endpoint || "";
      s3.region = s3.region || "";
      s3.bucket = s3.bucket || "";
      s3.access_key = s3.access_key || "";
      s3.secret_key = s3.secret_key || "";
      s3.public_base_url = s3.public_base_url || "";
      s3.force_path_style = !!s3.force_path_style;
      const r2 = (data.data.common.r2 = data.data.common.r2 || {
        account_id: "",
        jurisdiction: "",
        bucket: "",
        access_key: "",
        secret_key: "",
        public_base_url: "",
      });
      r2.account_id = r2.account_id || "";
      r2.jurisdiction = r2.jurisdiction || "";
      r2.bucket = r2.bucket || "";
      r2.access_key = r2.access_key || "";
      r2.secret_key = r2.secret_key || "";
      r2.public_base_url = r2.public_base_url || "";
      data.data.common.orphan_cleanup_enabled =
        !!data.data.common.orphan_cleanup_enabled;
      data.data.common.orphan_cleanup_interval =
        typeof data.data.common.orphan_cleanup_interval === "number" &&
        data.data.common.orphan_cleanup_interval > 0
          ? data.data.common.orphan_cleanup_interval
          : 60;

      if (
        !data.data.common.group ||
        Object.keys(data.data.common.group).length === 0
      ) {
        data.data.common.group = {
          anonymous: {
            buy_price: 1,
            consume_price: 1,
            description: "",
          },
          normal: {
            buy_price: 1,
            consume_price: 1,
            description: "",
          },
          basic: {
            buy_price: 1,
            consume_price: 1,
            description: "",
          },
          standard: {
            buy_price: 1,
            consume_price: 1,
            description: "",
          },
          pro: {
            buy_price: 1,
            consume_price: 1,
            description: "",
          },
          admin: {
            buy_price: 1,
            consume_price: 1,
            description: "",
          },
        };
      }

      const rt = (data.data.general.realtime =
        data.data.general.realtime || {});
      const ws = (rt.ws = rt.ws || {});
      ws.buffer_size =
        typeof ws.buffer_size === "number" && ws.buffer_size > 0
          ? ws.buffer_size
          : 1;
      ws.aggregate = typeof ws.aggregate === "boolean" ? ws.aggregate : true;
      ws.aggregate_window_ms =
        typeof ws.aggregate_window_ms === "number" && ws.aggregate_window_ms > 0
          ? ws.aggregate_window_ms
          : 20;

      const at = (data.data.auto_title = data.data.auto_title || {
        enabled: false,
        model: "",
        max_len: 50,
        min_msgs: 6,
        overwrite: false,
        prompt: "",
      });
      at.enabled = !!at.enabled;
      at.model = at.model || "";
      at.max_len =
        typeof at.max_len === "number" && at.max_len > 0 ? at.max_len : 50;
      at.min_msgs =
        typeof at.min_msgs === "number" && at.min_msgs > 0 ? at.min_msgs : 6;
      at.overwrite = !!at.overwrite;
      at.prompt = at.prompt || "";
    }

    return data;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function setConfig(config: SystemProps): Promise<CommonResponse> {
  try {
    const response = await axios.post(`/admin/config/update`, config);
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

type UploadResponse = CommonResponse & {
  url?: string;
};

type PasswordUpdateResponse = CommonResponse & {
  token?: string;
};

export async function uploadFavicon(file: File): Promise<UploadResponse> {
  try {
    const formData = new FormData();
    formData.append("file", file);

    const response = await axios.post(`/admin/favicon/upload`, formData, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
    });
    return response.data as UploadResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function uploadResource(file: File): Promise<UploadResponse> {
  try {
    const formData = new FormData();
    formData.append("file", file);

    const response = await axios.post(`/admin/resource/upload`, formData, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
    });

    const data = response.data as UploadResponse;
    if (data.status) {
      data.url = backendEndpoint + data.url;
    }

    return data;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function updateRootPassword(
  password: string,
): Promise<PasswordUpdateResponse> {
  try {
    const response = await axios.post(`/admin/user/root`, { password });
    return response.data as PasswordUpdateResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function testWebSearching(
  query: string,
): Promise<TestWebSearchResponse> {
  try {
    const response = await axios.get(
      `/admin/config/test/search?query=${encodeURIComponent(query)}`,
    );
    return response.data as TestWebSearchResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e), result: "" };
  }
}

export async function testStorageConfig(
  config: SystemProps,
): Promise<CommonResponse> {
  try {
    const response = await axios.post(`/admin/config/test/storage`, config);
    return response.data as CommonResponse;
  } catch (e) {
    return { status: false, error: getErrorMessage(e) };
  }
}

export enum AuditTypes {
  None = "none",
  Dict = "dict",
  Regex = "regex",
  Baidu = "baidu",
  Custom = "custom",
}

export const auditTypes: string[] = [
  AuditTypes.None,
  AuditTypes.Dict,
  AuditTypes.Regex,
  AuditTypes.Baidu,
  AuditTypes.Custom,
];

export const commonWhiteList: string[] = [
  "gmail.com",
  "outlook.com",
  "yahoo.com",
  "hotmail.com",
  "foxmail.com",
  "icloud.com",
  "qq.com",
  "163.com",
  "126.com",
];
