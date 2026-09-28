import { Navigate, useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { InputField } from "../../components/ui/Field";
import { PageHeader } from "../../components/ui/PageHeader";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { downloadBlob } from "../../lib/images";
import { usePicFlowStore } from "../../store/usePicFlowStore";

export function SizeChartResultPage() {
  const navigate = useNavigate();
  const result = usePicFlowStore((state) => state.sizeChart);
  const annotation = usePicFlowStore((state) => state.annotation);
  if (!result || !annotation) return <Navigate to="/size-chart" replace />;
  return (
    <section className="page">
      <PageHeader title="尺寸图已生成" description={`${result.name} · ${result.width} × ${result.height}`} actions={<StatusBadge>生成成功</StatusBadge>} />
      <div className="size-layout">
        <section className="panel preview-panel"><img className="generated-image" src={result.url} alt="生成的尺寸图" /></section>
        <aside className="panel size-settings"><h3>标注信息</h3><InputField label="宽度" readOnly value={`${annotation.width} ${annotation.unit}`} /><InputField label="高度" readOnly value={`${annotation.height} ${annotation.unit}`} /><InputField label="深度" readOnly value={`${annotation.depth ?? "-"} ${annotation.unit}`} /><InputField label="模板" readOnly value={annotation.template} /><div className="neutral-strip">原图未被覆盖，尺寸图已作为新文件输出。</div><Button onClick={() => downloadBlob(result.blob, result.name)}>下载尺寸图</Button><Button variant="secondary" onClick={() => navigate("/results")}>返回结果列表</Button></aside>
      </div>
    </section>
  );
}
