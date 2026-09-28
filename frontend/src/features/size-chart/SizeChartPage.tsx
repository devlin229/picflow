import { useMemo, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { errorMessage } from "../../api/client";
import { createSizeChart } from "../../api/tasks";
import { Button } from "../../components/ui/Button";
import { InputField, SelectField } from "../../components/ui/Field";
import { PageHeader } from "../../components/ui/PageHeader";
import { usePicFlowStore } from "../../store/usePicFlowStore";
import type { SizeAnnotation } from "../../types";

export function SizeChartPage() {
  const navigate = useNavigate();
  const outputs = usePicFlowStore((state) => state.outputs);
  const taskId = usePicFlowStore((state) => state.taskId);
  const setAnnotation = usePicFlowStore((state) => state.setAnnotation);
  const setSizeChart = usePicFlowStore((state) => state.setSizeChart);
  const [sourceId, setSourceId] = useState(outputs[0]?.id ?? "");
  const [form, setForm] = useState({ width: 42, height: 58, depth: 46, unit: "cm" as "cm" | "mm" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const source = useMemo(() => outputs.find((item) => item.id === sourceId) ?? outputs[0], [outputs, sourceId]);
  if (!source || !taskId) return <Navigate to="/" replace />;

  const generate = async () => {
    setLoading(true);
	setError("");
    try {
	  const annotation: SizeAnnotation = { sourceId: source.sourceId, ...form, templateId: "size-standard", template: "标准蓝色标注" };
	  const result = await createSizeChart(taskId, annotation);
      setAnnotation(annotation);
      setSizeChart(result);
      navigate("/size-chart/result");
	} catch (requestError) {
	  setError(errorMessage(requestError));
	} finally { setLoading(false); }
  };

  return (
    <section className="page">
      <PageHeader title="生成尺寸图" description="选择图片并输入真实尺寸，系统仅负责绘制标注。" />
      <div className="size-layout">
        <section className="panel preview-panel">
          <h3>标注预览</h3>
          <div className="dimension-preview">
            <img src={source.url} alt={source.name} />
            <span className="dimension-line dimension-line--vertical"><i>{form.height} {form.unit}</i></span>
            <span className="dimension-line dimension-line--horizontal"><i>{form.width} {form.unit}</i></span>
            {form.depth > 0 && <span className="dimension-line dimension-line--depth"><i>{form.depth} {form.unit}</i></span>}
          </div>
        </section>
        <aside className="panel size-settings">
          <h3>尺寸参数</h3>
          <SelectField label="目标图片" value={source.id} onChange={(event) => setSourceId(event.target.value)}>{outputs.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</SelectField>
          <div className="field-row"><InputField label="宽度" type="number" min={0.1} value={form.width} onChange={(event) => setForm({ ...form, width: Number(event.target.value) })} /><InputField label="高度" type="number" min={0.1} value={form.height} onChange={(event) => setForm({ ...form, height: Number(event.target.value) })} /></div>
          <InputField label="深度（可选）" type="number" min={0} value={form.depth} onChange={(event) => setForm({ ...form, depth: Number(event.target.value) })} />
          <SelectField label="单位" value={form.unit} onChange={(event) => setForm({ ...form, unit: event.target.value as "cm" | "mm" })}><option value="cm">cm</option><option value="mm">mm</option></SelectField>
          <SelectField label="尺寸图模板" value="标准蓝色标注"><option>标准蓝色标注</option></SelectField>
          <div className="info-strip"><strong>尺寸不会自动推断</strong><span>请以商品真实测量值为准。</span></div>
		  {error && <p className="form-error" role="alert">{error}</p>}
          <div className="settings-actions"><Button variant="secondary" onClick={() => navigate("/results")}>返回结果</Button><Button disabled={loading || form.width <= 0 || form.height <= 0 || form.depth < 0} onClick={generate}>{loading ? "生成中…" : "生成尺寸图"}</Button></div>
        </aside>
      </div>
    </section>
  );
}
