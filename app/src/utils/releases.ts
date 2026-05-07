const GITHUB_RELEASES_API =
  "https://api.github.com/repos/qwq202/prism/releases";

export type GitHubRelease = {
  tag_name: string;
  html_url: string;
  draft: boolean;
  prerelease: boolean;
};

type ParsedVersion = {
  numbers: number[];
  prerelease: string[];
};

export function isPreviewVersion(value: string): boolean {
  return /-(preview|alpha|beta|rc)/i.test(value);
}

export async function getLatestAvailableRelease(
  currentVersion: string,
): Promise<GitHubRelease | null> {
  const response = await fetch(GITHUB_RELEASES_API, {
    headers: {
      Accept: "application/vnd.github+json",
    },
  });

  if (!response.ok) {
    return null;
  }

  const releases = (await response.json()) as GitHubRelease[];
  const includePrerelease = isPreviewVersion(currentVersion);

  return (
    releases
      .filter((release) => !release.draft)
      .filter((release) => includePrerelease || !release.prerelease)
      .filter((release) => isNewerVersion(release.tag_name, currentVersion))
      .sort((a, b) => compareVersions(b.tag_name, a.tag_name))[0] ?? null
  );
}

function isNewerVersion(candidate: string, current: string): boolean {
  return compareVersions(candidate, current) > 0;
}

function compareVersions(left: string, right: string): number {
  const leftVersion = parseVersion(left);
  const rightVersion = parseVersion(right);
  const maxLength = Math.max(
    leftVersion.numbers.length,
    rightVersion.numbers.length,
  );

  for (let index = 0; index < maxLength; index += 1) {
    const leftPart = leftVersion.numbers[index] ?? 0;
    const rightPart = rightVersion.numbers[index] ?? 0;
    if (leftPart !== rightPart) {
      return leftPart - rightPart;
    }
  }

  return comparePrerelease(leftVersion.prerelease, rightVersion.prerelease);
}

function comparePrerelease(left: string[], right: string[]): number {
  if (left.length === 0 && right.length === 0) {
    return 0;
  }
  if (left.length === 0) {
    return 1;
  }
  if (right.length === 0) {
    return -1;
  }

  const maxLength = Math.max(left.length, right.length);
  for (let index = 0; index < maxLength; index += 1) {
    const leftPart = left[index];
    const rightPart = right[index];
    if (leftPart === undefined) {
      return -1;
    }
    if (rightPart === undefined) {
      return 1;
    }

    const result = comparePrereleasePart(leftPart, rightPart);
    if (result !== 0) {
      return result;
    }
  }

  return 0;
}

function comparePrereleasePart(left: string, right: string): number {
  const leftNumber = Number.parseInt(left, 10);
  const rightNumber = Number.parseInt(right, 10);
  const leftIsNumber = /^\d+$/.test(left);
  const rightIsNumber = /^\d+$/.test(right);

  if (leftIsNumber && rightIsNumber) {
    return leftNumber - rightNumber;
  }
  if (leftIsNumber) {
    return -1;
  }
  if (rightIsNumber) {
    return 1;
  }

  return left.localeCompare(right);
}

function parseVersion(value: string): ParsedVersion {
  const normalized = value.trim().replace(/^v/i, "");
  const [core, prerelease = ""] = normalized.split("-");

  return {
    numbers: core
      .split(".")
      .map((part) => Number.parseInt(part, 10))
      .map((part) => (Number.isNaN(part) ? 0 : part)),
    prerelease: prerelease.length > 0 ? prerelease.split(/[.-]/) : [],
  };
}
