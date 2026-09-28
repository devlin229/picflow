import { ImageUp, Trash2 } from "lucide-react";
import { useCallback, useState } from "react";
import { useDropzone } from "react-dropzone";
import { useNavigate } from "react-router-dom";
import { errorMessage } from "../../api/client";
import { createTask } from "../../api/tasks";
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
      const validFiles = files.filter((file) => ["image/jpeg", "image/png", "image/webp"].includes(file.type));
      const next = await Promise.all(validFiles.map(createImageAsset));
      setAssets([...assets, ...next]);
    } catch {
      setError("部分图片无法读取，请确认文件没有损坏后重试。");
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

  const startProcessing = async () => {
    setSubmitting(true);
    setError("");
    try {
      const task = await createTask(assets, config);
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
            <Button variant="ghost" onClick={() => setAssets([])}><Trash2 size={15} />清空</Button>
          </div>
          <div className="asset-grid">
            {assets.map((asset) => <ImageCard key={asset.id} src={asset.url} name={asset.name} meta={`${asset.width} × ${asset.height} · ${asset.format}`} />)}
          </div>
          <div className="info-strip">默认等比缩放并补边，保持商品主体完整、不拉伸。</div>
        </section>
        <aside className="panel settings-panel">
          <h3>输出设置</h3>
          <SelectField label="处理模板" value={config.templateId} onChange={(event) => {
            const template = templates.find((item) => item.id === event.target.value);
            if (template) applyTemplate(template);
          }}>
            {templates.filter((item) => item.type === "main").map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </SelectField>
          <SelectField label="输出比例" value={config.ratio} onChange={(event) => {
            const ratio = event.target.value;
            const [w, h] = ratio.split(":").map(Number);
            setConfig({ ratio, height: Math.round(config.width * h / w) });
          }}>
            {["1:1", "4:5", "3:4", "9:16"].map((value) => <option key={value}>{value}</option>)}
          </SelectField>
          <InputField label="输出宽度" type="number" min={100} max={5000} value={config.width} onChange={(event) => setConfig({ width: Number(event.target.value) })} />
          <InputField label="输出高度" type="number" min={100} max={5000} value={config.height} onChange={(event) => setConfig({ height: Number(event.target.value) })} />
          <InputField label="背景" value={config.background} onChange={(event) => setConfig({ background: event.target.value })} />
          <SelectField label="输出格式" value={config.format} onChange={(event) => setConfig({ format: event.target.value as OutputFormat })}>
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
