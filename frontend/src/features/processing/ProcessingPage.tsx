import { Check, CircleAlert, LoaderCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { PageHeader } from "../../components/ui/PageHeader";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { errorMessage } from "../../api/client";
import { toProcessedAsset, waitForTask } from "../../api/tasks";
import { usePicFlowStore } from "../../store/usePicFlowStore";

export function ProcessingPage() {
  const navigate = useNavigate();
  const assets = usePicFlowStore((state) => state.assets);
  const taskId = usePicFlowStore((state) => state.taskId);
  const setOutputs = usePicFlowStore((state) => state.setOutputs);
  const [completed, setCompleted] = useState(0);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!assets.length || !taskId) return;
	const activeTaskId = taskId;
    const controller = new AbortController();
    async function run() {
      try {
		const task = await waitForTask(activeTaskId, controller.signal);
		const outputs = await Promise.all(task.outputs.filter((output) => output.type === "standardized").map(toProcessedAsset));
		setCompleted(outputs.length);
		setOutputs(outputs);
		window.setTimeout(() => navigate("/results", { replace: true }), 350);
      } catch (requestError) {
		if (requestError instanceof DOMException && requestError.name === "AbortError") return;
		setError(errorMessage(requestError));
      }
    }
    void run();
    return () => controller.abort();
  }, [assets, navigate, setOutputs, taskId]);

  if (!assets.length || !taskId) return <Navigate to="/" replace />;
  const percent = Math.round((completed / assets.length) * 100);

  return (
    <section className="page">
	  <PageHeader title="正在处理图片" description={`任务 ${taskId}`} actions={<StatusBadge tone="processing">处理中</StatusBadge>} />
      <div className="panel processing-card">
        {error ? <>
          <CircleAlert className="processing-error" size={78} />
          <h2>处理没有完成</h2>
          <p>{error}</p>
          <Button onClick={() => navigate("/")}>返回配置</Button>
        </> : <>
          <LoaderCircle className="processing-spinner" size={96} />
          <h2>正在批量生成标准化主图</h2>
          <p>已完成 {completed} / {assets.length}</p>
          <div className="progress"><span style={{ width: `${percent}%` }} /></div>
          <div className="processing-steps">
            <span><Check />读取原图</span><span><Check />统一画布</span><span className={completed === assets.length ? "done" : "active"}>… 导出文件</span>
          </div>
          <small>处理完成后将自动进入结果页</small>
        </>}
      </div>
    </section>
  );
}
