import axios from "axios";

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
        `[parser] xai image upload detected compatible type "${file.type || "unknown"}", keeping original bytes`,
      );
      return file;
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
  onProgress?.(0);
  if (file.size === 0 || file.name.length === 0) {
    throw new Error("File is empty");
  }

  if (!file.type.startsWith("image/")) {
    throw new Error("Only image uploads are supported");
  }

  if (!model.vision_model) {
    throw new Error("The current model does not support image recognition");
  }

  try {
    console.log("[parser] hit image/* file, using local parser");
    const imageFile = await ensureGrokCompatibleImage(file, model);
    onProgress?.(40);
    const attachmentUrl = await uploadImageAttachment(imageFile);
    if (attachmentUrl) {
      onProgress?.(100);
      return attachmentUrl;
    }
    throw new Error("Failed to upload image attachment");
  } catch (e) {
    console.error("[parser] local image parser failed:", e);
    throw e instanceof Error ? e : new Error("Failed to process image");
  }
}

async function uploadImageAttachment(file: File): Promise<string> {
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

    return data.url;
  } catch (error) {
    console.warn("[parser] failed to upload image attachment:", error);
    return "";
  }
}
