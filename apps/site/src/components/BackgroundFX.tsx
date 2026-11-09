import { useEffect, useRef } from 'react';

export function BackgroundFX() {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

    let frame = 0;
    let pointerX = 0;
    let pointerY = 0;

    const handlePointerMove = (event: PointerEvent) => {
      pointerX = event.clientX;
      pointerY = event.clientY;
      if (frame) return;

      frame = requestAnimationFrame(() => {
        frame = 0;
        element.style.setProperty('--pointer-x', `${pointerX}px`);
        element.style.setProperty('--pointer-y', `${pointerY}px`);
        element.style.setProperty('--pointer-opacity', '1');
      });
    };

    window.addEventListener('pointermove', handlePointerMove, { passive: true });
    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      if (frame) cancelAnimationFrame(frame);
    };
  }, []);

  return (
    <div aria-hidden="true" className="background-fx" ref={ref}>
      <div className="background-fx__aurora background-fx__aurora--one" />
      <div className="background-fx__aurora background-fx__aurora--two" />
      <div className="background-fx__aurora background-fx__aurora--three" />
      <svg
        className="background-fx__contours"
        preserveAspectRatio="none"
        viewBox="0 0 1600 1000"
      >
        <g>
          {Array.from({ length: 11 }, (_, index) => {
            const y = 70 + index * 92;
            return (
              <path
                d={`M -120 ${y} C 240 ${y - 68}, 460 ${y + 62}, 760 ${y} S 1240 ${y - 58}, 1720 ${y + 44}`}
                key={index}
              />
            );
          })}
        </g>
      </svg>
      <div className="background-fx__grid" />
      <div className="background-fx__pointer" />
      <div className="background-fx__grain" />
    </div>
  );
}
