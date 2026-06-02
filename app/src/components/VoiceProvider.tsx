import { ChatAction } from "@/components/home/assemblies/ChatAction.tsx";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Mic } from "lucide-react";
import React from "react";
import { setMemory } from "@/utils/memory.ts";

type BrowserSpeechRecognitionResult = {
  isFinal: boolean;
  [index: number]: {
    transcript: string;
  };
};

type BrowserSpeechRecognitionEvent = Event & {
  resultIndex: number;
  results: {
    length: number;
    [index: number]: BrowserSpeechRecognitionResult;
  };
};

type BrowserSpeechRecognitionErrorEvent = Event & {
  error: string;
};

type BrowserSpeechRecognition = {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  start: () => void;
  stop: () => void;
  abort: () => void;
  onstart: (() => void) | null;
  onend: (() => void) | null;
  onresult: ((event: BrowserSpeechRecognitionEvent) => void) | null;
  onerror: ((event: BrowserSpeechRecognitionErrorEvent) => void) | null;
};

type BrowserSpeechRecognitionConstructor = new () => BrowserSpeechRecognition;

type WindowWithSpeechRecognition = Window &
  typeof globalThis & {
    SpeechRecognition?: BrowserSpeechRecognitionConstructor;
    webkitSpeechRecognition?: BrowserSpeechRecognitionConstructor;
  };

type VoiceActionProps = {
  value: string;
  onValueChange: (value: string) => void;
  target?: React.RefObject<HTMLTextAreaElement | null>;
};

function getSpeechRecognitionConstructor() {
  const speechWindow = window as WindowWithSpeechRecognition;
  return speechWindow.SpeechRecognition ?? speechWindow.webkitSpeechRecognition;
}

function getSpeechRecognitionErrorKey(error: string) {
  switch (error) {
    case "no-speech":
      return "chat.voice-error-no-speech";
    case "audio-capture":
      return "chat.voice-error-audio-capture";
    case "network":
      return "chat.voice-error-network";
    case "not-allowed":
    case "service-not-allowed":
      return "chat.voice-error-not-allowed";
    case "aborted":
      return "chat.voice-error-aborted";
    case "language-not-supported":
    case "phrases-not-supported":
      return "chat.voice-error-language";
    default:
      return "chat.voice-error";
  }
}

function appendTranscript(value: string, transcript: string) {
  const text = transcript.trim();
  if (text.length === 0) return value;

  const separator = value.length === 0 || /\s$/.test(value) ? "" : " ";
  return `${value}${separator}${text}`;
}

function getRecognitionLanguage(language: string) {
  const normalized = language.toLowerCase();

  if (normalized === "cn" || normalized.startsWith("zh-cn")) return "zh-CN";
  if (normalized === "tw" || normalized.startsWith("zh-tw")) return "zh-TW";
  if (normalized.startsWith("ja")) return "ja-JP";
  if (normalized.startsWith("ru")) return "ru-RU";
  if (normalized.startsWith("en")) return "en-US";

  return navigator.language || "zh-CN";
}

export function VoiceAction({
  value,
  onValueChange,
  target,
}: VoiceActionProps) {
  const { t, i18n } = useTranslation();
  const [listening, setListening] = React.useState(false);
  const recognitionRef = React.useRef<BrowserSpeechRecognition | null>(null);
  const baseValueRef = React.useRef("");
  const finalTranscriptRef = React.useRef("");
  const manuallyStoppingRef = React.useRef(false);

  React.useEffect(() => {
    return () => {
      recognitionRef.current?.abort();
    };
  }, []);

  function updateInput(transcript: string) {
    const nextValue = appendTranscript(baseValueRef.current, transcript);
    onValueChange(nextValue);
    setMemory("history", nextValue);
    target?.current?.focus();
  }

  function stopRecognition() {
    manuallyStoppingRef.current = true;
    recognitionRef.current?.stop();
  }

  function startRecognition() {
    if (!window.isSecureContext) {
      toast.error(t("chat.voice-insecure-context"));
      return;
    }

    const SpeechRecognition = getSpeechRecognitionConstructor();
    if (!SpeechRecognition) {
      toast.error(t("chat.voice-unsupported"));
      return;
    }

    const recognition = new SpeechRecognition();
    recognition.lang = getRecognitionLanguage(i18n.language);
    recognition.continuous = true;
    recognition.interimResults = true;
    recognitionRef.current = recognition;
    baseValueRef.current = value;
    finalTranscriptRef.current = "";
    manuallyStoppingRef.current = false;

    recognition.onstart = () => {
      setListening(true);
      toast.info(t("chat.voice-started"));
      target?.current?.focus();
    };

    recognition.onresult = (event) => {
      let interimTranscript = "";
      let finalTranscript = finalTranscriptRef.current;

      for (let i = event.resultIndex; i < event.results.length; i += 1) {
        const result = event.results[i];
        const transcript = result[0]?.transcript ?? "";
        if (result.isFinal) {
          finalTranscript += transcript;
        } else {
          interimTranscript += transcript;
        }
      }

      finalTranscriptRef.current = finalTranscript;
      updateInput(`${finalTranscript}${interimTranscript}`);
    };

    recognition.onerror = (event) => {
      manuallyStoppingRef.current = false;
      setListening(false);
      recognitionRef.current = null;
      toast.error(t(getSpeechRecognitionErrorKey(event.error)));
    };

    recognition.onend = () => {
      setListening(false);
      recognitionRef.current = null;

      if (manuallyStoppingRef.current) {
        toast.info(t("chat.voice-stopped"));
      }
    };

    try {
      recognition.start();
    } catch {
      recognitionRef.current = null;
      setListening(false);
      toast.error(t("chat.voice-error"));
    }
  }

  function handleClick() {
    if (listening) {
      stopRecognition();
      return;
    }

    startRecognition();
  }

  return (
    <ChatAction active={listening} text={t("chat.voice")} onClick={handleClick}>
      <Mic className={"w-4 h-4"} />
    </ChatAction>
  );
}
