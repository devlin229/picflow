import { Check, CircleAlert, LoaderCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { PageHeader } from "../../components/ui/PageHeader";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { errorMessage } from "../../api/client";
import { retryTask, waitForTask } from "../../api/tasks";
import { usePicFlowStore } from "../../store/usePicFlowStore";

export function ProcessingPage() {
  const navigate = useNavigate();
  const taskId = usePicFlowStore((state) => state.taskId);
  const [completed, setCompleted] = useState(0);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState("");
  const [retryVersion, setRetryVersion] = useState(0);
  const [retrying, setRetrying] = useState(false);

  useEffect(() => {
    if (!taskId) return;
    const activeTaskId = taskId;
    const controller = new AbortController();
    async function run() {
      try {
        const task = await waitForTask(activeTaskId, controller.signal, (current) => {
          setCompleted(current.completed_assets);
          setTotal(current.total_assets);
        });
        if (controller.signal.aborted) return;
        setCompleted(task.completed_assets);
        setTotal(task.total_assets);
        navigate(`/results?task=${encodeURIComponent(activeTaskId)}`, { replace: true });
      } catch (requestError) {
        if (requestError instanceof DOMException && requestError.name === "AbortError") return;
        setError(errorMessage(requestError));
      }
    }
    void run();
    return () => controller.abort();
  }, [navigate, retryVersion, taskId]);

  if (!taskId) return <Navigate to="/" replace />;
  const percent = total > 0 ? Math.round((completed / total) * 100) : 0;

  const retry = async () => {
    setRetrying(true);
    try {
      await retryTask(taskId);
      setCompleted(0);
      setError("");
      setRetryVersion((value) => value + 1);
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setRetrying(false);
    }
  };

  return (
    <section className="page">
      <PageHeader title="正在处理图片" description={`任务 ${taskId}`} actions={<StatusBadge tone="processing">处理中</StatusBadge>} />
      <div className="panel processing-card">
        {error ? <>
          <CircleAlert className="processing-error" size={78} />
          <h2>处理没有完成</h2>
          <p>{error}</p>
          <div className="processing-error-actions"><Button variant="secondary" onClick={() => navigate("/")}>返回配置</Button><Button disabled={retrying} onClick={() => void retry()}>{retrying ? "重新提交中…" : "重试当前任务"}</Button></div>
        </> : <>
          <LoaderCircle className="processing-spinner" size={96} />
          <h2>正在批量生成标准化主图</h2>
          <p>已完成 {completed} / {total || "-"}</p>
          <div className="progress"><span style={{ width: `${percent}%` }} /></div>
          <div className="processing-steps">
            <span><Check />读取原图</span><span><Check />统一画布</span><span className={total > 0 && completed === total ? "done" : "active"}>… 导出文件</span>
          </div>
          <small>处理完成后将自动进入结果页</small>
        </>}
      </div>
    </section>
  );
}
