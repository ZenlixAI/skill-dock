import { useEffect } from 'react';

interface DocumentMetaProps {
  title: string;
  description: string;
}

export function DocumentMeta({ title, description }: DocumentMetaProps) {
  useEffect(() => {
    document.title = title;

    let descriptionTag = document.querySelector('meta[name="description"]');
    if (!descriptionTag) {
      descriptionTag = document.createElement('meta');
      descriptionTag.setAttribute('name', 'description');
      document.head.appendChild(descriptionTag);
    }
    descriptionTag.setAttribute('content', description);
  }, [description, title]);

  return null;
}
