import { AgentStrip } from '../components/AgentStrip';
import { Architecture } from '../components/Architecture';
import { Capabilities } from '../components/Capabilities';
import { Faq } from '../components/Faq';
import { FinalCta } from '../components/FinalCta';
import { Hero } from '../components/Hero';
import { HowItWorks } from '../components/HowItWorks';
import { QuickStart } from '../components/QuickStart';
import { ReleaseDiscipline } from '../components/ReleaseDiscipline';

export function LandingPage() {
  return (
    <div className="landing">
      <Hero />
      <AgentStrip />
      <ReleaseDiscipline />
      <HowItWorks />
      <Capabilities />
      <Architecture />
      <QuickStart />
      <Faq />
      <FinalCta />
    </div>
  );
}
