import {
  getDev,
  getRestApi,
  getTokenField,
  getWebsocketApi,
} from "@/conf/env.ts";
import { syncSiteInfo } from "@/admin/api/info.ts";
import { setAxiosConfig } from "@/conf/api.ts";
import { version as _version } from "./version.json";
import { markVolatileMemoryKey } from "@/utils/memory.ts";

export const version: string = _version; // version of the current build
export const dev: boolean = getDev(); // is in development mode (for debugging, in localhost origin)
export const deploy: boolean = true; // is production environment (for api endpoint)
export const tokenField = getTokenField(deploy); // token field name for storing token
markVolatileMemoryKey(tokenField);

export const apiEndpoint: string = getRestApi(deploy); // api endpoint for rest api calls
export const websocketEndpoint: string = getWebsocketApi(deploy); // api endpoint for websocket calls

setAxiosConfig({
  endpoint: apiEndpoint,
  token: tokenField,
});

syncSiteInfo();
