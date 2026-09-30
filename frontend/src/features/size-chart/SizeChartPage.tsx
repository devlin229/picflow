import { GripVertical, Plus, Trash2 } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import type { CSSProperties, PointerEvent as ReactPointerEvent } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { errorMessage } from "../../api/client";
import { createSizeChart } from "../../api/tasks";
import { Button } from "../../components/ui/Button";
import { InputField, RangeField, SelectField } from "../../components/ui/Field";
import { PageHeader } from "../../components/ui/PageHeader";
import { specificationStylePresets, usePicFlowStore } from "../../store/usePicFlowStore";
import type { SizeAnnotation, SpecificationStylePreset, SpecificationTablePosition, SpecificationTablePreset, SpecificationTableStyle } from "../../types";

const presetLabels: Record<Exclude<SpecificationTablePreset, "custom">, string> = {
  "top-left": "左上",
  "top-right": "右上",
  "bottom-left": "左下",
  "bottom-right": "右下",
};

const initialPosition: SpecificationTablePosition = { preset: "top-right", x: 0.78, y: 0.22, width: 0.42 };

function colorWithOpacity(color: string, opacity: number) {
  const alpha = Math.round(opacity / 100 * 255).toString(16).padStart(2, "0");
  return `${color}${alpha}`;
}

export function SizeChartPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const outputs = usePicFlowStore((state) => state.outputs);
  const assets = usePicFlowStore((state) => state.assets);
  const taskId = usePicFlowStore((state) => state.taskId);
  const templates = usePicFlowStore((state) => state.templates);
  const specifications = usePicFlowStore((state) => state.specificationDraft);
  const specificationStyle = usePicFlowStore((state) => state.specificationStyleDraft);
  const setAnnotation = usePicFlowStore((state) => state.setAnnotation);
  const setSizeChart = usePicFlowStore((state) => state.setSizeChart);
  const setSpecifications = usePicFlowStore((state) => state.setSpecificationDraft);
  const setSpecificationStyle = usePicFlowStore((state) => state.setSpecificationStyleDraft);
  const [sourceId, setSourceId] = useState(outputs[0]?.id ?? "");
  const requestedTemplate = searchParams.get("template");
  const [templateId, setTemplateId] = useState(templates.find((item) => item.id === requestedTemplate && item.type === "size")?.id ?? templates.find((item) => item.type === "size")?.id ?? "size-standard");
  const [tablePosition, setTablePosition] = useState<SpecificationTablePosition>(initialPosition);
  const [dragging, setDragging] = useState(false);
  const [resizing, setResizing] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const previewRef = useRef<HTMLDivElement>(null);
  const tableRef = useRef<HTMLDivElement>(null);
  const source = useMemo(() => outputs.find((item) => item.id === sourceId) ?? outputs[0], [outputs, sourceId]);
  const sourcePreview = useMemo(() => assets.find((item) => item.id === source?.sourceId), [assets, source]);
  const template = useMemo(() => templates.find((item) => item.id === templateId && item.type === "size"), [templateId, templates]);
  const visibleSpecifications = useMemo(() => specifications.filter((item) => item.label.trim() && item.value.trim()), [specifications]);

  if (!source || !taskId) return <Navigate to="/" replace />;

  const updateSpecification = (id: string, key: "label" | "value", value: string) => {
    setSpecifications(specifications.map((item) => item.id === id ? { ...item, [key]: value } : item));
  };

  const addSpecification = () => {
    setSpecifications([...specifications, { id: crypto.randomUUID(), label: "", value: "" }]);
  };

  const removeSpecification = (id: string) => {
    setSpecifications(specifications.filter((item) => item.id !== id));
  };

  const clampPosition = (x: number, y: number) => {
    const preview = previewRef.current?.getBoundingClientRect();
    const table = tableRef.current?.getBoundingClientRect();
    if (!preview || !table) return { x: Math.min(0.9, Math.max(0.1, x)), y: Math.min(0.9, Math.max(0.1, y)) };
    const minimumX = Math.min(0.5, (table.width / 2 + 12) / preview.width);
    const minimumY = Math.min(0.5, (table.height / 2 + 12) / preview.height);
    return {
      x: Math.min(1 - minimumX, Math.max(minimumX, x)),
      y: Math.min(1 - minimumY, Math.max(minimumY, y)),
    };
  };

  const setPreset = (preset: Exclude<SpecificationTablePreset, "custom">) => {
    const preview = previewRef.current?.getBoundingClientRect();
    const table = tableRef.current?.getBoundingClientRect();
    const edgeX = preview && table ? Math.min(0.5, (table.width / 2 + 18) / preview.width) : 0.2;
    const edgeY = preview && table ? Math.min(0.5, (table.height / 2 + 18) / preview.height) : 0.2;
    setTablePosition({
      preset,
      x: preset.endsWith("left") ? edgeX : 1 - edgeX,
      y: preset.startsWith("top") ? edgeY : 1 - edgeY,
      width: tablePosition.width,
    });
  };

  const moveTable = (event: ReactPointerEvent<HTMLDivElement>) => {
    const preview = previewRef.current?.getBoundingClientRect();
    if (!preview) return;
    const next = clampPosition((event.clientX - preview.left) / preview.width, (event.clientY - preview.top) / preview.height);
    setTablePosition((current) => ({ ...current, preset: "custom", ...next }));
  };

  const startDragging = (event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragging(true);
    moveTable(event);
  };

  const resizeTable = (event: ReactPointerEvent<HTMLDivElement>) => {
    const preview = previewRef.current?.getBoundingClientRect();
    if (!preview) return;
    const pointerX = (event.clientX - preview.left) / preview.width;
    const availableWidth = 2 * Math.min(tablePosition.x - 0.02, 0.98 - tablePosition.x);
    const width = Math.min(0.8, Math.max(0.24, Math.min(availableWidth, (pointerX - tablePosition.x) * 2)));
    setTablePosition((current) => ({ ...current, width }));
  };

  const startResizing = (event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
    event.currentTarget.setPointerCapture(event.pointerId);
    setResizing(true);
    resizeTable(event);
  };

  const setTableWidth = (width: number) => {
    const maximum = 2 * Math.min(tablePosition.x - 0.02, 0.98 - tablePosition.x);
    setTablePosition((current) => ({ ...current, width: Math.min(maximum, Math.max(0.24, width)) }));
  };

  const applySpecificationStylePreset = (preset: SpecificationStylePreset) => {
    if (preset === "custom") {
      setSpecificationStyle({ ...specificationStyle, preset });
      return;
    }
    setSpecificationStyle({ ...specificationStylePresets[preset] });
  };

  const updateSpecificationStyle = (value: Partial<SpecificationTableStyle>) => {
    setSpecificationStyle({ ...specificationStyle, ...value, preset: "custom" });
  };

  const generate = async () => {
    if (!template) {
      setError("请选择有效的规格图模板");
      return;
    }
    if (!visibleSpecifications.length) {
      setError("请至少填写一项规格参数");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const annotation: SizeAnnotation = {
        sourceId: source.sourceId,
        specifications: visibleSpecifications.map((item) => ({ ...item, label: item.label.trim(), value: item.value.trim() })),
        tablePosition,
        tableStyle: { ...specificationStyle, title: specificationStyle.title.trim() || "商品规格" },
        templateId: template.id,
        template: template.name,
      };
      const result = await createSizeChart(taskId, annotation);
      setAnnotation(annotation);
      setSizeChart(result);
      navigate("/size-chart/result");
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  };

  const canvasWidth = template?.sizeConfig?.canvasWidth ?? source.width;
  const canvasHeight = template?.sizeConfig?.canvasHeight ?? source.height;
  const canvasBackground = template?.sizeConfig?.background ?? "#FFFFFF";
  const annotationColor = template?.sizeConfig?.annotationColor ?? "#2563EB";

  return (
    <section className="page">
      <PageHeader title="生成商品规格图" description="填写需要展示的规格，并将表格放到商品图上的合适位置。" />
      <div className="size-layout">
        <section className="panel preview-panel">
          <div className="panel-title-row"><h3>规格表格预览</h3><span className="preview-hint">拖动表头调整位置，拖动右下角调整大小</span></div>
          <div className="spec-preview-stage">
            <div
              className="spec-preview-canvas"
              ref={previewRef}
              style={{ aspectRatio: `${canvasWidth} / ${canvasHeight}`, background: canvasBackground === "transparent" ? undefined : canvasBackground }}
            >
              <img src={sourcePreview?.url ?? source.url} alt={sourcePreview?.name ?? source.name} />
              <div
                className={`spec-overlay-table ${dragging || resizing ? "spec-overlay-table--dragging" : ""}`}
                ref={tableRef}
                style={{
                  left: `${tablePosition.x * 100}%`,
                  top: `${tablePosition.y * 100}%`,
                  width: `${tablePosition.width * 100}%`,
                  color: specificationStyle.textColor,
                  backgroundColor: colorWithOpacity(specificationStyle.backgroundColor, specificationStyle.opacity),
                  borderColor: specificationStyle.accentColor || annotationColor,
                  "--spec-scale": Math.min(1.45, Math.max(0.78, tablePosition.width / 0.42)),
                  "--spec-accent": specificationStyle.accentColor || annotationColor,
                  "--spec-text": specificationStyle.textColor,
                } as CSSProperties}
              >
                <div
                  className="spec-overlay-table__header"
                  style={{ background: specificationStyle.accentColor || annotationColor }}
                  onPointerDown={startDragging}
                  onPointerMove={(event) => dragging && moveTable(event)}
                  onPointerUp={(event) => { event.currentTarget.releasePointerCapture(event.pointerId); setDragging(false); }}
                  onPointerCancel={() => setDragging(false)}
                >
                  <GripVertical size={15} />{specificationStyle.title.trim() || "商品规格"}
                </div>
                <table><tbody>
                  {visibleSpecifications.length ? visibleSpecifications.map((item) => <tr key={item.id}><th>{item.label}</th><td>{item.value}</td></tr>) : <tr><td className="spec-overlay-table__empty">填写参数后将在这里显示</td></tr>}
                </tbody></table>
                <div
                  className="spec-overlay-table__resize"
                  role="slider"
                  aria-label="调整规格表格宽度"
                  aria-valuemin={24}
                  aria-valuemax={80}
                  aria-valuenow={Math.round(tablePosition.width * 100)}
                  onPointerDown={startResizing}
                  onPointerMove={(event) => resizing && resizeTable(event)}
                  onPointerUp={(event) => { event.currentTarget.releasePointerCapture(event.pointerId); setResizing(false); }}
                  onPointerCancel={() => setResizing(false)}
                />
              </div>
            </div>
          </div>
        </section>
        <aside className="panel size-settings">
          <h3>规格参数</h3>
          <SelectField label="目标图片" value={source.id} onChange={(event) => setSourceId(event.target.value)}>{outputs.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</SelectField>
          <div className="specification-editor">
            <div className="specification-editor__heading"><span>参数名称</span><span>参数值（可含单位）</span><i /></div>
            {specifications.map((item) => (
              <div className="specification-editor__row" key={item.id}>
                <input aria-label="参数名称" maxLength={20} placeholder="例如：重量" value={item.label} onChange={(event) => updateSpecification(item.id, "label", event.target.value)} />
                <input aria-label="参数值" maxLength={40} placeholder="例如：2.5 kg" value={item.value} onChange={(event) => updateSpecification(item.id, "value", event.target.value)} />
                <button type="button" aria-label={`删除${item.label || "空白"}参数`} disabled={specifications.length === 1} onClick={() => removeSpecification(item.id)}><Trash2 size={15} /></button>
              </div>
            ))}
            <Button type="button" variant="secondary" disabled={specifications.length >= 10} onClick={addSpecification}><Plus size={15} />添加参数</Button>
          </div>
          <div className="field"><span>表格预设位置</span><div className="position-presets">
            {(Object.keys(presetLabels) as Array<Exclude<SpecificationTablePreset, "custom">>).map((preset) => <button className={tablePosition.preset === preset ? "active" : ""} type="button" key={preset} onClick={() => setPreset(preset)}>{presetLabels[preset]}</button>)}
          </div></div>
          <RangeField label={`表格大小（${Math.round(tablePosition.width * 100)}%）`} min={24} max={80} value={Math.round(tablePosition.width * 100)} onValueChange={(width) => setTableWidth(width / 100)} />
          <div className="settings-section-heading"><div><strong>规格样式</strong><span>预设后仍可继续调整颜色与透明度</span></div></div>
          <SelectField label="样式预设" value={specificationStyle.preset} onChange={(event) => applySpecificationStylePreset(event.target.value as SpecificationStylePreset)}>
            <option value="classic">经典蓝</option><option value="dark">深色</option><option value="commerce-red">电商红</option><option value="custom">自定义</option>
          </SelectField>
          <InputField label="表头文字" maxLength={20} value={specificationStyle.title} onChange={(event) => updateSpecificationStyle({ title: event.target.value })} />
          <div className="field-row specification-style-colors">
            <InputField label="主题色" type="color" value={specificationStyle.accentColor} onChange={(event) => updateSpecificationStyle({ accentColor: event.target.value.toUpperCase() })} />
            <InputField label="背景色" type="color" value={specificationStyle.backgroundColor} onChange={(event) => updateSpecificationStyle({ backgroundColor: event.target.value.toUpperCase() })} />
            <InputField label="文字色" type="color" value={specificationStyle.textColor} onChange={(event) => updateSpecificationStyle({ textColor: event.target.value.toUpperCase() })} />
          </div>
          <RangeField label={`背景透明度（${specificationStyle.opacity}%）`} min={55} max={100} value={specificationStyle.opacity} onValueChange={(opacity) => updateSpecificationStyle({ opacity })} />
          <SelectField label="规格图模板" value={templateId} onChange={(event) => setTemplateId(event.target.value)}>{templates.filter((item) => item.type === "size").map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</SelectField>
          <div className="info-strip"><strong>{tablePosition.preset === "custom" ? "当前为自由位置" : `当前为${presetLabels[tablePosition.preset]}预设`}</strong><span>拖动蓝色表头调整位置，拖动右下角调整大小；未填写的参数不会出现在结果中。</span></div>
          {error && <p className="form-error" role="alert">{error}</p>}
          <div className="settings-actions"><Button variant="secondary" onClick={() => navigate("/results")}>返回结果</Button><Button disabled={loading || !visibleSpecifications.length} onClick={() => void generate()}>{loading ? "生成中…" : "生成规格图"}</Button></div>
        </aside>
      </div>
    </section>
  );
}
