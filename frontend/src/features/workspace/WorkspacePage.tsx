import { ImageUp, Plus, Trash2 } from "lucide-react";
import { useCallback, useState } from "react";
import { useDropzone } from "react-dropzone";
import { useNavigate } from "react-router-dom";
import { errorMessage } from "../../api/client";
import { createTask, deleteTask } from "../../api/tasks";
import { createTemplate } from "../../api/templates";
import { Button } from "../../components/ui/Button";
import { InputField, SelectField } from "../../components/ui/Field";
import { ImageCard } from "../../components/ui/ImageCard";
import { PageHeader } from "../../components/ui/PageHeader";
import { createImageAsset } from "../../lib/images";
import { usePicFlowStore } from "../../store/usePicFlowStore";
import type { ImageTemplate, OutputFormat } from "../../types";

export function WorkspacePage() {
  const navigate = useNavigate();
  const assets = usePicFlowStore((state) => state.assets);
  const config = usePicFlowStore((state) => state.config);
  const templates = usePicFlowStore((state) => state.templates);
  const taskId = usePicFlowStore((state) => state.taskId);
  const setAssets = usePicFlowStore((state) => state.setAssets);
  const setConfig = usePicFlowStore((state) => state.setConfig);
  const setTaskId = usePicFlowStore((state) => state.setTaskId);
  const addTemplate = usePicFlowStore((state) => state.addTemplate);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const onDrop = useCallback(async (files: File[]) => {
    setLoading(true);
    setError("");
    try {
      if (taskId) await deleteTask(taskId);
      const validFiles = files.filter((file) => ["image/jpeg", "image/png", "image/webp"].includes(file.type));
      const next = await Promise.all(validFiles.map(createImageAsset));
      setAssets([...assets, ...next]);
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, [assets, setAssets, taskId]);

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

  const removeAsset = async (assetId: string) => {
    setError("");
    try {
      if (taskId) await deleteTask(taskId);
      const removed = assets.find((asset) => asset.id === assetId);
      if (removed) URL.revokeObjectURL(removed.url);
      setAssets(assets.filter((asset) => asset.id !== assetId));
    } catch (requestError) {
      setError(errorMessage(requestError));
    }
  };

  const clearAssets = async () => {
    setError("");
    try {
      if (taskId) await deleteTask(taskId);
      assets.forEach((asset) => URL.revokeObjectURL(asset.url));
      setAssets([]);
    } catch (requestError) {
      setError(errorMessage(requestError));
    }
  };

  const startProcessing = async () => {
    setSubmitting(true);
    setError("");
    try {
      const task = await createTask(assets, config);
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
    setSubmitting(true);
    setError("");
    try {
      const name = `${config.width}×${config.height} 自定义模板`;
      const template = await createTemplate(name, `${config.width} × ${config.height} · ${config.format.toUpperCase()}`, config);
      addTemplate(template);
      setConfig({ templateId: template.id, template: template.name });
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
    <section className="page">
      <PageHeader title="配置处理规则" description={`已上传 ${assets.length} 张商品图，本批任务将使用同一套输出规则。`} />
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
            {assets.map((asset) => <ImageCard key={asset.id} src={asset.url} name={asset.name} meta={`${asset.width} × ${asset.height} · ${asset.format}`} onRemove={() => void removeAsset(asset.id)} />)}
          </div>
          <div className="info-strip">默认等比缩放并补边，保持商品主体完整、不拉伸。</div>
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
          <SelectField label="背景" value={config.background === "transparent" ? "transparent" : config.background.toUpperCase() === "#FFFFFF" ? "white" : "custom"} onChange={(event) => {
            if (event.target.value === "white") setCustomConfig({ background: "#FFFFFF" });
            if (event.target.value === "transparent") setCustomConfig({ background: "transparent", format: config.format === "jpeg" ? "png" : config.format });
            if (event.target.value === "custom" && (config.background === "transparent" || config.background.toUpperCase() === "#FFFFFF")) setCustomConfig({ background: "#F3F4F6" });
          }}>
            <option value="white">白底</option><option value="transparent">透明底</option><option value="custom">自定义纯色</option>
          </SelectField>
          {config.background !== "transparent" && config.background.toUpperCase() !== "#FFFFFF" && <InputField label="自定义背景色" type="color" value={config.background} onChange={(event) => setCustomConfig({ background: event.target.value.toUpperCase() })} />}
          <SelectField label="边距" value={config.marginMode} onChange={(event) => setCustomConfig({ marginMode: event.target.value as "auto" | "fixed" })}>
            <option value="auto">自动边距</option><option value="fixed">固定像素</option>
          </SelectField>
          {config.marginMode === "fixed" && <InputField label="边距像素" type="number" min={0} max={Math.floor(Math.min(config.width, config.height) / 2) - 1} value={config.margin} onChange={(event) => setCustomConfig({ margin: Number(event.target.value) })} />}
          <SelectField label="输出格式" value={config.format} onChange={(event) => {
            const format = event.target.value as OutputFormat;
            setCustomConfig({ format, background: format === "jpeg" && config.background === "transparent" ? "#FFFFFF" : config.background });
          }}>
            <option value="jpeg">JPG</option><option value="png">PNG</option><option value="webp">WebP</option>
          </SelectField>
          <div className="settings-actions">
            <Button variant="secondary" disabled={submitting} onClick={saveTemplate}>保存为模板</Button>
            <Button disabled={submitting} onClick={startProcessing}>{submitting ? "提交中…" : "开始处理"}</Button>
          </div>
          {error && <p className="form-error" role="alert">{error}</p>}
        </aside>
      </div>
    </section>
  );
}
