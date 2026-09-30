export type OutputFormat = "jpeg" | "png" | "webp";

export type LayoutMode = "contain" | "cover-center" | "cover-top" | "cover-bottom";

export type ProcessConfig = {
  templateId: string;
  template: string;
  ratio: string;
  width: number;
  height: number;
  background: string;
  format: OutputFormat;
  layoutMode: LayoutMode;
  marginMode: "auto" | "fixed";
  margin: number;
  quality: number;
  replaceSimpleBackground: boolean;
  backgroundTolerance: number;
};

export type SizeTemplateConfig = {
  canvasWidth: number;
  canvasHeight: number;
  background: string;
  outputFormat: OutputFormat;
  margin: number;
  annotationColor: string;
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

export type SpecificationItem = {
  id: string;
  label: string;
  value: string;
};

export type SpecificationDisplayStyle = "none" | "simple-table";

export type SpecificationStylePreset = "classic" | "dark" | "commerce-red" | "custom";

export type SpecificationTableStyle = {
  style: "simple-table";
  borderWidth: number;
  borderColor: string;
  backgroundColor: string;
  textColor: string;
  // 兼容已隐藏的旧规格图页面，当前工作台不再展示这些配置。
  preset: SpecificationStylePreset;
  title: string;
  accentColor: string;
  opacity: number;
};

export type SpecificationTablePreset = "top-left" | "top-right" | "bottom-left" | "bottom-right" | "custom";

export type SpecificationTablePosition = {
  preset: SpecificationTablePreset;
  x: number;
  y: number;
  width: number;
};

export type SizeAnnotation = {
  sourceId: string;
  specifications: SpecificationItem[];
  tablePosition: SpecificationTablePosition;
  tableStyle: SpecificationTableStyle;
  templateId: string;
  template: string;
};

export type TaskSpecification = {
  specifications: SpecificationItem[];
  tablePosition: SpecificationTablePosition;
  tableStyle: SpecificationTableStyle;
};

export type ImageTemplate = {
  id: string;
  name: string;
  type: "main" | "size";
  description: string;
  config: Partial<ProcessConfig>;
  sizeConfig?: SizeTemplateConfig;
  builtIn?: boolean;
};
