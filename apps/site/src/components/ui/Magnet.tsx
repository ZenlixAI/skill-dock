import { motion, useReducedMotion } from 'motion/react';
import { useRef, useState } from 'react';
import type { PropsWithChildren } from 'react';
import clsx from '../../vendor/clsx';

interface MagnetProps extends PropsWithChildren {
  className?: string;
  strength?: number;
}

export function Magnet({ children, className, strength = 8 }: MagnetProps) {
  const ref = useRef<HTMLDivElement>(null);
  const reduce = useReducedMotion();
  const [offset, setOffset] = useState({ x: 0, y: 0 });

  function handlePointerMove(event: React.PointerEvent<HTMLDivElement>) {
    if (reduce || !ref.current) return;
    const rect = ref.current.getBoundingClientRect();
    const x = ((event.clientX - rect.left) / rect.width - 0.5) * 2;
    const y = ((event.clientY - rect.top) / rect.height - 0.5) * 2;
    setOffset({ x: x * strength, y: y * strength });
  }

  return (
    <motion.div
      animate={{ x: offset.x, y: offset.y }}
      className={clsx('magnet', className)}
      onPointerLeave={() => setOffset({ x: 0, y: 0 })}
      onPointerMove={handlePointerMove}
      ref={ref}
      transition={{ type: 'spring', stiffness: 260, damping: 22, mass: 0.4 }}
    >
      {children}
    </motion.div>
  );
}

interface TiltProps extends PropsWithChildren {
  className?: string;
  max?: number;
}

export function Tilt({ children, className, max = 4 }: TiltProps) {
  const ref = useRef<HTMLDivElement>(null);
  const reduce = useReducedMotion();
  const [rotation, setRotation] = useState({ x: 0, y: 0 });

  function handlePointerMove(event: React.PointerEvent<HTMLDivElement>) {
    if (reduce || !ref.current) return;
    const rect = ref.current.getBoundingClientRect();
    const x = (event.clientY - rect.top) / rect.height - 0.5;
    const y = (event.clientX - rect.left) / rect.width - 0.5;
    setRotation({ x: x * -max, y: y * max });
  }

  return (
    <motion.div
      animate={{ rotateX: rotation.x, rotateY: rotation.y }}
      className={clsx('tilt', className)}
      onPointerLeave={() => setRotation({ x: 0, y: 0 })}
      onPointerMove={handlePointerMove}
      ref={ref}
      style={{ transformPerspective: 1000 }}
      transition={{ type: 'spring', stiffness: 180, damping: 20 }}
    >
      {children}
    </motion.div>
  );
}
