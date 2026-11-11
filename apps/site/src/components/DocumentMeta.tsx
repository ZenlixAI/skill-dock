import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { getContent, SITE_URL } from '../lib/content';
import { useLocale } from '../lib/i18n';
import { localizePath, stripLocalePrefix } from '../lib/locale';
import { buildStructuredData } from '../lib/structured-data';

interface DocumentMetaProps {
  title: string;
  description: string;
}

export function DocumentMeta({ title, description }: DocumentMetaProps) {
  const location = useLocation();
  const locale = useLocale();

  useEffect(() => {
    const content = getContent(locale);
    document.documentElement.lang = content.htmlLang;
    document.title = title;

    let descriptionTag = document.querySelector('meta[name="description"]');
    if (!descriptionTag) {
      descriptionTag = document.createElement('meta');
      descriptionTag.setAttribute('name', 'description');
      document.head.appendChild(descriptionTag);
    }
    descriptionTag.setAttribute('content', description);

    const meta = (selector: string, attr: string, key: string, content: string) => {
      let tag = document.head.querySelector<HTMLMetaElement>(selector);
      if (!tag) {
        tag = document.createElement('meta');
        tag.setAttribute(attr, key);
        document.head.appendChild(tag);
      }
      tag.content = content;
    };
    meta('meta[property="og:title"]', 'property', 'og:title', title);
    meta('meta[property="og:description"]', 'property', 'og:description', description);
    meta('meta[name="twitter:title"]', 'name', 'twitter:title', title);
    meta('meta[name="twitter:description"]', 'name', 'twitter:description', description);
    meta('meta[property="og:locale"]', 'property', 'og:locale', locale === 'zh' ? 'zh_CN' : 'en_US');
    meta(
      'meta[property="og:locale:alternate"]',
      'property',
      'og:locale:alternate',
      locale === 'zh' ? 'en_US' : 'zh_CN'
    );

    const barePath = stripLocalePrefix(location.pathname);
    const canonicalURL = new URL(localizePath(barePath, locale), SITE_URL).href;

    let canonical = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]');
    if (!canonical) {
      canonical = document.createElement('link');
      canonical.rel = 'canonical';
      document.head.appendChild(canonical);
    }
    canonical.href = canonicalURL;
    meta('meta[property="og:url"]', 'property', 'og:url', canonicalURL);

    const alternates: Array<{ hreflang: string; href: string }> = [
      { hreflang: 'en', href: new URL(localizePath(barePath, 'en'), SITE_URL).href },
      { hreflang: 'zh-CN', href: new URL(localizePath(barePath, 'zh'), SITE_URL).href },
      { hreflang: 'x-default', href: new URL(localizePath(barePath, 'en'), SITE_URL).href },
    ];

    for (const alternate of alternates) {
      let link = document.head.querySelector<HTMLLinkElement>(
        `link[rel="alternate"][hreflang="${alternate.hreflang}"]`
      );
      if (!link) {
        link = document.createElement('link');
        link.rel = 'alternate';
        link.hreflang = alternate.hreflang;
        document.head.appendChild(link);
      }
      link.href = alternate.href;
    }

    const structuredData = document.getElementById('site-structured-data');
    if (structuredData) {
      structuredData.textContent = buildStructuredData(locale);
    }
  }, [description, locale, location.pathname, title]);

  return null;
}
