export type OutputFormat = "jpeg" | "png" | "webp";

export type ProcessConfig = {
  templateId: string;
  template: string;
  ratio: string;
  width: number;
  height: number;
  background: string;
  format: OutputFormat;
  margin: number;
};

export type ImageAsset = {
  id: string;
  file: File;
  name: string;
  url: string;
  width: number;
  height: number;
  format: string;
};

export type ProcessedAsset = {
  id: string;
  taskId: string;
  sourceId: string;
  name: string;
  blob: Blob;
  url: string;
  downloadUrl: string;
  width: number;
  height: number;
  size: number;
};

export type SizeAnnotation = {
  sourceId: string;
  width: number;
  height: number;
  depth?: number;
  unit: "cm" | "mm";
  templateId: string;
  template: string;
};

export type ImageTemplate = {
  id: string;
  name: string;
  type: "main" | "size";
  description: string;
  config: Partial<ProcessConfig>;
  builtIn?: boolean;
};
