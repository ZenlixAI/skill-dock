import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function TermsPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="法律"
        title="条款"
        description="注册表仅用于经批准的内部技能发现、安装和运营流程。"
        tone="legal"
      />

      <SurfaceCard
        title="使用要求"
        description="这些条款概述了门户和控制台的基础规则。"
        variant="subtle"
      >
        <div className="stack-column">
          <p>只发布你有权管理的包，并在分发给其他团队前复核安装命令。</p>
          <p>治理和审计视图属于产品界面的一部分，不要使用未文档化的运营流程绕过它们。</p>
          <p>所有使用行为仍需遵守公司的安全、合规和软件供应链政策。</p>
        </div>
      </SurfaceCard>
    </div>
  );
}
