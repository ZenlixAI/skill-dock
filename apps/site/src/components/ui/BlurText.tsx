import { useReducedMotion } from 'motion/react';
import { motion } from 'motion/react';
import type { ReactNode } from 'react';
import clsx from '../../vendor/clsx';

interface BlurTextProps {
  text: string;
  className?: string;
  delay?: number;
  stagger?: number;
}

export function BlurText({ text, className, delay = 0, stagger = 0.03 }: BlurTextProps) {
  const reduce = useReducedMotion();
  const words = text.split(' ');

  if (reduce) {
    return <span className={className}>{text}</span>;
  }

  return (
    <span className={clsx('blur-text', className)}>
      <span className="sr-only">{text}</span>
      {words.map((word, index) => (
        <motion.span
          aria-hidden="true"
          className="blur-text__word"
          initial={{ filter: 'blur(8px)', y: 12 }}
          key={`${word}-${index}`}
          transition={{
            duration: 0.45,
            delay: delay + index * stagger,
            ease: [0.16, 1, 0.3, 1],
          }}
          viewport={{ once: true, amount: 0.2 }}
          whileInView={{ filter: 'blur(0px)', y: 0 }}
        >
          {word}
          {index < words.length - 1 ? '\u00a0' : ''}
        </motion.span>
      ))}
    </span>
  );
}

interface RevealProps {
  children: ReactNode;
  className?: string;
  delay?: number;
}

export function Reveal({ children, className, delay = 0 }: RevealProps) {
  const reduce = useReducedMotion();

  if (reduce) {
    return <div className={className}>{children}</div>;
  }

  return (
    <motion.div
      className={className}
      initial={{ opacity: 0, y: 22 }}
      transition={{ duration: 0.6, delay, ease: [0.16, 1, 0.3, 1] }}
      viewport={{ once: true, amount: 0.15 }}
      whileInView={{ opacity: 1, y: 0 }}
    >
      {children}
    </motion.div>
  );
}
