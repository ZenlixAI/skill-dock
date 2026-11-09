import { Navigate, Route, Routes } from 'react-router-dom';
import { AppShell } from './components/AppShell';
import { LandingPage } from './pages/LandingPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { PrivacyPage } from './pages/PrivacyPage';
import { TermsPage } from './pages/TermsPage';

export function App() {
  return (
    <AppShell>
      <Routes>
        <Route element={<LandingPage />} path="/" />
        <Route element={<Navigate replace to="/#how-it-works" />} path="/features" />
        <Route element={<PrivacyPage />} path="/privacy" />
        <Route element={<TermsPage />} path="/terms" />
        <Route element={<NotFoundPage />} path="*" />
      </Routes>
    </AppShell>
  );
}
