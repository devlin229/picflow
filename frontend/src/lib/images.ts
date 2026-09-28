import type { ImageAsset } from "../types";

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = reject;
    image.src = src;
  });
}

export async function createImageAsset(file: File): Promise<ImageAsset> {
  const url = URL.createObjectURL(file);
  let image: HTMLImageElement;
  try {
    image = await loadImage(url);
  } catch (error) {
    URL.revokeObjectURL(url);
    throw error;
  }
  return {
    id: crypto.randomUUID(),
    file,
    name: file.name,
    url,
    width: image.naturalWidth,
    height: image.naturalHeight,
    format: file.type.split("/")[1]?.toUpperCase() || "IMAGE",
  };
}

export function downloadBlob(blob: Blob, name: string) {
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = name;
  link.click();
  setTimeout(() => URL.revokeObjectURL(link.href), 1000);
}

export function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
