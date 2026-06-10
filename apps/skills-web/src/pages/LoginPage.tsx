import { FormEvent, useState } from 'react';
import { Navigate, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { getRegistryClientConfig, loginSession } from '../api/client';
import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';
import { useRegistrySession } from '../session/SessionProvider';

export function LoginPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { session, refreshSession } = useRegistrySession();
  const config = getRegistryClientConfig();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const next = searchParams.get('next') || '/';
  const forceLogoutLanding =
    typeof location.state === 'object' &&
    location.state !== null &&
    'forceLogoutLanding' in location.state &&
    location.state.forceLogoutLanding === true;

  if (config.mode === 'mock' || config.hasToken || config.role) {
    return <Navigate replace to="/" />;
  }

  if (!forceLogoutLanding && !session.loading && session.data) {
    return <Navigate replace to={next} />;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await loginSession(username, password);
      refreshSession();
      navigate(next, { replace: true });
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : String(submitError));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="登录"
        title="登录"
        description="使用你的仓库账号登录。"
        tone="backstage"
      />

      <SurfaceCard title="登录" variant="console">
        <form className="stack-column" onSubmit={handleSubmit}>
          <label className="field">
            <span>用户名</span>
            <input autoComplete="username" onChange={(event) => setUsername(event.target.value)} value={username} />
          </label>
          <label className="field">
            <span>密码</span>
            <input
              autoComplete="current-password"
              onChange={(event) => setPassword(event.target.value)}
              type="password"
              value={password}
            />
          </label>
          {error ? <div className="error-box">{error}</div> : null}
          <div className="authoring-actions">
            <button className="button button--primary" disabled={submitting} type="submit">
              {submitting ? '登录中…' : '登录'}
            </button>
          </div>
        </form>
      </SurfaceCard>
    </div>
  );
}
