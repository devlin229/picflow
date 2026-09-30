import { Navigate, useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { InputField } from "../../components/ui/Field";
import { PageHeader } from "../../components/ui/PageHeader";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { downloadBlob } from "../../lib/images";
import { usePicFlowStore } from "../../store/usePicFlowStore";

const positionLabels = {
  "top-left": "左上",
  "top-right": "右上",
  "middle-right": "右侧居中",
  "bottom-left": "左下",
  "bottom-right": "右下",
  custom: "自由位置",
};

export function SizeChartResultPage() {
  const navigate = useNavigate();
  const result = usePicFlowStore((state) => state.sizeChart);
  const annotation = usePicFlowStore((state) => state.annotation);
  if (!result || !annotation) return <Navigate to="/size-chart" replace />;
  return (
    <section className="page">
      <PageHeader title="规格图已生成" description={`${result.name} · ${result.width} × ${result.height}`} actions={<StatusBadge>生成成功</StatusBadge>} />
      <div className="size-layout">
        <section className="panel preview-panel"><img className="generated-image" src={result.url} alt="生成的商品规格图" /></section>
        <aside className="panel size-settings">
          <h3>规格信息</h3>
          <div className="result-specifications">{annotation.specifications.map((item) => <div key={item.id}><span>{item.label}</span><strong>{item.value}</strong></div>)}</div>
          <InputField label="表格位置" readOnly value={positionLabels[annotation.tablePosition.preset]} />
          <InputField label="表格大小" readOnly value={`${Math.round(annotation.tablePosition.width * 100)}%`} />
          <InputField label="规格样式" readOnly value={annotation.tableStyle.preset === "custom" ? "自定义" : annotation.tableStyle.preset === "classic" ? "经典蓝" : annotation.tableStyle.preset === "dark" ? "深色" : "电商红"} />
          <InputField label="模板" readOnly value={annotation.template} />
          <div className="neutral-strip">原图未被覆盖，规格图已作为新文件输出。</div>
          <Button onClick={() => downloadBlob(result.blob, result.name)}>下载规格图</Button>
          <Button variant="secondary" onClick={() => navigate("/results")}>返回结果列表</Button>
        </aside>
      </div>
    </section>
  );
}
