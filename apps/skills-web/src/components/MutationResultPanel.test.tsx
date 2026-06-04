import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { RegistryHttpError } from '../api/httpClient';
import {
  inferMutationFailureOutcome,
  MutationResultPanel,
  readMutationResultSearchParams,
} from './MutationResultPanel';

afterEach(() => {
  cleanup();
});

describe('MutationResultPanel', () => {
  it('renders failure details and an audit deep link', () => {
    render(
      <MemoryRouter>
        <MutationResultPanel
          result={{
            action: 'pack',
            outcome: 'failed',
            message: 'bundle digest mismatched the uploaded archive',
            requestId: 'req_pack_123',
            packageName: '@platform.skill/pdf-skill',
            version: '1.2.0',
            actor: 'svc-publisher',
          }}
        />
      </MemoryRouter>
    );

    expect(screen.getByText('pack')).toBeInTheDocument();
    expect(screen.getByText('failure')).toBeInTheDocument();
    expect(screen.getByText('bundle digest mismatched the uploaded archive')).toBeInTheDocument();
    expect(screen.getByText('Request ID: req_pack_123')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open audit trail' })).toHaveAttribute(
      'href',
      '/audit?package=%40platform.skill%2Fpdf-skill&version=1.2.0&outcome=failed&actor=svc-publisher'
    );
  });

  it('renders nothing when no mutation result is present', () => {
    const { container } = render(
      <MemoryRouter>
        <MutationResultPanel result={null} />
      </MemoryRouter>
    );

    expect(container).toBeEmptyDOMElement();
  });

  it('round-trips forbidden mutation outcomes from URL state', () => {
    const result = readMutationResultSearchParams(
      new URLSearchParams(
        'mutation=retire&mutationOutcome=forbidden&mutationMessage=blocked&mutationPackage=%40platform.skill%2Fpdf-skill'
      )
    );

    expect(result).toMatchObject({
      action: 'retire',
      outcome: 'forbidden',
      message: 'blocked',
      packageName: '@platform.skill/pdf-skill',
    });
  });

  it('infers auth-specific audit outcomes from HTTP status', () => {
    expect(
      inferMutationFailureOutcome(
        new RegistryHttpError('unauthorized', 401, {
          error: { message: 'unauthorized' },
        })
      )
    ).toBe('unauthorized');
    expect(
      inferMutationFailureOutcome(
        new RegistryHttpError('forbidden', 403, {
          error: { message: 'forbidden' },
        })
      )
    ).toBe('forbidden');
    expect(inferMutationFailureOutcome(new Error('boom'))).toBe('failed');
  });
});
