import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function PrivacyPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="法律"
        title="隐私"
        description="内部注册表会记录保障目录安全和审计变更历史所需的运营数据。"
        tone="legal"
      />

      <SurfaceCard
        title="我们收集的信息"
        description="出于安全、包来源追踪和运营支持需要，注册表会记录相关活动。"
        variant="subtle"
      >
        <div className="stack-column">
          <p>我们会保存运行注册表所需的账号身份、会话状态、包操作记录和 API 活动。</p>
          <p>审计数据会被保留，用于调查发布事件、治理变更和访问控制决策。</p>
          <p>除非平台策略明确允许，否则不要在包元数据中存放个人、客户或受监管数据。</p>
        </div>
      </SurfaceCard>
    </div>
  );
}
