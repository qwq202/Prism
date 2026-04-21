import axios from "axios";
import { blobEndpoint } from "@/conf/env.ts";
import { trimSuffixes } from "@/utils/base.ts";

export type BlobParserResponse = {
  status: boolean;
  content: string;
  error?: string;
};

export type FileObject = {
  name: string;
  content: string;
  size?: number;
};

type Model = {
  id: string;
  channel_type?: string;
  ocr_model?: boolean;
  vision_model?: boolean;
  reverse_model?: boolean;
};

export type FileArray = FileObject[];

const GROK_IMAGE_MIME_TYPES = new Set(["image/jpeg", "image/jpg", "image/png"]);
const GROK_IMAGE_EXTENSIONS = new Set(["jpg", "jpeg", "png"]);
const LOCAL_ATTACHMENT_HOSTS = new Set(["localhost", "127.0.0.1", "::1"]);
const BLOCKED_ATTACHMENT_HOST_SUFFIXES = ["r2.cloudflarestorage.com"];

function getFileExtension(filename: string): string {
  const segments = filename.toLowerCase().split(".");
  return segments.length > 1 ? segments.at(-1) || "" : "";
}

function isXAIChannelModel(model: Model): boolean {
  return (model.channel_type || "").toLowerCase() === "xai";
}

function isGrokCompatibleImage(file: File): boolean {
  const mimeType = file.type.toLowerCase();
  if (GROK_IMAGE_MIME_TYPES.has(mimeType)) {
    return true;
  }
  return GROK_IMAGE_EXTENSIONS.has(getFileExtension(file.name));
}

function normalizeAttachmentUrl(url: string): string {
  if (!url) {
    return "";
  }

  try {
    const baseUrl =
      typeof axios.defaults.baseURL === "string" && axios.defaults.baseURL.length > 0
        ? axios.defaults.baseURL
        : window.location.origin;
    const resolved = new URL(url, baseUrl);
    if (LOCAL_ATTACHMENT_HOSTS.has(resolved.hostname)) {
      return "";
    }
    if (
      BLOCKED_ATTACHMENT_HOST_SUFFIXES.some(
        (suffix) =>
          resolved.hostname === suffix || resolved.hostname.endsWith(`.${suffix}`),
      )
    ) {
      console.warn(
        `[parser] attachment url "${resolved.hostname}" looks like an object storage api endpoint, fallback to base64`,
      );
      return "";
    }
    return resolved.toString();
  } catch {
    return "";
  }
}

async function decodeImageFile(file: File): Promise<HTMLImageElement> {
  const objectUrl = URL.createObjectURL(file);

  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => {
      URL.revokeObjectURL(objectUrl);
      resolve(image);
    };
    image.onerror = () => {
      URL.revokeObjectURL(objectUrl);
      reject(new Error("Failed to decode image"));
    };
    image.src = objectUrl;
  });
}

async function convertImageFileToPng(file: File): Promise<File> {
  const image = await decodeImageFile(file);
  const width = image.naturalWidth || image.width;
  const height = image.naturalHeight || image.height;
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;

  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Failed to create canvas context");
  }
  context.drawImage(image, 0, 0, width, height);

  const pngBlob = await new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (blob) {
        resolve(blob);
      } else {
        reject(new Error("Failed to encode png"));
      }
    }, "image/png");
  });

  const nextName = /\.[^.]+$/.test(file.name)
    ? file.name.replace(/\.[^.]+$/, ".png")
    : `${file.name}.png`;

  return new File([pngBlob], nextName, {
    type: "image/png",
    lastModified: file.lastModified,
  });
}

async function ensureGrokCompatibleImage(file: File, model: Model): Promise<File> {
  if (!isXAIChannelModel(model)) {
    return file;
  }

  try {
    if (isGrokCompatibleImage(file)) {
      console.log(
        `[parser] xai image upload detected "${file.type || "unknown"}", normalizing to image/png for stable decoding`,
      );
    } else {
      console.log(
        `[parser] xai image upload received unsupported image type "${file.type || "unknown"}", converting to image/png`,
      );
    }
    return await convertImageFileToPng(file);
  } catch (error) {
    console.warn(
      "[parser] failed to normalize image for xai compatibility, fallback to original file:",
      error,
    );
    return file;
  }
}

export async function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(new Error("Failed to read file"));
  });
}

export function checkFileSuffix(
  filename: string,
  suffixes: string | string[],
): boolean {
  filename = filename.toLowerCase();

  if (typeof suffixes === "string") {
    return filename.endsWith(suffixes);
  }

  return suffixes.some((suffix) => filename.endsWith(suffix));
}

export async function quickBlobParser(
  file: File,
  model: Model,
  onProgress?: (progress: number) => void,
): Promise<string> {
  // this function is used to parse the file quickly in local
  // otherwise, it will be parsed as a file

  if (file.size === 0 || file.name.length === 0) {
    throw new Error("File is empty");
  }

  if (!model.reverse_model) {
    try {
      // if the file is an image, it will be parsed as an image by local parser first
      const couldLocalVision = model.vision_model;
      if (couldLocalVision && file.type.startsWith("image/")) {
        console.log("[parser] hit image/* file, using local parser");
        const imageFile = await ensureGrokCompatibleImage(file, model);
        if (isXAIChannelModel(model)) {
          const attachmentUrl = await uploadXAIImage(imageFile);
          if (attachmentUrl) {
            return attachmentUrl;
          }
        }
        // parse image as base64 (e.g. result: data:image/png;base64,xxx)
        const base64 = await fileToBase64(imageFile);
        return base64;
      }

      // if the file is txt, parse it as txt
      if (
        file.type === "text/plain" ||
        checkFileSuffix(file.name, [
          "txt",
          "md",
          "markdown",
          "json",
          "xml",
          "csv",
          "yaml",
          "yml",
          "toml",
          "ini",
          "cfg",
          "conf",
        ])
      ) {
        console.log("[parser] hit text/plain file, using local parser");
        return await file.text();
      }
      console.log(file.type);
    } catch (e) {
      console.error(
        "[parser] local parser failed, switch to server parser: ",
        e,
      );
    }
  }

  return blobParser(file, model, onProgress);
}

async function uploadXAIImage(file: File): Promise<string> {
  try {
    const formData = new FormData();
    formData.append("file", file);

    const response = await axios.post("/attachment/upload", formData, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
    });

    const data = response.data as BlobParserResponse & { url?: string };
    if (!data.status || !data.url) {
      return "";
    }

    return normalizeAttachmentUrl(data.url);
  } catch (error) {
    console.warn("[parser] failed to upload xai image attachment, fallback to base64:", error);
    return "";
  }
}

export async function blobParser(
  file: File,
  model: Model,
  onProgress?: (progress: number) => void,
): Promise<string> {
  const endpoint = trimSuffixes(blobEndpoint, ["/upload", "/"]);

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    const formData = new FormData();
    formData.append("file", file);
    formData.append("model", model.id);
    formData.append("enable_ocr", (model.ocr_model ?? false).toString());
    formData.append("enable_vision", (model.vision_model ?? false).toString());
    formData.append("save_all", (model.reverse_model ?? false).toString());
    xhr.open("POST", `${endpoint}/upload`, true);
    xhr.upload.onprogress = (progressEvent) => {
      console.debug(progressEvent);
      if (progressEvent.lengthComputable) {
        const percentCompleted = Math.round(
          (progressEvent.loaded * 100) / progressEvent.total,
        );
        console.debug(percentCompleted);
        onProgress?.(percentCompleted);
      }
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const data = JSON.parse(xhr.responseText) as BlobParserResponse;
          if (!data.status) {
            reject(new Error(data.error));
          } else if (data.content.length === 0) {
            reject(new Error("Result is empty"));
          } else {
            resolve(data.content);
          }
        } catch (e) {
          reject(new Error("Invalid JSON response"));
        }
      } else {
        reject(new Error(`HTTP error! status: ${xhr.status}`));
      }
    };
    xhr.onerror = () => {
      reject(new Error("Network error"));
    };
    xhr.send(formData);
  });
}
