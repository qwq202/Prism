import type { TFunction } from "i18next";

const errorKeyMap: Record<string, string> = {
  "current user is banned": "errors.current-user-banned",
  "invalid username or password": "errors.invalid-username-or-password",
  "invalid username or password format":
    "errors.invalid-username-or-password-format",
  "deeptrain mode is disabled": "errors.deeptrain-disabled",
  "cannot validate access token": "errors.invalid-access-token",
  "this site is not open for registration": "errors.registration-closed",
  "relay api is disabled": "errors.relay-api-disabled",
};

export function localizeError(t: TFunction, error?: string | null) {
  const fallback = error?.trim() || t("unknown");
  const key = errorKeyMap[fallback.toLowerCase()];
  return key ? t(key) : fallback;
}
