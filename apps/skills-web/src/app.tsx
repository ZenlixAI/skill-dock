import { Navigate, Route, Routes } from 'react-router-dom';
import { AppShell } from './components/AppShell';
import { DocumentMeta } from './components/DocumentMeta';
import { RequireAuthenticatedSession, RequireOperatorSession } from './components/RouteGuards';
import { getRouteMeta } from './lib/routeMeta';
import { AuditPage } from './pages/AuditPage';
import { AuthoringFlowPage } from './pages/AuthoringFlowPage';
import { CatalogPage } from './pages/CatalogPage';
import { GovernancePage } from './pages/GovernancePage';
import { ImportPage } from './pages/ImportPage';
import { LoginPage } from './pages/LoginPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { OverviewPage } from './pages/OverviewPage';
import { PrivacyPage } from './pages/PrivacyPage';
import { SkillDetailPage } from './pages/SkillDetailPage';
import { TermsPage } from './pages/TermsPage';
import { VersionDetailPage } from './pages/VersionDetailPage';
import { WorkbenchHomePage } from './pages/WorkbenchHomePage';

export function App() {
  const loginMeta = getRouteMeta('/login');

  return (
    <Routes>
      <Route
        path="/login"
        element={
          <>
            <DocumentMeta description={loginMeta.description} title={loginMeta.title} />
            <LoginPage />
          </>
        }
      />
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
                <Route path="/privacy" element={<PrivacyPage />} />
                <Route path="/terms" element={<TermsPage />} />
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
                <Route path="*" element={<NotFoundPage />} />
              </Routes>
            </AppShell>
          </RequireAuthenticatedSession>
        }
      />
    </Routes>
  );
}
