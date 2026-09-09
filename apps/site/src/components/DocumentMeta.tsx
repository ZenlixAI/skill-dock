import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

interface DocumentMetaProps {
  title: string;
  description: string;
}

export function DocumentMeta({ title, description }: DocumentMetaProps) {
  const location = useLocation();

  useEffect(() => {
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

    let canonical = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]');
    if (!canonical) {
      canonical = document.createElement('link');
      canonical.rel = 'canonical';
      document.head.appendChild(canonical);
    }
    canonical.href = new URL(location.pathname, window.location.origin).href;
  }, [description, location.pathname, title]);

  return null;
}
