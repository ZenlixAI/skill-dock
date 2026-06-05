import { Link, useSearchParams } from 'react-router-dom';
import { getRegistryClientConfig, getSession, listAuditEvents, listGovernanceChanges, listSkills } from '../api/client';
import { canMutateCatalog, describeSessionPosture, describeSessionScope } from '../api/session';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { useAsyncData } from '../lib/useAsyncData';
import { buildAuditLink } from '../lib/auditLink';
import { buildReleaseHandoffPath, readReleaseHandoff } from '../lib/releaseHandoff';

export function AuthoringFlowPage() {
  const [searchParams] = useSearchParams();
  const handoff = readReleaseHandoff(searchParams);
  const hasHandoff = Boolean(handoff.manifestDigest || handoff.bundlePath || handoff.workspacePath);
  const config = getRegistryClientConfig();

  const session = useAsyncData(() => getSession(), [config.mode, config.baseUrl]);
  const skills = useAsyncData(() => listSkills({ includeRetired: true, pageSize: 8 }), []);
  const governance = useAsyncData(() => listGovernanceChanges({ pageSize: 8 }), []);
  const audit = useAsyncData(() => listAuditEvents({ pageSize: 8 }), []);

  const releaseTarget =
    handoff.packageName && handoff.version
      ? buildReleaseHandoffPath(handoff, `/skills/${encodeURIComponent(handoff.packageName)}/versions/${encodeURIComponent(handoff.version)}`)
      : null;
  const auditTarget = buildAuditLink({
    packageName: handoff.packageName,
    version: handoff.version,
    actor: session.data?.actor ?? config.actor,
  });

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="平台作者台"
        title="把外部导入、发布登记、治理审阅接成上层平台里的一个作者工作台。"
        description="这个页面不是单独 registry 页签，而是模拟更上层 authoring / platform 入口该怎么承接 skills-web 的行为。"
        tone="backstage"
      />

      <div className="flow-ribbon">
        <article className={hasHandoff ? 'flow-node flow-node--active' : 'flow-node'}>
          <span>Intake</span>
          <strong>来源接入到 Prepare</strong>
          <p>从外部 skill、local 目录或平台 authoring 输入开始。</p>
        </article>
        <article className={hasHandoff ? 'flow-node flow-node--active' : 'flow-node'}>
          <span>Review</span>
          <strong>Validate 到 Pack 再到 Release</strong>
          <p>pack 结果变成 handoff，直接进入 release review。</p>
        </article>
        <article className="flow-node">
          <span>Operate</span>
          <strong>Governance to Audit</strong>
          <p>发布后继续看治理动作、失败原因和追溯链路。</p>
        </article>
      </div>

      <div className="context-strip">
        <StatusPill tone={config.mode === 'real' ? 'good' : 'warn'}>{config.mode === 'real' ? '真实模式' : '模拟模式'}</StatusPill>
        <StatusPill tone={canMutateCatalog(session.data) ? 'accent' : 'default'}>
          {session.loading ? '会话加载中' : session.error ? '会话不可用' : describeSessionPosture(session.data)}
        </StatusPill>
        {session.data ? <StatusPill>{session.data.actor}</StatusPill> : null}
        {session.data && describeSessionScope(session.data) ? <StatusPill>{describeSessionScope(session.data)}</StatusPill> : null}
        {hasHandoff && handoff.packageName ? <StatusPill tone="accent">{handoff.packageName}</StatusPill> : null}
        {hasHandoff && handoff.version ? <StatusPill>{handoff.version}</StatusPill> : null}
      </div>

      <div className="metric-grid">
        <div className="metric-card">
          <strong>{skills.data?.items.length || '—'}</strong>
          <span>Registry 包</span>
        </div>
        <div className="metric-card">
          <strong>{audit.data?.items.filter((item) => item.action.startsWith('packaging.')).length || '—'}</strong>
          <span>打包动作</span>
        </div>
        <div className="metric-card">
          <strong>{audit.data?.items.filter((item) => item.action === 'release.record').length || '—'}</strong>
          <span>发布登记</span>
        </div>
        <div className="metric-card">
          <strong>{governance.data?.items.length || '—'}</strong>
          <span>Governance changes</span>
        </div>
      </div>

      <SurfaceCard
        title="写操作与审计上下文"
        description="作者工作台需要知道当前写权限、运行环境，以及 mutation 完成后应该落到哪条 audit 线索上。"
        variant="console"
        action={
          <Link className="button button--secondary" to={auditTarget}>
            打开定向审计
          </Link>
        }
      >
        <dl className="property-list">
          <div>
            <dt>运行模式</dt>
            <dd>{config.mode}</dd>
          </div>
          <div>
            <dt>基础地址</dt>
            <dd>{config.baseUrl || '同源运行时'}</dd>
          </div>
          <div>
            <dt>会话态势</dt>
            <dd>{session.loading ? '正在加载会话…' : session.error ? '会话不可用' : describeSessionPosture(session.data)}</dd>
          </div>
          <div>
            <dt>actor</dt>
            <dd>{session.data?.actor || config.actor || '—'}</dd>
          </div>
          <div>
            <dt>当前包 / 版本焦点</dt>
            <dd>{handoff.packageName && handoff.version ? `${handoff.packageName} · ${handoff.version}` : '当前还没有激活的发布交接。'}</dd>
          </div>
        </dl>
      </SurfaceCard>

      {hasHandoff ? (
        <SurfaceCard
          title="当前发布交接"
          description="这是从 `Import & Package` 带过来的 authoring handoff，平台层可以直接承接到 release review。"
          variant="rail"
          action={
            releaseTarget ? (
              <Link className="button button--primary" to={releaseTarget}>
                继续发布审阅
              </Link>
            ) : null
          }
        >
          <dl className="property-list">
            <div>
              <dt>包名</dt>
              <dd>{handoff.packageName || '—'}</dd>
            </div>
            <div>
              <dt>version</dt>
              <dd>{handoff.version || '—'}</dd>
            </div>
            <div>
              <dt>工作目录</dt>
              <dd>{handoff.workspacePath || '—'}</dd>
            </div>
            <div>
              <dt>bundle path</dt>
              <dd>{handoff.bundlePath || '—'}</dd>
            </div>
            <div>
              <dt>manifest digest</dt>
              <dd>{handoff.manifestDigest || '—'}</dd>
            </div>
            <div>
              <dt>components</dt>
              <dd>{handoff.components?.join(', ') || '—'}</dd>
            </div>
          </dl>
        </SurfaceCard>
      ) : null}

      <div className="authoring-grid">
        <SurfaceCard
          title="1. 接入与归一化"
          description="把社区 / 本地 / 平台 authoring 源接进 package 工作流。"
          variant="console"
          action={
            <Link className="button button--secondary" to="/workbench/import">
              打开导入流程
            </Link>
          }
        >
          <div className="list-stack">
            <article className="list-card">
              <strong>外部来源接入</strong>
              <p>进入 prepare，生成 workspace、warningDetails、importReport。</p>
            </article>
            <article className="list-card">
              <strong>校验审阅</strong>
              <p>明确哪些 warning 是平台自动修复，哪些必须由作者决定是否接受。</p>
            </article>
          </div>
        </SurfaceCard>

        <SurfaceCard
          title="2. 发布审阅"
          description="pack 的产物不直接丢给用户，而是进入一个 release review 承接点。"
          variant="console"
          action={
            releaseTarget ? (
              <Link className="button button--secondary" to={releaseTarget}>
                打开版本详情
              </Link>
            ) : null
          }
        >
          <div className="stack-column">
            <div className="pill-row">
              <StatusPill tone={hasHandoff ? 'good' : 'warn'}>
                {hasHandoff ? '交接就绪' : '等待 pack 结果'}
              </StatusPill>
              {handoff.publishRegistry ? <StatusPill tone="accent">{handoff.publishRegistry}</StatusPill> : null}
            </div>
            <p className="support-copy">
              这里承接 `manifestDigest`、`components`、`publishRegistry`，然后进入 `release-record` 写接口。
            </p>
          </div>
        </SurfaceCard>

        <SurfaceCard
          title="3. 治理与审计"
          description="发布之后，平台层继续把 retire / deprecate / yank 以及审计线索暴露出来。"
          action={
            <div className="authoring-actions">
              <Link className="button button--secondary" to="/workbench/governance">
                治理
              </Link>
              <Link className="button button--secondary" to="/workbench/audit">
                审计
              </Link>
            </div>
          }
        >
          <div className="list-stack">
            <article className="list-card">
              <strong>治理视图</strong>
              <p>从产品入口看得到 skill / version 当前是否退役、弃用、撤回。</p>
            </article>
            <article className="list-card">
              <strong>审计视图</strong>
              <p>从平台层回查打包、发布登记、治理动作是否成功，以及失败的真实后端原因。</p>
            </article>
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
