import { Navigate, Route, Routes } from 'react-router-dom';
import { AppShell } from './components/AppShell';
import { LocaleProvider } from './lib/i18n';
import { LandingPage } from './pages/LandingPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { PrivacyPage } from './pages/PrivacyPage';
import { SkillsDirectoryPage } from './pages/SkillsDirectoryPage';
import { TermsPage } from './pages/TermsPage';

export function App() {
  return (
    <LocaleProvider>
      <AppShell>
        <Routes>
          <Route element={<LandingPage />} path="/" />
          <Route element={<LandingPage />} path="/zh" />
          <Route element={<SkillsDirectoryPage />} path="/registry" />
          <Route element={<SkillsDirectoryPage />} path="/zh/registry" />
          <Route element={<Navigate replace to="/#how-it-works" />} path="/features" />
          <Route element={<Navigate replace to="/zh#how-it-works" />} path="/zh/features" />
          <Route element={<PrivacyPage />} path="/privacy" />
          <Route element={<PrivacyPage />} path="/zh/privacy" />
          <Route element={<TermsPage />} path="/terms" />
          <Route element={<TermsPage />} path="/zh/terms" />
          <Route element={<NotFoundPage />} path="*" />
        </Routes>
      </AppShell>
    </LocaleProvider>
  );
}
