import { Route, Routes } from 'react-router-dom';
import { AppShell } from './components/AppShell';
import { FeaturesPage } from './pages/FeaturesPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { OverviewPage } from './pages/OverviewPage';
import { PrivacyPage } from './pages/PrivacyPage';
import { TermsPage } from './pages/TermsPage';

export function App() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <AppShell>
            <OverviewPage />
          </AppShell>
        }
      />
      <Route
        path="/features"
        element={
          <AppShell>
            <FeaturesPage />
          </AppShell>
        }
      />
      <Route
        path="/privacy"
        element={
          <AppShell>
            <PrivacyPage />
          </AppShell>
        }
      />
      <Route
        path="/terms"
        element={
          <AppShell>
            <TermsPage />
          </AppShell>
        }
      />
      <Route
        path="*"
        element={
          <AppShell>
            <NotFoundPage />
          </AppShell>
        }
      />
    </Routes>
  );
}
