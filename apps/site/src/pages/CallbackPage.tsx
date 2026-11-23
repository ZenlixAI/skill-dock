import { useHandleSignInCallback } from '@logto/react';
import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../lib/auth';

export function CallbackPage() {
  const auth = useAuth();
  if (!auth.enabled) {
    return (
      <div className="registry-state registry-state--empty">
        <h1>Sign-in is not configured</h1>
        <p>This site is running without a Logto account backend.</p>
      </div>
    );
  }
  return <LogtoCallback />;
}

function LogtoCallback() {
  const navigate = useNavigate();
  const { isLoading, error } = useHandleSignInCallback(() => {
    const returnTo = sessionStorage.getItem('skill-dock:return-to') || '/';
    sessionStorage.removeItem('skill-dock:return-to');
    navigate(returnTo, { replace: true });
  });

  useEffect(() => {
    if (error) {
      const timer = window.setTimeout(() => navigate('/', { replace: true }), 2500);
      return () => window.clearTimeout(timer);
    }
    return undefined;
  }, [error, navigate]);

  return (
    <div className="registry-state registry-state--empty" aria-busy={isLoading}>
      <h1>{error ? 'Sign-in failed' : 'Signing in…'}</h1>
      <p>{error ? error.message : 'Finishing the Logto sign-in flow.'}</p>
    </div>
  );
}
