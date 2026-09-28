import { CircleHelp, Images, LayoutTemplate, WandSparkles } from "lucide-react";
import { useEffect } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { listTemplates } from "../../api/templates";
import { usePicFlowStore } from "../../store/usePicFlowStore";

const navItems = [
  { to: "/", label: "工作台", icon: WandSparkles, end: true },
  { to: "/results", label: "结果", icon: Images },
  { to: "/templates", label: "模板管理", icon: LayoutTemplate },
];

export function AppShell() {
  const setTemplates = usePicFlowStore((state) => state.setTemplates);

  useEffect(() => {
    void listTemplates().then(setTemplates).catch(() => {
      // 后端暂时不可用时保留内置模板，具体错误会在执行操作时展示。
    });
  }, [setTemplates]);

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <strong>PicFlow</strong>
          <span>电商图片工作台</span>
        </div>
        <nav className="sidebar__nav">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink key={to} to={to} end={end} className={({ isActive }) => `nav-item ${isActive ? "nav-item--active" : ""}`}>
              <Icon size={16} strokeWidth={2} />
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="sidebar__spacer" />
        <div className="help-card">
          <span><CircleHelp size={15} />需要帮助？</span>
          <p>查看格式、尺寸和批量处理说明</p>
        </div>
      </aside>
      <main className="main-content"><Outlet /></main>
    </div>
  );
}
