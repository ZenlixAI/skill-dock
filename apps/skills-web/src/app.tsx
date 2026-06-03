import { Navigate, Route, Routes } from 'react-router-dom';
import { AppShell } from './components/AppShell';
import { AuditPage } from './pages/AuditPage';
import { AuthoringFlowPage } from './pages/AuthoringFlowPage';
import { CatalogPage } from './pages/CatalogPage';
import { GovernancePage } from './pages/GovernancePage';
import { ImportPage } from './pages/ImportPage';
import { OverviewPage } from './pages/OverviewPage';
import { SkillDetailPage } from './pages/SkillDetailPage';
import { VersionDetailPage } from './pages/VersionDetailPage';

export function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<OverviewPage />} />
        <Route path="/authoring" element={<AuthoringFlowPage />} />
        <Route path="/catalog" element={<CatalogPage />} />
        <Route path="/import" element={<ImportPage />} />
        <Route path="/governance" element={<GovernancePage />} />
        <Route path="/audit" element={<AuditPage />} />
        <Route path="/skills/:packageName" element={<SkillDetailPage />} />
        <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
        <Route path="*" element={<Navigate replace to="/" />} />
      </Routes>
    </AppShell>
  );
}
