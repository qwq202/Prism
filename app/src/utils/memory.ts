const volatileKeys = new Set<string>();

export function markVolatileMemoryKey(key: string) {
  volatileKeys.add(key);
  const persistedValue = localStorage.getItem(key);
  if (persistedValue && !sessionStorage.getItem(key)) {
    sessionStorage.setItem(key, persistedValue.trim());
  }
  localStorage.removeItem(key);
}

export function setMemory(key: string, value: string) {
  const data = value.trim();
  if (volatileKeys.has(key)) {
    sessionStorage.setItem(key, data);
    localStorage.removeItem(key);
    return;
  }
  localStorage.setItem(key, data);
}

export function setBooleanMemory(key: string, value: boolean) {
  setMemory(key, String(value));
}

export function setNumberMemory(key: string, value: number) {
  setMemory(key, value.toString());
}

export function setArrayMemory(key: string, value: string[]) {
  setMemory(key, value.join(","));
}

export function getMemory(key: string, defaultValue?: string): string {
  if (volatileKeys.has(key)) {
    return (sessionStorage.getItem(key) || (defaultValue ?? "")).trim();
  }
  return (localStorage.getItem(key) || (defaultValue ?? "")).trim();
}

export function getBooleanMemory(key: string, defaultValue: boolean): boolean {
  const value = getMemory(key);
  return value ? value === "true" : defaultValue;
}

export function getNumberMemory(key: string, defaultValue: number): number {
  const value = getMemory(key);
  return value ? Number(value) : defaultValue;
}

export function getArrayMemory(key: string): string[] {
  const value = getMemory(key);
  return value ? value.split(",") : [];
}

export function forgetMemory(key: string) {
  localStorage.removeItem(key);
  sessionStorage.removeItem(key);
}

export function clearMemory() {
  localStorage.clear();
  sessionStorage.clear();
}

export function popMemory(key: string): string {
  const value = getMemory(key);
  forgetMemory(key);
  return value;
}
