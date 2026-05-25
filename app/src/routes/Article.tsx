import "@/assets/pages/article.less";
import { Button } from "@/components/ui/button.tsx";
import router from "@/router.tsx";
import { Check, ChevronLeft, Cloud, Files, Globe, Loader2 } from "lucide-react";
import { Textarea } from "@/components/ui/textarea.tsx";
import { useTranslation } from "react-i18next";
import { useDispatch, useSelector } from "react-redux";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { useState } from "react";
import ModelArea from "@/components/home/ModelArea.tsx";
import { Toggle } from "@/components/ui/toggle.tsx";
import { selectModel, selectWeb, setWeb } from "@/store/chat.ts";
import { Label } from "@/components/ui/label.tsx";
import {
  apiEndpoint,
  tokenField,
  websocketEndpoint,
} from "@/conf/bootstrap.ts";
import { getMemory } from "@/utils/memory.ts";
import { Progress } from "@/components/ui/progress.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import { toast } from "sonner";
import { getErrorMessage } from "@/utils/base.ts";

type ProgressProps = {
  current: number;
  total: number;
};

function GenerateProgress({
  current,
  total,
  quota,
}: ProgressProps & { quota: number }) {
  const { t } = useTranslation();
  const progress = total > 0 ? (100 * current) / total : 0;

  return (
    <div className={`article-progress w-full mb-4`}>
      <p
        className={`select-none mt-4 mb-2.5 flex flex-row items-center content-center w-full justify-center text-center`}
      >
        {total !== 0 && current === total ? (
          <>
            <Check
              className={`h-5 w-5 mr-2 inline-block animate-out shrink-0`}
            />
            {t("article.generate-success")}
          </>
        ) : (
          <>
            <Loader2
              className={`h-5 w-5 mr-2 inline-block animate-spin shrink-0`}
            />
            {t("article.progress-title", { current, total })}
          </>
        )}
      </p>
      <Progress value={progress} />
      <div
        className={`article-quota flex flex-row mt-4 border border-input rounded-md py-1 px-3 select-none w-max items-center mx-auto`}
      >
        <Cloud className={`h-4 w-4 mr-2`} />
        <p>{quota.toFixed(2)}</p>
      </div>
    </div>
  );
}

function ArticleContent() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const web = useSelector(selectWeb);
  const model = useSelector(selectModel);

  const [prompt, setPrompt] = useState("");
  const [title, setTitle] = useState("");
  const [progress, setProgress] = useState(false);

  const [state, setState] = useState<ProgressProps>({ current: 0, total: 0 });
  const [quota, setQuota] = useState<number>(0);
  const [hash, setHash] = useState("");

  function clear() {
    setPrompt("");
    setTitle("");
    setHash("");
    setProgress(false);
    setQuota(0);
    setState({ current: 0, total: 0 });
  }

  function generate() {
    if (!title.trim() || !prompt.trim()) return;

    setProgress(true);
    const connection = new WebSocket(`${websocketEndpoint}/article/create`);
    let settled = false;

    const fail = (reason?: string, close = true) => {
      if (settled) return;
      settled = true;
      toast.error(t("article.generate-failed"), {
        description: reason
          ? `${t("article.generate-failed-prompt")} (${reason})`
          : t("article.generate-failed-prompt"),
      });
      setProgress(false);
      if (close) connection.close();
    };

    connection.onopen = () => {
      connection.send(
        JSON.stringify({
          token: getMemory(tokenField),
          web,
          title: title.trim(),
          prompt: prompt.trim(),
          model,
        }),
      );
    };

    connection.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data) as {
          hash?: string;
          data?: ProgressProps & { quota?: number };
          error?: string;
        };

        if (data.error) {
          fail(data.error);
          return;
        }

        if (typeof data.hash === "string") {
          if (data.hash.length === 0) {
            fail("empty download hash");
            return;
          }

          settled = true;
          toast.success(t("article.generate-success"), {
            description: t("article.generate-success-prompt"),
          });
          setHash(data.hash);
          return;
        }

        if (data.data) {
          if (typeof data.data.quota === "number") {
            setQuota((current) => current + data.data!.quota!);
          }
          setState({
            current: data.data.current,
            total: data.data.total,
          });
          return;
        }

        fail("invalid websocket message");
      } catch (error) {
        fail(getErrorMessage(error));
      }
    };

    connection.onerror = (e: Event) => {
      console.debug(`[article] error during generation: ${e}`);
      fail(e.toString());
    };

    connection.onclose = () => {
      fail("websocket connection closed", false);
    };
  }

  return progress ? (
    <>
      <GenerateProgress {...state} quota={quota} />
      {hash && (
        <div className={`article-action flex flex-row items-center my-4 gap-4`}>
          <Button
            variant={`outline`}
            className={`w-full whitespace-nowrap`}
            onClick={() => {
              location.href = `${apiEndpoint}/article/download/zip?hash=${hash}`;
            }}
          >
            {" "}
            {t("article.download-format", { name: "zip" })}{" "}
          </Button>

          <Button
            variant={`outline`}
            className={`w-full whitespace-nowrap`}
            onClick={() => {
              location.href = `${apiEndpoint}/article/download/tar?hash=${hash}`;
            }}
          >
            {" "}
            {t("article.download-format", { name: "tar" })}{" "}
          </Button>
        </div>
      )}
      <Button
        variant={`default`}
        className={`mt-5 w-full mx-auto`}
        onClick={clear}
      >
        {t("close")}
      </Button>
    </>
  ) : (
    <>
      <div className={`flex flex-row items-center mx-auto`}>
        <Toggle
          aria-label={t("chat.web-aria")}
          defaultPressed={false}
          onPressedChange={(state: boolean) => dispatch(setWeb(state))}
          variant={`outline`}
        >
          <Globe className={cn("h-4 w-4 web", web && "enable")} />
        </Toggle>
        <Label className={`ml-2.5 whitespace-nowrap`}>
          {t("article.web-checkbox")}
        </Label>
      </div>
      <Textarea
        placeholder={t("article.prompt-placeholder")}
        rows={3}
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
      />
      <Textarea
        placeholder={t("article.input-placeholder")}
        rows={8}
        value={title}
        onChange={(e) => setTitle(e.target.value)}
      />
      <ModelArea side={`bottom`} />
      <Button
        variant={`default`}
        className={`mt-5 w-full mx-auto`}
        onClick={generate}
        disabled={progress || !title.trim() || !prompt.trim()}
      >
        {t("article.generate")}
      </Button>
    </>
  );
}

function Wrapper() {
  const { t } = useTranslation();

  return (
    <Card className={`article-wrapper`}>
      <CardHeader className={`py-4`}>
        <CardTitle className={`article-title`}>
          <Files className={`h-5 w-5 mr-2`} />
          {t("article.title")}
        </CardTitle>
      </CardHeader>
      <CardContent className={`article-content`}>
        <ArticleContent />
      </CardContent>
    </Card>
  );
}

function Article() {
  return (
    <div className={`article-page`}>
      <div className={`article-container`}>
        <Button
          className={`action`}
          variant={`ghost`}
          size={`icon`}
          onClick={() => router.navigate("/")}
        >
          <ChevronLeft className={`h-5 w-5 back`} />
        </Button>
        <Wrapper />
      </div>
    </div>
  );
}

export default Article;
