export const popupTypes = {
  Text: "text",
  Password: "password",
  Number: "number",
  Switch: "switch",
  Clock: "clock",
  List: "list",
  MultiList: "multi-list",
  Empty: "empty",
} as const;

export type PopupType = (typeof popupTypes)[keyof typeof popupTypes];
