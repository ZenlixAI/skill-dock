import type { Locale } from './locale';

export function formatInstalls(value: number, locale: Locale = 'en'): string {
  return new Intl.NumberFormat(locale === 'zh' ? 'zh-CN' : 'en-US', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value);
}
