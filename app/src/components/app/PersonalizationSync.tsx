import { useCallback, useEffect, useRef, useState } from "react";
import { shallowEqual, useDispatch, useSelector } from "react-redux";
import {
  loadPersonalizationSettings,
  savePersonalizationSettings,
  type PersonalizationResponse,
} from "@/api/personalization.ts";
import { selectAuthenticated, selectUsername } from "@/store/auth.ts";
import * as settings from "@/store/settings.ts";

const PERSONALIZATION_SAVE_DELAY_MS = 750;
const PERSONALIZATION_REFRESH_INTERVAL_MS = 60_000;

function getFailureStatus(): "offline" | "error" {
  return typeof navigator !== "undefined" && navigator.onLine === false
    ? "offline"
    : "error";
}

function PersonalizationSync() {
  const dispatch = useDispatch();
  const authenticated = useSelector(selectAuthenticated);
  const username = useSelector(selectUsername);
  const owner = useSelector(settings.personalizationSyncOwnerSelector);
  const dirty = useSelector(settings.personalizationSyncDirtySelector);
  const revision = useSelector(settings.personalizationSyncRevisionSelector);
  const changeID = useSelector(settings.personalizationSyncChangeIDSelector);
  const syncRequest = useSelector(settings.personalizationSyncRequestSelector);
  const personalization = useSelector(
    settings.syncedPersonalizationSettingsSelector,
    shallowEqual,
  );
  const [readyAccount, setReadyAccount] = useState("");
  const personalizationRef = useRef(personalization);
  const dirtyRef = useRef(dirty);
  const revisionRef = useRef(revision);
  const changeIDRef = useRef(changeID);
  const lastRefreshRef = useRef(0);

  personalizationRef.current = personalization;
  dirtyRef.current = dirty;
  revisionRef.current = revision;
  changeIDRef.current = changeID;

  const setFailure = useCallback(
    (message?: string) => {
      dispatch(
        settings.setPersonalizationSyncStatus({
          status: getFailureStatus(),
          error: message || "",
        }),
      );
    },
    [dispatch],
  );

  const applySaveResponse = useCallback(
    (response: PersonalizationResponse, capturedChangeID: number) => {
      if (response.status && response.data) {
        dispatch(
          settings.markPersonalizationSynced({
            revision: response.data.revision,
            updated_at: response.data.updated_at,
            change_id: capturedChangeID,
          }),
        );
        return true;
      }

      if (response.code === "revision_conflict" && response.data) {
        if (changeIDRef.current !== capturedChangeID) {
          dispatch(
            settings.rebasePersonalizationSync({
              revision: response.data.revision,
              updated_at: response.data.updated_at,
            }),
          );
        } else {
          dispatch(settings.hydratePersonalization(response.data));
          dispatch(
            settings.setPersonalizationSyncStatus({
              status: "conflict",
              error: "",
            }),
          );
        }
        return false;
      }

      setFailure(response.message);
      return false;
    },
    [dispatch, setFailure],
  );

  useEffect(() => {
    if (!authenticated || !username) {
      setReadyAccount("");
      dispatch(
        settings.setPersonalizationSyncStatus({ status: "idle", error: "" }),
      );
      return;
    }
    if (owner !== username) {
      setReadyAccount("");
      dispatch(settings.preparePersonalizationAccount(username));
    }
  }, [authenticated, dispatch, owner, username]);

  useEffect(() => {
    if (!authenticated || !username || owner !== username) return;

    let cancelled = false;
    setReadyAccount("");
    dispatch(
      settings.setPersonalizationSyncStatus({ status: "loading", error: "" }),
    );

    void (async () => {
      const response = await loadPersonalizationSettings();
      if (cancelled) return;

      if (!response.status) {
        setFailure(response.message);
        setReadyAccount(username);
        return;
      }
      lastRefreshRef.current = Date.now();

      const remote = response.data;
      if (!remote) {
        const capturedChangeID = changeIDRef.current;
        const saved = await savePersonalizationSettings(
          personalizationRef.current,
          0,
        );
        if (cancelled) return;
        applySaveResponse(saved, capturedChangeID);
        setReadyAccount(username);
        return;
      }

      if (dirtyRef.current && revisionRef.current > 0) {
        const capturedChangeID = changeIDRef.current;
        const saved = await savePersonalizationSettings(
          personalizationRef.current,
          revisionRef.current,
        );
        if (cancelled) return;
        applySaveResponse(saved, capturedChangeID);
      } else {
        dispatch(settings.hydratePersonalization(remote));
      }
      setReadyAccount(username);
    })();

    return () => {
      cancelled = true;
    };
  }, [
    applySaveResponse,
    authenticated,
    dispatch,
    owner,
    setFailure,
    syncRequest,
    username,
  ]);

  useEffect(() => {
    if (
      !authenticated ||
      !username ||
      owner !== username ||
      readyAccount !== username ||
      !dirty
    ) {
      return;
    }

    const capturedChangeID = changeID;
    const timer = window.setTimeout(() => {
      dispatch(
        settings.setPersonalizationSyncStatus({ status: "saving", error: "" }),
      );
      void savePersonalizationSettings(personalization, revision).then(
        (response) => applySaveResponse(response, capturedChangeID),
      );
    }, PERSONALIZATION_SAVE_DELAY_MS);

    return () => window.clearTimeout(timer);
  }, [
    applySaveResponse,
    authenticated,
    changeID,
    dirty,
    dispatch,
    owner,
    personalization,
    readyAccount,
    revision,
    username,
  ]);

  useEffect(() => {
    if (!authenticated) return;

    const requestRefresh = () => {
      if (
        Date.now() - lastRefreshRef.current <
        PERSONALIZATION_REFRESH_INTERVAL_MS
      ) {
        return;
      }
      dispatch(settings.requestPersonalizationSync());
    };
    const handleVisibility = () => {
      if (document.visibilityState === "visible") requestRefresh();
    };

    const handleOnline = () => {
      dispatch(settings.requestPersonalizationSync());
    };

    window.addEventListener("online", handleOnline);
    window.addEventListener("focus", requestRefresh);
    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("focus", requestRefresh);
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [authenticated, dispatch]);

  return null;
}

export default PersonalizationSync;
