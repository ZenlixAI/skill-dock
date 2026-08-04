import { FormEvent, useState } from 'react';
import { Link, Navigate, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { getRegistryClientConfig, loginSession } from '../api/client';
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
    <main className="login-page">
      <section aria-labelledby="login-title" className="login-layout">
        <div className="login-context">
          <span className="login-context__eyebrow">控制台登录</span>
          <h1 id="login-title">进入技能注册表控制台</h1>
          <p className="login-context__description">导入、编写、治理和审计流程统一收口到这个控制台入口。</p>
        </div>

        <section aria-label="登录表单区域" className="login-panel">
          <div className="login-panel__header">
            <p className="login-panel__eyebrow">身份验证</p>
            <h2>登录</h2>
            <p>使用你的仓库账号继续访问控制台。</p>
          </div>

          <form className="login-form" onSubmit={handleSubmit}>
            <label className="field login-form__field" htmlFor="login-username">
              <span>用户名</span>
              <input
                autoComplete="username"
                id="login-username"
                onChange={(event) => setUsername(event.target.value)}
                value={username}
              />
            </label>
            <label className="field login-form__field" htmlFor="login-password">
              <span>密码</span>
              <input
                autoComplete="current-password"
                id="login-password"
                onChange={(event) => setPassword(event.target.value)}
                type="password"
                value={password}
              />
            </label>
            {error ? (
              <div aria-live="polite" className="error-box login-form__error" role="status">
                {error}
              </div>
            ) : null}
            <button className="button button--primary login-form__submit" disabled={submitting} type="submit">
              {submitting ? '登录中…' : '登录'}
            </button>
          </form>

          <div className="login-panel__footer">
            <span>继续即表示你将进入内部控制台。</span>
            <div className="login-panel__links">
              <Link to="/privacy">隐私</Link>
              <Link to="/terms">条款</Link>
            </div>
          </div>
        </section>
      </section>
    </main>
  );
}
