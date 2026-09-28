import { Download, Ruler } from "lucide-react";
import { useEffect, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { errorMessage } from "../../api/client";
import { deleteTask, downloadTaskArchive, getTask, toProcessedAsset } from "../../api/tasks";
import { Button } from "../../components/ui/Button";
import { ImageCard } from "../../components/ui/ImageCard";
import { PageHeader } from "../../components/ui/PageHeader";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { downloadBlob, formatBytes } from "../../lib/images";
import { usePicFlowStore } from "../../store/usePicFlowStore";

export function ResultsPage() {
  const navigate = useNavigate();
  const outputs = usePicFlowStore((state) => state.outputs);
  const taskId = usePicFlowStore((state) => state.taskId);
  const setOutputs = usePicFlowStore((state) => state.setOutputs);
  const resetTask = usePicFlowStore((state) => state.resetTask);
  const [downloading, setDownloading] = useState(false);
  const [loading, setLoading] = useState(Boolean(taskId && !outputs.length));
  const [error, setError] = useState("");

  useEffect(() => {
    if (!taskId || outputs.length) return;
    const controller = new AbortController();
    void getTask(taskId, controller.signal).then(async (task) => {
      if (task.status === "failed") throw new Error(task.error_message || "图片处理失败");
      if (task.status !== "succeeded") {
        navigate("/processing", { replace: true });
        return;
      }
      const recovered = await Promise.all(task.outputs.filter((output) => output.type === "standardized").map(toProcessedAsset));
      setOutputs(recovered);
    }).catch((requestError) => {
      if (requestError instanceof DOMException && requestError.name === "AbortError") return;
      setError(errorMessage(requestError));
    }).finally(() => setLoading(false));
    return () => controller.abort();
  }, [navigate, outputs.length, setOutputs, taskId]);

  if (!taskId) return <Navigate to="/" replace />;

  const startNewTask = async () => {
    setError("");
    try {
      await deleteTask(taskId);
      resetTask();
      navigate("/");
    } catch (requestError) {
      setError(errorMessage(requestError));
    }
  };

  const downloadAll = async () => {
	setDownloading(true);
	setError("");
	try {
	  const blob = await downloadTaskArchive(taskId);
	  downloadBlob(blob, "picflow-results.zip");
	} catch (requestError) {
	  setError(errorMessage(requestError));
	} finally {
	  setDownloading(false);
	}
  };

  return (
    <section className="page">
	  <PageHeader title="处理结果" description={`${outputs.length} 张图片已全部完成，可单张下载或批量导出。`} actions={<><StatusBadge>已完成</StatusBadge><Button variant="secondary" onClick={() => void startNewTask()}>新建任务</Button><Button disabled={downloading || loading || !outputs.length} onClick={downloadAll}><Download size={16} />{downloading ? "下载中…" : "下载全部 ZIP"}</Button></>} />
      <section className="panel results-panel">
		{loading && <p>正在恢复任务结果…</p>}
		{error && <p className="form-error" role="alert">{error}</p>}
        <div className="panel-title-row"><h3>输出图片</h3><Button variant="secondary" onClick={() => navigate("/size-chart")}><Ruler size={16} />生成尺寸图</Button></div>
        {!loading && <div className="result-grid">
          {outputs.map((output) => (
            <article key={output.id} className="result-card">
              <ImageCard src={output.url} name={output.name} meta={`${output.width} × ${output.height} · ${formatBytes(output.size)}`} />
              <Button variant="ghost" onClick={() => downloadBlob(output.blob, output.name)}><Download size={15} />下载</Button>
            </article>
          ))}
        </div>}
      </section>
    </section>
  );
}
