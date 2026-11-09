import { RELEASE_POINTS } from '../lib/content';
import { Reveal } from './ui/BlurText';

export function ReleaseDiscipline() {
  return (
    <section aria-labelledby="release-title" className="section release-discipline">
      <Reveal className="release-discipline__intro">
        <p className="eyebrow">Release discipline</p>
        <h2 id="release-title">A git ref is not a release.</h2>
        <p className="lede">
          A skill can change how an agent works. It can also ship scripts. If the source is a
          branch, the same name can point at different files tomorrow. Skill Dock gives each
          bundle an exact version, an integrity hash, and a record of whether it is installable.
        </p>
      </Reveal>

      <Reveal delay={0.05}>
        <div className="release-points">
          {RELEASE_POINTS.map(({ icon: Icon, title, body }) => (
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
