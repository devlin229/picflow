import type { ImageTemplate, ProcessConfig } from "../types";
import { request } from "./client";

type TemplateDTO = {
  id: string;
  name: string;
  type: "main_image" | "size_chart";
  description?: string;
  config: Record<string, unknown>;
  built_in: boolean;
};

function fromTemplateDTO(value: TemplateDTO): ImageTemplate {
  const config = value.config;
  return {
    id: value.id,
    name: value.name,
    type: value.type === "size_chart" ? "size" : "main",
    description: value.description ?? "",
    builtIn: value.built_in,
    config: value.type === "main_image" ? {
      templateId: value.id,
      template: value.name,
      width: Number(config.canvas_width ?? 1000),
      height: Number(config.canvas_height ?? 1000),
      ratio: `${Number(config.canvas_width ?? 1000)}:${Number(config.canvas_height ?? 1000)}`,
      background: String(config.background ?? "#FFFFFF"),
      format: String(config.output_format ?? "jpg") === "jpg" ? "jpeg" : String(config.output_format ?? "png") as ProcessConfig["format"],
      margin: Number(config.margin ?? 80),
    } : {},
  };
}

export async function listTemplates() {
  const result = await request<{ items: TemplateDTO[] }>("/api/templates");
  return result.items.map(fromTemplateDTO);
}

export async function createTemplate(name: string, description: string, config: ProcessConfig) {
  const value = await request<TemplateDTO>("/api/templates", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name,
      type: "main_image",
      description,
      config: {
        canvas_width: config.width,
        canvas_height: config.height,
        background: config.background,
        layout_mode: "center_fit",
        keep_subject_complete: true,
        output_format: config.format,
        margin: config.margin,
      },
    }),
  });
  return fromTemplateDTO(value);
}

export function deleteTemplate(templateId: string) {
  return request<null>(`/api/templates/${encodeURIComponent(templateId)}`, { method: "DELETE" });
}
