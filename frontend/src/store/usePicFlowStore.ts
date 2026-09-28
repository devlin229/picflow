import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { ImageAsset, ImageTemplate, ProcessConfig, ProcessedAsset, SizeAnnotation } from "../types";

const defaultConfig: ProcessConfig = {
  templateId: "square-white",
  template: "白底正方形主图",
  ratio: "1:1",
  width: 1000,
  height: 1000,
  background: "#FFFFFF",
  format: "jpeg",
  margin: 80,
};

export const builtInTemplates: ImageTemplate[] = [
  { id: "square-white", name: "白底正方形主图", type: "main", description: "1000 × 1000 · 白底 · JPG", config: defaultConfig, builtIn: true },
  { id: "square-transparent", name: "透明底商品图", type: "main", description: "1000 × 1000 · 透明 · PNG", config: { ...defaultConfig, templateId: "square-transparent", template: "透明底商品图", background: "transparent", format: "png" }, builtIn: true },
  { id: "portrait", name: "竖版商品图", type: "main", description: "4:5 · 白底 · JPG", config: { ...defaultConfig, templateId: "portrait", template: "竖版商品图", ratio: "4:5", width: 1000, height: 1250 }, builtIn: true },
  { id: "size-standard", name: "标准尺寸图", type: "size", description: "1000 × 1000 · 蓝色标注", config: {}, builtIn: true },
];

type PicFlowState = {
  assets: ImageAsset[];
  taskId: string | null;
  outputs: ProcessedAsset[];
  config: ProcessConfig;
  annotation: SizeAnnotation | null;
  sizeChart: ProcessedAsset | null;
  templates: ImageTemplate[];
  setAssets: (assets: ImageAsset[]) => void;
  setTaskId: (taskId: string) => void;
  setOutputs: (outputs: ProcessedAsset[]) => void;
  setConfig: (config: Partial<ProcessConfig>) => void;
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
      annotation: null,
      sizeChart: null,
      templates: builtInTemplates,
      setAssets: (assets) => set({ assets, taskId: null, outputs: [], sizeChart: null }),
      setTaskId: (taskId) => set({ taskId }),
      setOutputs: (outputs) => set({ outputs }),
      setConfig: (config) => set((state) => ({ config: { ...state.config, ...config } })),
      setAnnotation: (annotation) => set({ annotation }),
      setSizeChart: (sizeChart) => set({ sizeChart }),
      addTemplate: (template) => set((state) => ({ templates: [...state.templates, template] })),
      setTemplates: (templates) => set({ templates }),
      removeTemplate: (id) => set((state) => ({ templates: state.templates.filter((item) => item.builtIn || item.id !== id) })),
      resetTask: () => set({ assets: [], taskId: null, outputs: [], annotation: null, sizeChart: null }),
    }),
    {
      name: "picflow-preferences",
      partialize: (state) => ({ config: state.config, templates: state.templates }),
    },
  ),
);
