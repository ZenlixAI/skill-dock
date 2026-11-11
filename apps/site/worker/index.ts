import { handleSkillsShRequest, type SkillsShEnv } from './skills-sh';

interface Env extends SkillsShEnv {
  ASSETS: Fetcher;
}

interface ExecutionContext {
  waitUntil(promise: Promise<unknown>): void;
}

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    if (url.pathname === '/api/skills-sh' || url.pathname.startsWith('/api/skills-sh/')) {
      return handleSkillsShRequest(request, env, ctx);
    }
    return env.ASSETS.fetch(request);
  },
};
