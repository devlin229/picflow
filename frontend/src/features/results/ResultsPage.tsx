import { Download, Images } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { apiURL, errorMessage } from "../../api/client";
import { deleteTask, downloadOutput, downloadTaskArchive, getTask, listTasks } from "../../api/tasks";
import type { Task, TaskHistory } from "../../api/tasks";
import { Button } from "../../components/ui/Button";
import { ImageCard } from "../../components/ui/ImageCard";
import { Modal } from "../../components/ui/Modal";
import { PageHeader } from "../../components/ui/PageHeader";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { downloadBlob, formatBytes } from "../../lib/images";
import { usePicFlowStore } from "../../store/usePicFlowStore";

function TaskStatus({ task }: { task: Task }) {
  const labels = { queued: "排队中", processing: "处理中", succeeded: "已完成", failed: "失败" };
  return <StatusBadge tone={task.status === "succeeded" ? "success" : task.status === "failed" ? "error" : "processing"}>{labels[task.status]}</StatusBadge>;
}

export function ResultsPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const selectedId = params.get("task") || "";
  const requestedPage = Number(params.get("page") || 1);
  const page = Number.isSafeInteger(requestedPage) && requestedPage > 0 ? requestedPage : 1;
  const resetTask = usePicFlowStore((state) => state.resetTask);
  const setTaskId = usePicFlowStore((state) => state.setTaskId);
  const setOutputs = usePicFlowStore((state) => state.setOutputs);
  const [history, setHistory] = useState<TaskHistory | null>(null);
  const [task, setTask] = useState<Task | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [refresh, setRefresh] = useState(0);
  const [downloading, setDownloading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [pendingDeletion, setPendingDeletion] = useState<Task | null>(null);
  const [deleteError, setDeleteError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError("");
    setTask(null);
    async function load() {
      try {
        if (selectedId) {
          const result = await getTask(selectedId, controller.signal);
          if (!controller.signal.aborted) setTask(result);
        } else {
          const result = await listTasks(page, controller.signal);
          if (controller.signal.aborted) return;
          setHistory(result);
          if (result.page !== page) setParams({ page: String(result.page) }, { replace: true });
        }
      } catch (requestError) {
        if (!controller.signal.aborted) setError(errorMessage(requestError));
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }
    void load();
    return () => controller.abort();
  }, [page, selectedId, refresh, setParams]);

  const download = async (output?: Task["outputs"][number]) => {
    if (!task || downloading) return;
    setDownloading(true);
    setError("");
    try {
      const blob = output ? await downloadOutput(output.download_url) : await downloadTaskArchive(task.id);
      downloadBlob(blob, output?.filename ?? `picflow-${task.id}.zip`);
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setDownloading(false);
    }
  };

  const confirmDelete = async () => {
    if (!pendingDeletion || deleting) return;
    setDeleting(true);
    setDeleteError("");
    try {
      await deleteTask(pendingDeletion.id);
      if (usePicFlowStore.getState().taskId === pendingDeletion.id) {
        setTaskId(null);
        setOutputs([]);
      }
      setPendingDeletion(null);
      setRefresh((value) => value + 1);
    } catch (requestError) {
      setDeleteError(errorMessage(requestError));
    } finally {
      setDeleting(false);
    }
  };

  return <section className="page">
    <PageHeader title={selectedId ? "处理结果" : "结果"} description={selectedId ? "" : "查看和下载历史结果，无需重新生成。"} actions={<>
      {selectedId && <Button variant="secondary" disabled={downloading} onClick={() => setParams({ page: String(page) })}>返回列表</Button>}
      <Button variant="secondary" disabled={loading || downloading} onClick={() => setRefresh((value) => value + 1)}>刷新</Button>
      <Button variant="secondary" disabled={downloading} onClick={() => { resetTask(); navigate("/"); }}>新建任务</Button>
      {task && task.outputs.length > 0 && <Button disabled={downloading} onClick={() => void download()}><Download size={16} />{downloading ? "下载中…" : "下载全部 ZIP"}</Button>}
    </>} />
    <section className="panel results-panel" aria-busy={loading}>
      {loading && <p role="status">正在加载结果…</p>}
      {error && <p className="form-error" role="alert">{error}</p>}
      {!loading && error && !task && <Button variant="secondary" onClick={() => setRefresh((value) => value + 1)}>重新加载</Button>}
      {!loading && selectedId && task && <>
        <div className="panel-title-row"><h3>{task.assets[0]?.filename || "处理任务"}</h3><TaskStatus task={task} /></div>
        <p>{new Date(task.created_at).toLocaleString("zh-CN")} · {task.completed_assets} / {task.total_assets} 张</p>
        {task.error_message && <p className="form-error">{task.error_message}</p>}
        {task.status !== "succeeded" && <Button variant="secondary" onClick={() => { setOutputs([]); setTaskId(task.id); navigate("/processing"); }}>查看处理状态</Button>}
        {!task.outputs.length && <p>暂无可下载结果。</p>}
        <div className="result-grid">{task.outputs.map((output) => <article key={output.id} className="result-card">
          <ImageCard src={apiURL(output.download_url)} name={output.filename} meta={`${output.width} × ${output.height} · ${formatBytes(output.size)}`} />
          <Button variant="ghost" disabled={downloading} onClick={() => void download(output)}><Download size={15} />下载</Button>
        </article>)}</div>
      </>}
      {!loading && !selectedId && history && !error && <>
        {!history.items.length && <p>暂无历史结果，完成图片处理后会自动保存在这里。</p>}
        <div className="history-grid">{history.items.map((item) => <article className="history-card" key={item.id}>
          <button className="history-open" onClick={() => setParams({ page: String(page), task: item.id })} aria-label={`查看 ${item.assets[0]?.filename || "任务"} 的结果`}>
            <span className="history-thumbnail">{item.outputs[0] ? <img src={apiURL(item.outputs[0].download_url)} alt="" loading="lazy" /> : <Images size={36} />}</span>
            <strong title={item.assets[0]?.filename}>{item.assets[0]?.filename || "处理任务"}</strong>
            <span>{new Date(item.created_at).toLocaleString("zh-CN")}</span>
            <span>{item.total_assets} 张图片 · {item.outputs.length} 个结果</span>
          </button>
          <div className="history-actions"><TaskStatus task={item} /><Button variant="ghost" disabled={item.status === "queued" || item.status === "processing"} onClick={() => { setDeleteError(""); setPendingDeletion(item); }}>删除</Button></div>
        </article>)}</div>
        {history.total > 0 && <nav className="history-pagination" aria-label="历史结果分页">
          <Button variant="secondary" disabled={page <= 1} onClick={() => setParams({ page: String(page - 1) })}>上一页</Button>
          <span>{page} / {Math.max(1, Math.ceil(history.total / history.page_size))} 页 · 共 {history.total} 个任务</span>
          <Button variant="secondary" disabled={page * history.page_size >= history.total} onClick={() => setParams({ page: String(page + 1) })}>下一页</Button>
        </nav>}
      </>}
    </section>
    <Modal open={Boolean(pendingDeletion)} title="删除历史结果" busy={deleting} onClose={() => setPendingDeletion(null)}>
      <p>将删除此任务的原图、处理结果和 AI 返回图片，删除后无法恢复。确定删除吗？</p>
      {deleteError && <p className="form-error" role="alert">{deleteError}</p>}
      <div className="settings-actions"><Button variant="secondary" autoFocus disabled={deleting} onClick={() => setPendingDeletion(null)}>取消</Button><Button disabled={deleting} onClick={() => void confirmDelete()}>{deleting ? "删除中…" : "确认删除"}</Button></div>
    </Modal>
  </section>;
}
