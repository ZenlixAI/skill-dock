import { getContent, LICENSE_URL, REPO_URL, SITE_URL } from './content';
import type { Locale } from './locale';

export function buildStructuredData(locale: Locale): string {
  const content = getContent(locale);

  return JSON.stringify({
    '@context': 'https://schema.org',
    '@graph': [
      {
        '@type': 'WebSite',
        '@id': `${SITE_URL}/#website`,
        url: `${SITE_URL}/`,
        name: 'Skill Dock',
        description: content.hero.lede,
        inLanguage: content.htmlLang,
      },
      {
        '@type': 'SoftwareApplication',
        '@id': `${SITE_URL}/#software`,
        name: 'Skill Dock',
        url: `${SITE_URL}/`,
        applicationCategory: 'DeveloperApplication',
        operatingSystem: 'Linux, macOS, Windows',
        description: content.meta.home.description,
        license: LICENSE_URL,
        isAccessibleForFree: true,
        sameAs: [REPO_URL],
      },
      {
        '@type': 'FAQPage',
        '@id': `${SITE_URL}/#faq`,
        mainEntity: content.faq.items.map((item) => ({
          '@type': 'Question',
          name: item.question,
          acceptedAnswer: {
            '@type': 'Answer',
            text: item.answer,
          },
        })),
      },
    ],
  });
}
