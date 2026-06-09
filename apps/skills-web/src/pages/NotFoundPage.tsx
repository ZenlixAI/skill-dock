import { Link } from 'react-router-dom';
import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function NotFoundPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="404"
        title="页面不存在"
        description="你请求的路由不在注册表中。请使用下方入口返回。"
        tone="legal"
      />

      <SurfaceCard
        action={
          <div className="button-row">
            <Link className="button button--primary" to="/">
              返回概览
            </Link>
            <Link className="button button--secondary" to="/catalog">
              查看目录
            </Link>
          </div>
        }
        title="返回门户"
        description="从概览回到产品入口，或直接进入目录继续查找。"
        variant="subtle"
      >
        <div className="stack-column">
          <div className="empty-box">未知路由不再静默重定向。此页会保留上下文，并提供明确的返回入口。</div>
        </div>
      </SurfaceCard>
    </div>
  );
}
