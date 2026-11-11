import { useI18n } from '../lib/i18n';
import { Reveal } from './ui/BlurText';

export function ReleaseDiscipline() {
  const { content } = useI18n();
  const copy = content.release;

  return (
    <section aria-labelledby="release-title" className="section release-discipline">
      <Reveal className="release-discipline__intro">
        <p className="eyebrow">{copy.eyebrow}</p>
        <h2 id="release-title">{copy.title}</h2>
        <p className="lede">{copy.lede}</p>
      </Reveal>

      <Reveal delay={0.05}>
        <div className="release-points">
          {copy.points.map(({ icon: Icon, title, body }) => (
            <div className="release-point" key={title}>
              <span className="release-point__icon">
                <Icon aria-hidden="true" size={18} strokeWidth={1.8} />
              </span>
              <h3>{title}</h3>
              <p>{body}</p>
            </div>
          ))}
        </div>
      </Reveal>
    </section>
  );
}
