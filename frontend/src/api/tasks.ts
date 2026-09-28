import type { ImageAsset, ProcessConfig, ProcessedAsset, SizeAnnotation } from "../types";
import { request, requestBlob } from "./client";

export type TaskStatus = "queued" | "processing" | "succeeded" | "failed";

type TaskAsset = {
  id: string;
  task_id: string;
  filename: string;
  width: number;
  height: number;
  format: string;
  size: number;
};

type TaskOutput = {
  id: string;
  task_id: string;
  asset_id: string;
  type: "standardized" | "size_chart";
  filename: string;
  width: number;
  height: number;
  format: string;
  size: number;
  download_url: string;
};

export type Task = {
  id: string;
  type: string;
  status: TaskStatus;
  error_message?: string;
  total_assets: number;
  completed_assets: number;
  assets: TaskAsset[];
  outputs: TaskOutput[];
  created_at: string;
  updated_at: string;
};

function toServerConfig(config: ProcessConfig) {
  return {
    template_id: config.templateId,
    canvas_width: config.width,
    canvas_height: config.height,
    background: config.background,
    layout_mode: "center_fit",
    keep_subject_complete: true,
    output_format: config.format,
    margin_mode: config.marginMode,
    margin: config.margin,
  };
}

export async function createTask(assets: ImageAsset[], config: ProcessConfig) {
  const body = new FormData();
  body.set("config", JSON.stringify(toServerConfig(config)));
  assets.forEach((asset) => body.append("files", asset.file, asset.name));
  return request<Task>("/api/tasks", { method: "POST", body });
}

export function getTask(taskId: string, signal?: AbortSignal) {
  return request<Task>(`/api/tasks/${encodeURIComponent(taskId)}`, { signal });
}

export async function waitForTask(taskId: string, signal?: AbortSignal, onChange?: (task: Task) => void) {
  while (!signal?.aborted) {
    const task = await getTask(taskId, signal);
    onChange?.(task);
    if (task.status === "succeeded") return task;
    if (task.status === "failed") throw new Error(task.error_message || "图片处理失败");
    await new Promise<void>((resolve, reject) => {
      const timer = window.setTimeout(resolve, 600);
      signal?.addEventListener("abort", () => {
        window.clearTimeout(timer);
        reject(new DOMException("请求已取消", "AbortError"));
      }, { once: true });
    });
  }
  throw new DOMException("请求已取消", "AbortError");
}

export async function toProcessedAsset(output: TaskOutput): Promise<ProcessedAsset> {
  const blob = await requestBlob(output.download_url);
  return {
    id: output.id,
    taskId: output.task_id,
    sourceId: output.asset_id,
    name: output.filename,
    blob,
    url: URL.createObjectURL(blob),
    downloadUrl: output.download_url,
    width: output.width,
    height: output.height,
    size: output.size,
  };
}

export async function createSizeChart(taskId: string, annotation: SizeAnnotation) {
  const output = await request<TaskOutput>(`/api/tasks/${encodeURIComponent(taskId)}/size-charts`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      asset_id: annotation.sourceId,
      width: annotation.width,
      height: annotation.height,
      depth: annotation.depth,
      unit: annotation.unit,
      template_id: annotation.templateId,
    }),
  });
  return toProcessedAsset(output);
}

export function downloadTaskArchive(taskId: string) {
  return requestBlob(`/api/tasks/${encodeURIComponent(taskId)}/download.zip`);
}

export function deleteTask(taskId: string) {
  return request<null>(`/api/tasks/${encodeURIComponent(taskId)}`, { method: "DELETE" });
}

export function deleteTaskAsset(taskId: string, assetId: string) {
  return request<null>(`/api/tasks/${encodeURIComponent(taskId)}/assets/${encodeURIComponent(assetId)}`, { method: "DELETE" });
}
