import { describe, expect, it } from 'vitest';
import { getRouteMeta } from './routeMeta';

describe('getRouteMeta', () => {
  it('returns product metadata for the home page', () => {
    expect(getRouteMeta('/')).toEqual({
      title: 'Skill Dock · A self-hosted registry for agent skills',
      description:
        'Skill Dock is a self-hosted registry for agent skills. Store bundles in your npm registry, publish exact versions, resolve installs, and govern releases.',
      section: 'portal',
    });
  });

  it('returns legal metadata for the privacy and terms pages', () => {
    expect(getRouteMeta('/privacy')).toEqual({
      title: 'Privacy · Skill Dock',
      description: 'How the Skill Dock website handles site usage data.',
      section: 'legal',
    });
    expect(getRouteMeta('/terms')).toEqual({
      title: 'Terms · Skill Dock',
      description: 'Terms for using the Skill Dock website and related materials.',
      section: 'legal',
    });
  });

  it('returns not-found metadata for unknown routes', () => {
    expect(getRouteMeta('/catalog')).toEqual({
      title: 'Page not found · Skill Dock',
      description: 'The requested page is not available. Return home or read the quick start.',
      section: 'not-found',
    });
  });

  it('returns Chinese metadata for /zh routes', () => {
    expect(getRouteMeta('/zh')).toEqual({
      title: 'Skill Dock · 面向 Agent 技能的自托管 registry',
      description:
        'Skill Dock 是面向 Agent 技能的自托管 registry。将 bundle 存放在你的 npm registry 中，发布精确版本，解析安装，并治理每一次发布。',
      section: 'portal',
    });
    expect(getRouteMeta('/zh/privacy')).toEqual({
      title: '隐私 · Skill Dock',
      description: 'Skill Dock 网站如何处理站点使用数据。',
      section: 'legal',
    });
    expect(getRouteMeta('/zh/unknown')).toEqual({
      title: '页面未找到 · Skill Dock',
      description: '请求的页面不存在。请返回首页或阅读快速开始。',
      section: 'not-found',
    });
  });
});
