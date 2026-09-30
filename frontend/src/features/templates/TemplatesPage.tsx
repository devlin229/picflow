import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { errorMessage } from "../../api/client";
import { createTemplate, deleteTemplate } from "../../api/templates";
import { Button } from "../../components/ui/Button";
import { InputField } from "../../components/ui/Field";
import { PageHeader } from "../../components/ui/PageHeader";
import { usePicFlowStore } from "../../store/usePicFlowStore";

export function TemplatesPage() {
  const navigate = useNavigate();
  const templates = usePicFlowStore((state) => state.templates);
  const config = usePicFlowStore((state) => state.config);
  const setConfig = usePicFlowStore((state) => state.setConfig);
  const addTemplate = usePicFlowStore((state) => state.addTemplate);
  const removeTemplate = usePicFlowStore((state) => state.removeTemplate);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");

  const create = async () => {
	if (!name.trim()) {
	  setError("请输入模板名称");
	  return;
	}
	setLoading(true);
	setError("");
	try {
	  const template = await createTemplate(name.trim(), `${config.width} × ${config.height} · ${config.format.toUpperCase()}`, config);
	  addTemplate(template);
	  setName("");
	  setCreating(false);
	} catch (requestError) {
	  setError(errorMessage(requestError));
	} finally {
	  setLoading(false);
	}
  };

  const remove = async (id: string) => {
	setLoading(true);
	setError("");
	try {
	  await deleteTemplate(id);
	  removeTemplate(id);
	} catch (requestError) {
	  setError(errorMessage(requestError));
	} finally {
	  setLoading(false);
	}
  };
  return (
    <section className="page">
	  <PageHeader title="模板管理" description="保存和复用常用处理方案，确保规则型输出稳定一致。" actions={<Button disabled={loading} onClick={() => setCreating((value) => !value)}><Plus size={16} />新建模板</Button>} />
	  {error && <p className="form-error" role="alert">{error}</p>}
      {creating && <section className="panel template-create-form">
        <InputField label="模板名称" maxLength={80} value={name} onChange={(event) => setName(event.target.value)} />
        <div className="settings-actions"><Button variant="secondary" onClick={() => setCreating(false)}>取消</Button><Button disabled={loading} onClick={() => void create()}>{loading ? "保存中…" : "保存模板"}</Button></div>
      </section>}
      <div className="template-grid">
        {templates.filter((template) => template.type === "main").map((template) => (
          <article className="panel template-card" key={template.id}>
            <span>图片处理模板</span>
            <h2>{template.name}</h2><p>{template.description}</p>
			<div><Button variant="ghost" disabled={loading || template.builtIn} onClick={() => void remove(template.id)}><Trash2 size={14} />删除</Button><Button variant="secondary" onClick={() => {
			  setConfig({ ...template.config, templateId: template.id, template: template.name });
			  navigate("/");
			}}>应用模板</Button></div>
          </article>
        ))}
      </div>
    </section>
  );
}
