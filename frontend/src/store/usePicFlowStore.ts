import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { ImageAsset, ImageTemplate, ProcessConfig, ProcessedAsset, SizeAnnotation, SpecificationDisplayStyle, SpecificationItem, SpecificationStylePreset, SpecificationTablePosition, SpecificationTableStyle } from "../types";

const defaultConfig: ProcessConfig = {
  templateId: "square-white",
  template: "白底正方形主图",
  ratio: "1:1",
  width: 1000,
  height: 1000,
  background: "#FFFFFF",
  format: "jpeg",
  layoutMode: "contain",
  marginMode: "auto",
  margin: 80,
  quality: 90,
  replaceSimpleBackground: false,
  backgroundTolerance: 12,
};

export const builtInTemplates: ImageTemplate[] = [
  { id: "square-white", name: "白底正方形主图", type: "main", description: "1000 × 1000 · 白底 · JPG", config: defaultConfig, builtIn: true },
  { id: "square-transparent", name: "透明底商品图", type: "main", description: "1000 × 1000 · 单一背景移除 · PNG", config: { ...defaultConfig, templateId: "square-transparent", template: "透明底商品图", background: "transparent", format: "png", replaceSimpleBackground: true }, builtIn: true },
  { id: "portrait", name: "竖版商品图", type: "main", description: "4:5 · 白底 · JPG", config: { ...defaultConfig, templateId: "portrait", template: "竖版商品图", ratio: "4:5", width: 1000, height: 1250 }, builtIn: true },
  { id: "size-standard", name: "标准规格图", type: "size", description: "1000 × 1000 · 商品规格表格", config: {}, sizeConfig: { canvasWidth: 1000, canvasHeight: 1000, background: "#FFFFFF", outputFormat: "jpeg", margin: 80, annotationColor: "#2563EB" }, builtIn: true },
];

export const defaultSpecificationDraft: SpecificationItem[] = [
  { id: "width", label: "宽度", value: "42 cm" },
  { id: "height", label: "高度", value: "58 cm" },
  { id: "depth", label: "深度", value: "46 cm" },
  { id: "weight", label: "重量", value: "" },
];

export const defaultSpecificationStyle: SpecificationTableStyle = {
  style: "simple-table",
  borderWidth: 2,
  borderColor: "#DC2626",
  backgroundColor: "transparent",
  textColor: "#3F1D1D",
  preset: "custom",
  title: "",
  accentColor: "#DC2626",
  opacity: 0,
};
// 兼容暂时隐藏的旧规格图页面；工作台仅展示当前的参数表格样式。
export const specificationStylePresets: Record<Exclude<SpecificationStylePreset, "custom">, SpecificationTableStyle> = {
  classic: { ...defaultSpecificationStyle, preset: "classic", borderColor: "#2563EB", accentColor: "#2563EB", textColor: "#1F2937" },
  dark: { ...defaultSpecificationStyle, preset: "dark", borderColor: "#111827", accentColor: "#111827", textColor: "#111827" },
  "commerce-red": { ...defaultSpecificationStyle, preset: "commerce-red", borderColor: "#DC2626", accentColor: "#DC2626", textColor: "#3F1D1D" },
};
export const defaultSpecificationPosition: SpecificationTablePosition = { preset: "top-right", x: 0.78, y: 0.22, width: 0.42 };

type PicFlowState = {
  assets: ImageAsset[];
  taskId: string | null;
  outputs: ProcessedAsset[];
  config: ProcessConfig;
  specificationDraft: SpecificationItem[];
  specificationStyleDraft: SpecificationTableStyle;
  specificationDisplayStyle: SpecificationDisplayStyle;
  specificationPositionDraft: SpecificationTablePosition;
  annotation: SizeAnnotation | null;
  sizeChart: ProcessedAsset | null;
  templates: ImageTemplate[];
  setAssets: (assets: ImageAsset[]) => void;
  setTaskId: (taskId: string | null) => void;
  setOutputs: (outputs: ProcessedAsset[]) => void;
  setConfig: (config: Partial<ProcessConfig>) => void;
  setSpecificationDraft: (items: SpecificationItem[]) => void;
  setSpecificationStyleDraft: (style: SpecificationTableStyle) => void;
  setSpecificationDisplayStyle: (style: SpecificationDisplayStyle) => void;
  setSpecificationPositionDraft: (position: SpecificationTablePosition) => void;
  setAnnotation: (annotation: SizeAnnotation) => void;
  setSizeChart: (asset: ProcessedAsset) => void;
  addTemplate: (template: ImageTemplate) => void;
  setTemplates: (templates: ImageTemplate[]) => void;
  removeTemplate: (id: string) => void;
  resetTask: () => void;
};

export const usePicFlowStore = create<PicFlowState>()(
  persist(
    (set) => ({
      assets: [],
      taskId: null,
      outputs: [],
      config: defaultConfig,
      specificationDraft: defaultSpecificationDraft,
      specificationStyleDraft: defaultSpecificationStyle,
      specificationDisplayStyle: "none",
      specificationPositionDraft: defaultSpecificationPosition,
      annotation: null,
      sizeChart: null,
      templates: builtInTemplates,
      setAssets: (assets) => set((state) => {
        state.outputs.forEach((item) => URL.revokeObjectURL(item.url));
        if (state.sizeChart) URL.revokeObjectURL(state.sizeChart.url);
        return { assets, taskId: null, outputs: [], annotation: null, sizeChart: null };
      }),
      setTaskId: (taskId) => set({ taskId }),
      setOutputs: (outputs) => set((state) => {
        state.outputs.forEach((item) => URL.revokeObjectURL(item.url));
        return { outputs };
      }),
      setConfig: (config) => set((state) => ({ config: { ...state.config, ...config } })),
      setSpecificationDraft: (specificationDraft) => set({ specificationDraft }),
      setSpecificationStyleDraft: (specificationStyleDraft) => set({ specificationStyleDraft }),
      setSpecificationDisplayStyle: (specificationDisplayStyle) => set({ specificationDisplayStyle }),
      setSpecificationPositionDraft: (specificationPositionDraft) => set({ specificationPositionDraft }),
      setAnnotation: (annotation) => set({ annotation }),
      setSizeChart: (sizeChart) => set((state) => {
        if (state.sizeChart) URL.revokeObjectURL(state.sizeChart.url);
        return { sizeChart };
      }),
      addTemplate: (template) => set((state) => ({ templates: [...state.templates, template] })),
      setTemplates: (templates) => set({ templates }),
      removeTemplate: (id) => set((state) => ({ templates: state.templates.filter((item) => item.builtIn || item.id !== id) })),
      resetTask: () => set((state) => {
        state.assets.forEach((item) => URL.revokeObjectURL(item.url));
        state.outputs.forEach((item) => URL.revokeObjectURL(item.url));
        if (state.sizeChart) URL.revokeObjectURL(state.sizeChart.url);
        return { assets: [], taskId: null, outputs: [], annotation: null, sizeChart: null };
      }),
    }),
    {
      name: "picflow-preferences",
      partialize: (state) => ({
        config: state.config,
        specificationDraft: state.specificationDraft,
        specificationStyleDraft: state.specificationStyleDraft,
        specificationDisplayStyle: state.specificationDisplayStyle,
        specificationPositionDraft: state.specificationPositionDraft,
        templates: state.templates,
        taskId: state.taskId,
      }),
      merge: (persisted, current) => {
        const saved = persisted as Partial<PicFlowState>;
        return {
          ...current,
          ...saved,
          config: { ...current.config, ...saved.config },
          specificationDraft: saved.specificationDraft ?? current.specificationDraft,
          specificationStyleDraft: saved.specificationStyleDraft && "borderWidth" in saved.specificationStyleDraft
            ? { ...current.specificationStyleDraft, ...saved.specificationStyleDraft }
            : current.specificationStyleDraft,
          specificationDisplayStyle: saved.specificationDisplayStyle
            ?? ((saved as Partial<PicFlowState> & { includeSpecification?: boolean }).includeSpecification ? "simple-table" : "none"),
          specificationPositionDraft: { ...current.specificationPositionDraft, ...saved.specificationPositionDraft },
          templates: saved.templates ?? current.templates,
        };
      },
    },
  ),
);
