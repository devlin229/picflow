import { ImageUp, Plus, Trash2 } from "lucide-react";
import { useCallback, useState } from "react";
import { useDropzone } from "react-dropzone";
import { useNavigate } from "react-router-dom";
import { errorMessage } from "../../api/client";
import { createTask } from "../../api/tasks";
import { createTemplate } from "../../api/templates";
import { ProcessingPreview } from "../../components/ProcessingPreview";
import { Button } from "../../components/ui/Button";
import { InputField, RangeField, SelectField } from "../../components/ui/Field";
import { ImageCard } from "../../components/ui/ImageCard";
import { Modal } from "../../components/ui/Modal";
import { PageHeader } from "../../components/ui/PageHeader";
import { createImageAsset } from "../../lib/images";
import { usePicFlowStore } from "../../store/usePicFlowStore";
import type { ImageTemplate, LayoutMode, OutputFormat, SpecificationDisplayStyle, SpecificationTablePreset, SpecificationTableStyle, TaskSpecification } from "../../types";

const specificationPresetLabels: Record<Exclude<SpecificationTablePreset, "custom">, string> = {
  "top-left": "左上",
  "top-right": "右上",
  "bottom-left": "左下",
  "bottom-right": "右下",
};

export function WorkspacePage() {
  const navigate = useNavigate();
  const assets = usePicFlowStore((state) => state.assets);
  const config = usePicFlowStore((state) => state.config);
  const templates = usePicFlowStore((state) => state.templates);
  const specificationDraft = usePicFlowStore((state) => state.specificationDraft);
  const specificationStyle = usePicFlowStore((state) => state.specificationStyleDraft);
  const specificationDisplayStyle = usePicFlowStore((state) => state.specificationDisplayStyle);
  const specificationPosition = usePicFlowStore((state) => state.specificationPositionDraft);
  const setAssets = usePicFlowStore((state) => state.setAssets);
  const setConfig = usePicFlowStore((state) => state.setConfig);
  const setTaskId = usePicFlowStore((state) => state.setTaskId);
  const setSpecificationDraft = usePicFlowStore((state) => state.setSpecificationDraft);
  const setSpecificationStyle = usePicFlowStore((state) => state.setSpecificationStyleDraft);
  const setSpecificationDisplayStyle = usePicFlowStore((state) => state.setSpecificationDisplayStyle);
  const setSpecificationPosition = usePicFlowStore((state) => state.setSpecificationPositionDraft);
  const addTemplate = usePicFlowStore((state) => state.addTemplate);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [namingTemplate, setNamingTemplate] = useState(false);
  const [templateName, setTemplateName] = useState("");
  const [previewAssetId, setPreviewAssetId] = useState("");

  const onDrop = useCallback(async (files: File[]) => {
    setLoading(true);
    setError("");
    try {
      const validFiles = files.filter((file) => ["image/jpeg", "image/png", "image/webp"].includes(file.type));
      const next = await Promise.all(validFiles.map(createImageAsset));
      setAssets([...assets, ...next]);
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, [assets, setAssets]);

  const dropzone = useDropzone({
    onDrop,
    onDropRejected: () => setError("仅支持 JPG、PNG 和 WebP 图片。"),
    accept: { "image/jpeg": [], "image/png": [], "image/webp": [] },
    multiple: true,
  });

  const applyTemplate = (template: ImageTemplate) => {
    setConfig({ ...template.config, templateId: template.id, template: template.name });
  };

  const setCustomConfig = (value: Partial<typeof config>) => {
    setConfig({ ...value, templateId: "", template: "自定义配置" });
  };

  const updateSpecification = (id: string, key: "label" | "value", value: string) => {
    setSpecificationDraft(specificationDraft.map((item) => item.id === id ? { ...item, [key]: value } : item));
  };

  const addSpecification = () => {
    if (specificationDraft.length >= 10) return;
    setSpecificationDraft([...specificationDraft, { id: crypto.randomUUID(), label: "", value: "" }]);
  };

  const removeSpecification = (id: string) => {
    if (specificationDraft.length === 1) return;
    setSpecificationDraft(specificationDraft.filter((item) => item.id !== id));
  };

  const updateSpecificationStyle = (value: Partial<SpecificationTableStyle>) => {
    setSpecificationStyle({ ...specificationStyle, ...value });
  };

  const visibleSpecifications = specificationDraft.filter((item) => item.label.trim() && item.value.trim());
  const taskSpecification: TaskSpecification | undefined = specificationDisplayStyle === "simple-table" ? {
    specifications: visibleSpecifications.map((item) => ({ ...item, label: item.label.trim(), value: item.value.trim() })),
    tablePosition: specificationPosition,
    tableStyle: specificationStyle,
  } : undefined;
  const previewAsset = assets.find((asset) => asset.id === previewAssetId) ?? assets[0];

  const setSpecificationPreset = (preset: Exclude<SpecificationTablePreset, "custom">) => {
    const edgeX = specificationPosition.width / 2 + 0.03;
    setSpecificationPosition({
      ...specificationPosition,
      preset,
      x: preset.endsWith("left") ? edgeX : 1 - edgeX,
      y: preset.startsWith("top") ? 0.22 : 0.78,
    });
  };

  const removeAsset = async (assetId: string) => {
    setError("");
    try {
      const removed = assets.find((asset) => asset.id === assetId);
      if (removed) URL.revokeObjectURL(removed.url);
      const remainingAssets = assets.filter((asset) => asset.id !== assetId);
      setAssets(remainingAssets);
      if (previewAssetId === assetId) setPreviewAssetId(remainingAssets[0]?.id ?? "");
    } catch (requestError) {
      setError(errorMessage(requestError));
    }
  };

  const clearAssets = async () => {
    setError("");
    try {
      assets.forEach((asset) => URL.revokeObjectURL(asset.url));
      setAssets([]);
      setPreviewAssetId("");
    } catch (requestError) {
      setError(errorMessage(requestError));
    }
  };

  const startProcessing = async () => {
    if (config.aiBackground && config.background === "transparent") {
      setError("当前模板使用透明底，暂不支持 AI 图片处理，请换用非透明底模板");
      return;
    }
    if (specificationDisplayStyle === "simple-table" && !visibleSpecifications.length) {
      setError("请至少填写一项完整的商品规格，或将规格展示样式改为“不添加规格”");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      const task = await createTask(assets, config, taskSpecification);
      setAssets(assets.map((asset, index) => ({ ...asset, id: task.assets[index]?.id ?? asset.id })));
      setTaskId(task.id);
      navigate("/processing");
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setSubmitting(false);
    }
  };

  const saveTemplate = async () => {
    if (submitting) return;
    const name = templateName.trim();
    if (!name) {
      setError("请输入模板名称");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      const template = await createTemplate(name, `${config.width} × ${config.height} · ${config.format.toUpperCase()}`, config);
      addTemplate(template);
      setConfig({ templateId: template.id, template: template.name });
      setNamingTemplate(false);
      setTemplateName("");
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setSubmitting(false);
    }
  };

  if (!assets.length) {
    return (
      <section className="page">
        <PageHeader title="新建图片处理任务" description="上传商品图片并选择处理方式，快速得到统一规格的输出。" />
        <div className="upload-layout">
          <div {...dropzone.getRootProps({ className: `upload-zone ${dropzone.isDragActive ? "upload-zone--active" : ""}` })}>
            <input {...dropzone.getInputProps()} />
            <span className="upload-icon"><ImageUp size={44} /></span>
            <h2>{dropzone.isDragActive ? "松开即可添加图片" : "拖拽商品图片到这里"}</h2>
            <p>支持 JPG、PNG、WebP，可一次上传多张图片</p>
            {error && <p className="form-error" role="alert">{error}</p>}
            <Button type="button" disabled={loading}>{loading ? "读取中…" : "选择文件"}</Button>
          </div>
          <aside className="panel quick-templates">
            <h3>常用模板</h3>
            {templates.filter((item) => item.type === "main").slice(0, 4).map((template) => (
              <button key={template.id} type="button" onClick={() => applyTemplate(template)}>
                <strong>{template.name}</strong><span>{template.description}</span>
              </button>
            ))}
          </aside>
        </div>
      </section>
    );
  }

  return (
    <section className="page workspace-configure-page">
      <div className="configure-layout">
        <section className="panel asset-panel">
          <div className="panel-title-row">
            <h3>已上传图片（{assets.length}）</h3>
            <div className="panel-title-actions">
              <div {...dropzone.getRootProps()}><input {...dropzone.getInputProps()} /><Button type="button" variant="secondary" disabled={loading}><Plus size={15} />继续添加</Button></div>
              <Button variant="ghost" onClick={() => void clearAssets()}><Trash2 size={15} />清空</Button>
            </div>
          </div>
          <div className="asset-grid">
            {assets.map((asset) => <ImageCard key={asset.id} src={asset.url} name={asset.name} meta={`${asset.width} × ${asset.height} · ${asset.format}`} selected={asset.id === previewAsset.id} onClick={() => setPreviewAssetId(asset.id)} onRemove={() => void removeAsset(asset.id)} />)}
          </div>
          <div className="processing-preview">
            <div className="panel-title-row"><h3>预览</h3><span>{config.aiBackground ? "原图构图 · " : ""}输出 {config.width} × {config.height}</span></div>
            <div className="processing-preview__stage"><ProcessingPreview asset={previewAsset} config={config} specification={taskSpecification} onSpecificationPositionChange={setSpecificationPosition} /></div>
          </div>
        </section>
        <aside className="panel settings-panel">
          <h3>输出设置</h3>
          <SelectField label="处理模板" value={config.templateId || "custom"} onChange={(event) => {
            const template = templates.find((item) => item.id === event.target.value);
            if (template) applyTemplate(template);
          }}>
            <option value="custom">自定义配置</option>
            {templates.filter((item) => item.type === "main").map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </SelectField>
          <SelectField label="输出比例" value={config.ratio} onChange={(event) => {
            const ratio = event.target.value;
            if (ratio === "custom") {
              setCustomConfig({ ratio });
              return;
            }
            const [w, h] = ratio.split(":").map(Number);
            setCustomConfig({ ratio, height: Math.round(config.width * h / w) });
          }}>
            {["1:1", "4:5", "3:4", "9:16", "custom"].map((value) => <option key={value} value={value}>{value === "custom" ? "自定义" : value}</option>)}
          </SelectField>
          <SelectField label="常用尺寸" value={[800, 1000, 1200].includes(config.width) && config.width === config.height ? `${config.width}` : "custom"} onChange={(event) => {
            if (event.target.value === "custom") return;
            const size = Number(event.target.value);
            setCustomConfig({ width: size, height: size, ratio: "1:1" });
          }}>
            <option value="800">800 × 800</option><option value="1000">1000 × 1000</option><option value="1200">1200 × 1200</option><option value="custom">自定义尺寸</option>
          </SelectField>
          <InputField label="输出宽度" type="number" min={64} max={4096} value={config.width} onChange={(event) => setCustomConfig({ width: Number(event.target.value), ratio: "custom" })} />
          <InputField label="输出高度" type="number" min={64} max={4096} value={config.height} onChange={(event) => setCustomConfig({ height: Number(event.target.value), ratio: "custom" })} />
          <SelectField label="缩放与裁剪" value={config.layoutMode} onChange={(event) => setCustomConfig({ layoutMode: event.target.value as LayoutMode })}>
            <option value="contain">保持完整并留白</option>
            <option value="cover-center">填满画布 · 居中裁剪</option>
            <option value="cover-top">填满画布 · 顶部对齐</option>
            <option value="cover-bottom">填满画布 · 底部对齐</option>
          </SelectField>
          {config.layoutMode === "contain" && <>
            <SelectField label="边距" value={config.marginMode} onChange={(event) => setCustomConfig({ marginMode: event.target.value as "auto" | "fixed" })}>
              <option value="auto">自动边距</option><option value="fixed">固定像素</option>
            </SelectField>
            {config.marginMode === "fixed" && <InputField label="边距像素" type="number" min={0} max={Math.floor(Math.min(config.width, config.height) / 2) - 1} value={config.margin} onChange={(event) => setCustomConfig({ margin: Number(event.target.value) })} />}
          </>}
          <SelectField label="输出格式" value={config.format} onChange={(event) => {
            const format = event.target.value as OutputFormat;
            setCustomConfig({ format, background: format === "jpeg" && config.background === "transparent" ? "#FFFFFF" : config.background });
          }}>
            <option value="jpeg">JPG</option><option value="png">PNG</option><option value="webp">WebP</option>
          </SelectField>
          {config.format === "jpeg" && <RangeField label={`JPG 压缩质量（${config.quality}%）`} min={30} max={100} value={config.quality} onValueChange={(quality) => setCustomConfig({ quality })} />}
          <div className="settings-section-heading">
            <div><strong>AI 图片处理</strong></div>
          </div>
          <InputField label="提示词（可选）" maxLength={1000} placeholder="输入提示词让 AI 处理图片，留空不使用 AI" value={config.aiBackgroundPrompt} onChange={(event) => setCustomConfig({ aiBackgroundPrompt: event.target.value, aiBackground: Boolean(event.target.value.trim()) })} />
          {config.aiBackground && <small className="field-note">图片将发送至模型服务并按张计费，实际效果处理后查看。当前不支持透明底。</small>}
          <SelectField label="规格展示样式" value={specificationDisplayStyle} onChange={(event) => setSpecificationDisplayStyle(event.target.value as SpecificationDisplayStyle)}>
            <option value="none">不添加规格</option>
            <option value="simple-table">参数表格</option>
          </SelectField>
          {specificationDisplayStyle === "simple-table" && <>
          <div className="specification-editor">
            <div className="specification-editor__heading"><span>参数名称</span><span>参数值（可含单位）</span><i /></div>
            {specificationDraft.map((item) => (
              <div className="specification-editor__row" key={item.id}>
                <input aria-label="参数名称" maxLength={20} placeholder="例如：重量" value={item.label} onChange={(event) => updateSpecification(item.id, "label", event.target.value)} />
                <input aria-label="参数值" maxLength={40} placeholder="例如：2.5 kg" value={item.value} onChange={(event) => updateSpecification(item.id, "value", event.target.value)} />
                <button type="button" aria-label={`删除${item.label || "空白"}参数`} disabled={specificationDraft.length === 1} onClick={() => removeSpecification(item.id)}><Trash2 size={15} /></button>
              </div>
            ))}
            <Button type="button" variant="secondary" disabled={specificationDraft.length >= 10} onClick={addSpecification}><Plus size={15} />添加参数</Button>
          </div>
          <div className="field"><span>表格预设位置</span><div className="position-presets">
            {(Object.keys(specificationPresetLabels) as Array<Exclude<SpecificationTablePreset, "custom">>).map((preset) => <button className={specificationPosition.preset === preset ? "active" : ""} type="button" key={preset} onClick={() => setSpecificationPreset(preset)}>{specificationPresetLabels[preset]}</button>)}
          </div></div>
          <div className="specification-style-editor">
            <RangeField label={`表框粗细（${specificationStyle.borderWidth}px）`} min={1} max={8} value={specificationStyle.borderWidth} onValueChange={(borderWidth) => updateSpecificationStyle({ borderWidth })} />
            <div className="field-row specification-style-colors">
              <InputField label="表框颜色" type="color" value={specificationStyle.borderColor} onChange={(event) => updateSpecificationStyle({ borderColor: event.target.value.toUpperCase() })} />
              <InputField label="文字色" type="color" value={specificationStyle.textColor} onChange={(event) => updateSpecificationStyle({ textColor: event.target.value.toUpperCase() })} />
            </div>
            <SelectField label="背景颜色" value={specificationStyle.backgroundColor === "transparent" ? "transparent" : "custom"} onChange={(event) => updateSpecificationStyle({ backgroundColor: event.target.value === "transparent" ? "transparent" : "#FFFFFF" })}>
              <option value="transparent">透明</option>
              <option value="custom">自定义纯色</option>
            </SelectField>
            {specificationStyle.backgroundColor !== "transparent" && <InputField label="自定义背景色" type="color" value={specificationStyle.backgroundColor} onChange={(event) => updateSpecificationStyle({ backgroundColor: event.target.value.toUpperCase() })} />}
          </div>
          </>}
          <Modal open={namingTemplate} title="保存为模板" busy={submitting} onClose={() => { setNamingTemplate(false); setTemplateName(""); setError(""); }}>
          <form noValidate onSubmit={(event) => { event.preventDefault(); void saveTemplate(); }}>
            <InputField label="模板名称" placeholder="请输入模板名称" maxLength={80} autoFocus disabled={submitting} value={templateName} onChange={(event) => setTemplateName(event.target.value)} />
            {error && <p className="form-error" role="alert">{error}</p>}
            <div className="settings-actions">
              <Button type="button" variant="secondary" disabled={submitting} onClick={() => { setNamingTemplate(false); setTemplateName(""); setError(""); }}>取消</Button>
              <Button type="submit" disabled={submitting}>{submitting ? "保存中…" : "保存模板"}</Button>
            </div>
          </form>
          </Modal>
          <div className="settings-actions">
            <Button variant="secondary" disabled={submitting} onClick={() => { setNamingTemplate(true); setError(""); }}>保存为模板</Button>
            <Button disabled={submitting} onClick={startProcessing}>{submitting ? "提交中…" : "开始处理"}</Button>
          </div>
          {error && !namingTemplate && <p className="form-error" role="alert">{error}</p>}
        </aside>
      </div>
    </section>
  );
}
