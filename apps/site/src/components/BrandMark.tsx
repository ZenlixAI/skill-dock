import clsx from '../vendor/clsx';

interface BrandMarkProps {
  className?: string;
}

export function BrandMark({ className }: BrandMarkProps) {
  return (
    <svg
      aria-hidden="true"
      className={clsx('brand-mark', className)}
      fill="none"
      viewBox="0 0 32 32"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M16 4.5 27 10.5 16 16.5 5 10.5 16 4.5Z"
        fill="currentColor"
        opacity="0.95"
      />
      <path
        d="M5 16 16 22 27 16"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.7"
      />
      <path
        d="M5 21.5 16 27.5 27 21.5"
        opacity="0.5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.7"
      />
    </svg>
  );
}
