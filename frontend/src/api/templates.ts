import type { ImageTemplate, ProcessConfig, SizeTemplateConfig } from "../types";
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
  const canvasWidth = Number(config.canvas_width ?? 1000);
  const canvasHeight = Number(config.canvas_height ?? 1000);
  const knownRatio = [[1, 1], [4, 5], [3, 4], [9, 16]].find(([width, height]) => canvasWidth * height === canvasHeight * width);
  return {
    id: value.id,
    name: value.name,
    type: value.type === "size_chart" ? "size" : "main",
    description: value.description ?? "",
    builtIn: value.built_in,
    config: value.type === "main_image" ? {
      templateId: value.id,
      template: value.name,
      width: canvasWidth,
      height: canvasHeight,
      ratio: knownRatio ? `${knownRatio[0]}:${knownRatio[1]}` : "custom",
      background: String(config.background ?? "#FFFFFF"),
      format: String(config.output_format ?? "jpg") === "jpg" ? "jpeg" : String(config.output_format ?? "png") as ProcessConfig["format"],
      layoutMode: (String(config.layout_mode ?? "contain") === "center_fit" ? "contain" : String(config.layout_mode ?? "contain")) as ProcessConfig["layoutMode"],
      marginMode: String(config.margin_mode ?? "fixed") as ProcessConfig["marginMode"],
      margin: Number(config.margin ?? 80),
      quality: Number(config.output_quality ?? 90),
      replaceSimpleBackground: Boolean(config.replace_simple_background ?? false),
      backgroundTolerance: Number(config.background_tolerance ?? 12),
      aiBackground: Boolean(config.ai_background ?? false),
      aiBackgroundPrompt: String(config.ai_background_prompt ?? ""),
    } : {},
    sizeConfig: value.type === "size_chart" ? {
      canvasWidth,
      canvasHeight,
      background: String(config.background ?? "#FFFFFF"),
      outputFormat: String(config.output_format ?? "jpeg") as ProcessConfig["format"],
      margin: Number(config.margin ?? 150),
      annotationColor: String(config.annotation_color ?? "#2563EB"),
    } : undefined,
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
        layout_mode: config.layoutMode,
        keep_subject_complete: config.layoutMode === "contain",
        output_format: config.format,
        margin_mode: config.marginMode,
        margin: config.margin,
        output_quality: config.quality,
        replace_simple_background: config.replaceSimpleBackground,
        background_tolerance: config.backgroundTolerance,
        ai_background: config.aiBackground,
        ai_background_prompt: config.aiBackgroundPrompt,
      },
    }),
  });
  return fromTemplateDTO(value);
}

export function deleteTemplate(templateId: string) {
  return request<null>(`/api/templates/${encodeURIComponent(templateId)}`, { method: "DELETE" });
}

export async function createSizeTemplate(name: string, description: string, config: SizeTemplateConfig) {
  const value = await request<TemplateDTO>("/api/templates", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name,
      type: "size_chart",
      description,
      config: {
        canvas_width: config.canvasWidth,
        canvas_height: config.canvasHeight,
        background: config.background,
        output_format: config.outputFormat,
        margin: config.margin,
        annotation_color: config.annotationColor,
        annotation_style: "specification_table",
      },
    }),
  });
  return fromTemplateDTO(value);
}
