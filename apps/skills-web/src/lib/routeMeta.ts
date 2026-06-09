export type RouteSection = 'portal' | 'console' | 'auth' | 'legal' | 'not-found';

export interface RouteMeta {
  title: string;
  description: string;
  section: RouteSection;
}

const skillDetailPattern = /^\/skills\/[^/]+$/;
const versionDetailPattern = /^\/skills\/[^/]+\/versions\/[^/]+$/;

export function getRouteMeta(pathname: string): RouteMeta {
  if (pathname === '/') {
    return {
      title: '技能注册表门户',
      description: '在公司注册表中搜索、评估并安装技能。',
      section: 'portal',
    };
  }

  if (pathname === '/catalog') {
    return {
      title: '目录 · 技能注册表',
      description: '浏览可搜索的精选技能，并查看可直接安装的元数据。',
      section: 'portal',
    };
  }

  if (skillDetailPattern.test(pathname)) {
    return {
      title: '技能详情 · 技能注册表',
      description: '查看注册表技能的包详情、来源信息和安装说明。',
      section: 'portal',
    };
  }

  if (versionDetailPattern.test(pathname)) {
    return {
      title: '版本详情 · 技能注册表',
      description: '查看发布元数据并安装指定的技能版本。',
      section: 'portal',
    };
  }

  if (pathname === '/workbench') {
    return {
      title: '控制台 · 技能注册表',
      description: '处理导入、编写、治理和审计流程。',
      section: 'console',
    };
  }

  if (pathname === '/workbench/import') {
    return {
      title: '导入 · 技能注册表',
      description: '准备并发布新的技能到公司注册表。',
      section: 'console',
    };
  }

  if (pathname === '/workbench/authoring') {
    return {
      title: '编写 · 技能注册表',
      description: '在发布前创建并完善技能包内容。',
      section: 'console',
    };
  }

  if (pathname === '/workbench/governance') {
    return {
      title: '治理 · 技能注册表',
      description: '应用治理策略并审查技能的运营变更。',
      section: 'console',
    };
  }

  if (pathname === '/workbench/audit') {
    return {
      title: '审计 · 技能注册表',
      description: '查看注册表中的活动历史和运营证据。',
      section: 'console',
    };
  }

  if (pathname === '/login') {
    return {
      title: '登录 · 技能注册表',
      description: '登录内部技能注册表。',
      section: 'auth',
    };
  }

  if (pathname === '/privacy') {
    return {
      title: '隐私 · 技能注册表',
      description: '内部注册表如何处理操作人和目录活动数据。',
      section: 'legal',
    };
  }

  if (pathname === '/terms') {
    return {
      title: '条款 · 技能注册表',
      description: '使用内部技能注册表门户和控制台时适用的规则。',
      section: 'legal',
    };
  }

  return {
    title: '页面不存在 · 技能注册表',
    description: '请求的页面不存在。请返回门户概览或技能目录。',
    section: 'not-found',
  };
}
