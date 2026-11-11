import { Check, Copy, TerminalSquare } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useI18n } from '../../lib/i18n';
import clsx from '../../vendor/clsx';
import { copyText } from '../../lib/clipboard';

interface TerminalProps {
  command: string;
  title?: string;
  className?: string;
  secondary?: string;
}

export function Terminal({ command, title = 'terminal', className, secondary }: TerminalProps) {
  const { content } = useI18n();
  const copy = content.ui.terminal;
  const [copied, setCopied] = useState(false);
  const timeout = useRef<number | undefined>(undefined);

  useEffect(() => {
    return () => {
      if (timeout.current) window.clearTimeout(timeout.current);
    };
  }, []);

  async function copyCommand() {
    const copied = await copyText(command);
    setCopied(copied);
    if (copied) {
      if (timeout.current) window.clearTimeout(timeout.current);
      timeout.current = window.setTimeout(() => setCopied(false), 1800);
    }
  }

  return (
    <div className={clsx('terminal', className)}>
      <div className="terminal__bar">
        <span className="terminal__dots" aria-hidden="true">
          <i />
          <i />
          <i />
        </span>
        <span className="terminal__title">
          <TerminalSquare aria-hidden="true" size={13} strokeWidth={1.8} />
          {title}
        </span>
        <button
          aria-label={copied ? copy.copiedCommand : copy.copyCommand(title)}
          className="terminal__copy"
          onClick={() => void copyCommand()}
          type="button"
        >
          {copied ? <Check aria-hidden="true" size={14} /> : <Copy aria-hidden="true" size={14} />}
          <span>{copied ? copy.copied : copy.copy}</span>
        </button>
      </div>
      <pre className="terminal__body" tabIndex={0}>
        <code>
          {command.split('\n').map((line, index) => (
            <span className="terminal__line" key={`${line}-${index}`}>
              {line || '\u00a0'}
            </span>
          ))}
        </code>
      </pre>
      {secondary ? <div className="terminal__secondary">{secondary}</div> : null}
      <span aria-live="polite" className="sr-only">
        {copied ? copy.announcement : ''}
      </span>
    </div>
  );
}
