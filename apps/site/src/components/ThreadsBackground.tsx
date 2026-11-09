import { useEffect, useState } from 'react';
import Threads from './ui/Threads';

function readMode(): 'light' | 'dark' {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
}

export function ThreadsBackground() {
  const [mode, setMode] = useState<'light' | 'dark'>(readMode);

  useEffect(() => {
    const observer = new MutationObserver(() => setMode(readMode()));
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme'],
    });
    return () => observer.disconnect();
  }, []);

  const dark = mode === 'dark';

  return (
    <div aria-hidden="true" className="threads-backdrop">
      <Threads
        accentColor={dark ? '#8bb8ff' : '#4c6ca8'}
        amplitude={1.4}
        angle={-8}
        brightness={1.2}
        className="threads-backdrop__canvas"
        color={dark ? '#7cf0a7' : '#2d6b4d'}
        distance={0.45}
        enableMouseInteraction
        fade={0.12}
        fray={0.5}
        lineCount={56}
        opacity={dark ? 0.26 : 0.15}
        parting={0.35}
        seed={2}
        softness={1.5}
        speed={0.45}
        split={0.04}
        taper={0.9}
        thickness={0.55}
        waves={1.05}
      />
    </div>
  );
}
