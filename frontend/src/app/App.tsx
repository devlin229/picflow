import { Navigate, Route, Routes } from "react-router-dom";
import { AppShell } from "../components/layout/AppShell";
import { ProcessingPage } from "../features/processing/ProcessingPage";
import { ResultsPage } from "../features/results/ResultsPage";
import { SizeChartPage } from "../features/size-chart/SizeChartPage";
import { SizeChartResultPage } from "../features/size-chart/SizeChartResultPage";
import { TemplatesPage } from "../features/templates/TemplatesPage";
import { WorkspacePage } from "../features/workspace/WorkspacePage";

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<WorkspacePage />} />
        <Route path="processing" element={<ProcessingPage />} />
        <Route path="results" element={<ResultsPage />} />
        <Route path="size-chart" element={<SizeChartPage />} />
        <Route path="size-chart/result" element={<SizeChartResultPage />} />
        <Route path="templates" element={<TemplatesPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
