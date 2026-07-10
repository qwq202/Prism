import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";
import type {
  PersonalizationRecord,
  SyncedPersonalizationSettings,
} from "@/types/personalization.ts";

export type PersonalizationResponse = {
  status: boolean;
  data?: PersonalizationRecord | null;
  code?: "revision_conflict";
  message?: string;
};

export async function loadPersonalizationSettings(): Promise<PersonalizationResponse> {
  try {
    const response = await axios.get("/personalization");
    return response.data as PersonalizationResponse;
  } catch (error) {
    return { status: false, message: getErrorMessage(error) };
  }
}

export async function savePersonalizationSettings(
  settings: SyncedPersonalizationSettings,
  baseRevision: number,
): Promise<PersonalizationResponse> {
  try {
    const response = await axios.post("/personalization", {
      settings,
      base_revision: baseRevision,
    });
    return response.data as PersonalizationResponse;
  } catch (error) {
    return { status: false, message: getErrorMessage(error) };
  }
}
