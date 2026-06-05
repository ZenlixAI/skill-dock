import { Navigate, Route, Routes } from 'react-router-dom';
import { AppShell } from './components/AppShell';
import { RequireAuthenticatedSession, RequireOperatorSession } from './components/RouteGuards';
import { AuditPage } from './pages/AuditPage';
import { AuthoringFlowPage } from './pages/AuthoringFlowPage';
import { CatalogPage } from './pages/CatalogPage';
import { GovernancePage } from './pages/GovernancePage';
import { ImportPage } from './pages/ImportPage';
import { LoginPage } from './pages/LoginPage';
import { OverviewPage } from './pages/OverviewPage';
import { SkillDetailPage } from './pages/SkillDetailPage';
import { VersionDetailPage } from './pages/VersionDetailPage';
import { WorkbenchHomePage } from './pages/WorkbenchHomePage';

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="*"
        element={
          <RequireAuthenticatedSession>
            <AppShell>
              <Routes>
                <Route path="/" element={<OverviewPage />} />
                <Route path="/workbench" element={<WorkbenchHomePage />} />
                <Route
                  path="/workbench/authoring"
                  element={
                    <RequireOperatorSession requiredAction="packaging.prepare">
                      <AuthoringFlowPage />
                    </RequireOperatorSession>
                  }
                />
                <Route path="/catalog" element={<CatalogPage />} />
                <Route
                  path="/workbench/import"
                  element={
                    <RequireOperatorSession requiredAction="packaging.prepare">
                      <ImportPage />
                    </RequireOperatorSession>
                  }
                />
                <Route
                  path="/workbench/governance"
                  element={
                    <RequireOperatorSession requiredAction="catalog.governance.skill.apply">
                      <GovernancePage />
                    </RequireOperatorSession>
                  }
                />
                <Route path="/workbench/audit" element={<AuditPage />} />
                <Route path="/authoring" element={<Navigate replace to="/workbench/authoring" />} />
                <Route path="/import" element={<Navigate replace to="/workbench/import" />} />
                <Route path="/governance" element={<Navigate replace to="/workbench/governance" />} />
                <Route path="/audit" element={<Navigate replace to="/workbench/audit" />} />
                <Route path="/skills/:packageName" element={<SkillDetailPage />} />
                <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
                <Route path="*" element={<Navigate replace to="/" />} />
              </Routes>
            </AppShell>
          </RequireAuthenticatedSession>
        }
      />
    </Routes>
  );
}
