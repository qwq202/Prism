import { EventCommitter } from "@/events/struct.ts";

export const blobEvent = new EventCommitter<File | File[]>({
  name: "blob",
});

export const filePanelEvent = new EventCommitter<undefined>({
  name: "file-panel",
});
