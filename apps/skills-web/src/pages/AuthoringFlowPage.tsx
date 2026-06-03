import { Link, useSearchParams } from 'react-router-dom';
import { listAuditEvents, listGovernanceChanges, listSkills } from '../api/client';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { useAsyncData } from '../lib/useAsyncData';
import { buildReleaseHandoffPath, readReleaseHandoff } from '../lib/releaseHandoff';

export function AuthoringFlowPage() {
  const [searchParams] = useSearchParams();
  const handoff = readReleaseHandoff(searchParams);
  const hasHandoff = Boolean(handoff.manifestDigest || handoff.bundlePath || handoff.workspacePath);

  const skills = useAsyncData(() => listSkills({ includeRetired: true, pageSize: 8 }), []);
  const governance = useAsyncData(() => listGovernanceChanges({ pageSize: 8 }), []);
  const audit = useAsyncData(() => listAuditEvents({ pageSize: 8 }), []);

  const releaseTarget =
    handoff.packageName && handoff.version
      ? buildReleaseHandoffPath(handoff, `/skills/${encodeURIComponent(handoff.packageName)}/versions/${encodeURIComponent(handoff.version)}`)
      : null;

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Platform Authoring"
        title="把外部导入、发布登记、治理审阅接成上层平台里的一个作者工作台。"
        description="这个页面不是单独 registry 页签，而是模拟更上层 authoring / platform 入口该怎么承接 skills-web 的行为。"
        tone="backstage"
      />

      <div className="flow-ribbon">
        <article className={hasHandoff ? 'flow-node flow-node--active' : 'flow-node'}>
          <span>Intake</span>
          <strong>Source to Prepare</strong>
          <p>从外部 skill、local 目录或平台 authoring 输入开始。</p>
        </article>
        <article className={hasHandoff ? 'flow-node flow-node--active' : 'flow-node'}>
          <span>Review</span>
          <strong>Validate to Pack to Release</strong>
          <p>pack 结果变成 handoff，直接进入 release review。</p>
        </article>
        <article className="flow-node">
          <span>Operate</span>
          <strong>Governance to Audit</strong>
          <p>发布后继续看治理动作、失败原因和追溯链路。</p>
        </article>
      </div>

      <div className="metric-grid">
        <div className="metric-card">
          <strong>{skills.data?.items.length || '—'}</strong>
          <span>Registry packages</span>
        </div>
        <div className="metric-card">
          <strong>{audit.data?.items.filter((item) => item.action.startsWith('packaging.')).length || '—'}</strong>
          <span>Packaging actions</span>
        </div>
        <div className="metric-card">
          <strong>{audit.data?.items.filter((item) => item.action === 'release.record').length || '—'}</strong>
          <span>Release records</span>
        </div>
        <div className="metric-card">
          <strong>{governance.data?.items.length || '—'}</strong>
          <span>Governance changes</span>
        </div>
      </div>

      {hasHandoff ? (
        <SurfaceCard
          title="Active Release Handoff"
          description="这是从 `Import & Package` 带过来的 authoring handoff，平台层可以直接承接到 release review。"
          variant="rail"
          action={
            releaseTarget ? (
              <Link className="button button--primary" to={releaseTarget}>
                Continue release review
              </Link>
            ) : null
          }
        >
          <dl className="property-list">
            <div>
              <dt>package</dt>
              <dd>{handoff.packageName || '—'}</dd>
            </div>
            <div>
              <dt>version</dt>
              <dd>{handoff.version || '—'}</dd>
            </div>
            <div>
              <dt>workspace</dt>
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
          title="1. Intake & Normalize"
          description="把社区 / 本地 / 平台 authoring 源接进 package 工作流。"
          variant="console"
          action={
            <Link className="button button--secondary" to="/import">
              Open import workflow
            </Link>
          }
        >
          <div className="list-stack">
            <article className="list-card">
              <strong>External source intake</strong>
              <p>进入 prepare，生成 workspace、warningDetails、importReport。</p>
            </article>
            <article className="list-card">
              <strong>Validation review</strong>
              <p>明确哪些 warning 是平台自动修复，哪些必须由作者决定是否接受。</p>
            </article>
          </div>
        </SurfaceCard>

        <SurfaceCard
          title="2. Release Review"
          description="pack 的产物不直接丢给用户，而是进入一个 release review 承接点。"
          variant="console"
          action={
            releaseTarget ? (
              <Link className="button button--secondary" to={releaseTarget}>
                Open version detail
              </Link>
            ) : null
          }
        >
          <div className="stack-column">
            <div className="pill-row">
              <StatusPill tone={hasHandoff ? 'good' : 'warn'}>
                {hasHandoff ? 'handoff ready' : 'waiting for pack result'}
              </StatusPill>
              {handoff.publishRegistry ? <StatusPill tone="accent">{handoff.publishRegistry}</StatusPill> : null}
            </div>
            <p className="support-copy">
              这里承接 `manifestDigest`、`components`、`publishRegistry`，然后进入 `release-record` 写接口。
            </p>
          </div>
        </SurfaceCard>

        <SurfaceCard
          title="3. Governance & Audit"
          description="发布之后，平台层继续把 retire / deprecate / yank 以及审计线索暴露出来。"
          action={
            <div className="authoring-actions">
              <Link className="button button--secondary" to="/governance">
                Governance
              </Link>
              <Link className="button button--secondary" to="/audit">
                Audit
              </Link>
            </div>
          }
        >
          <div className="list-stack">
            <article className="list-card">
              <strong>Governance surface</strong>
              <p>从产品入口看得到 skill / version 当前是否退役、弃用、撤回。</p>
            </article>
            <article className="list-card">
              <strong>Audit surface</strong>
              <p>从平台层回查打包、发布登记、治理动作是否成功，以及失败的真实后端原因。</p>
            </article>
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
