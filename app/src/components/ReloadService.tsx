import { version } from "@/conf/bootstrap.ts";
import { useTranslation } from "react-i18next";
import { getMemory, setMemory } from "@/utils/memory.ts";
import { Badge } from "@/components/ui/badge.tsx";
import { toast } from "sonner";
import { useEffect, useRef } from "react";
import { getFreshSiteInfo } from "@/admin/api/info.ts";

const serverUpdateCheckInterval = 60_000;

function ReloadPrompt() {
  const { t } = useTranslation();
  const initialRuntimeID = useRef<string>("");
  const notifiedRuntimeID = useRef<string>("");

  useEffect(() => {
    let closed = false;

    const checkServerRuntime = async () => {
      if (document.visibilityState === "hidden") return;

      try {
        const info = await getFreshSiteInfo();
        const runtimeID = (info.runtime_id || "").trim();
        if (!runtimeID || closed) return;

        if (!initialRuntimeID.current) {
          initialRuntimeID.current = runtimeID;
          return;
        }

        if (
          runtimeID === initialRuntimeID.current ||
          runtimeID === notifiedRuntimeID.current
        ) {
          return;
        }

        notifiedRuntimeID.current = runtimeID;
        toast.info(t("service.server-updated-title"), {
          description: t("service.server-updated-description"),
          duration: Infinity,
          action: {
            label: t("service.refresh"),
            onClick: () => window.location.reload(),
          },
        });
      } catch (error) {
        console.debug("[service] cannot check server runtime", error);
      }
    };

    void checkServerRuntime();
    const timer = window.setInterval(
      checkServerRuntime,
      serverUpdateCheckInterval,
    );

    return () => {
      closed = true;
      window.clearInterval(timer);
    };
  }, [t]);

  const before = getMemory("version");
  if (version.length === 0) {
    return <></>;
  }
  if (before.length > 0 && before !== version) {
    setMemory("version", version);

    setTimeout(() => {
      toast.success(t("service.update-success"), {
        description: (
          <div className="flex items-center">
            <Badge variant={`outline`} className={`font-medium mr-1`}>
              v{version}
            </Badge>
            {t("service.update-success-prompt")}
          </div>
        ),
      });
    }, 2500);

    console.debug(
      `[service] service worker updated (from ${before} to ${version})`,
    );
  }
  setMemory("version", version);

  return <></>;
}

export default ReloadPrompt;
