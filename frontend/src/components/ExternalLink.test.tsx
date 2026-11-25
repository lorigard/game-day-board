import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ExternalLink } from './ExternalLink';

describe('ExternalLink', () => {
  it('opens in a new tab safely and tells screen reader users so', () => {
    render(<ExternalLink href="https://example.com">Example</ExternalLink>);

    // getByRole finds the link by its accessible name, which is what a screen
    // reader announces, so this also checks that the hidden hint is read out.
    const link = screen.getByRole('link', { name: 'Example (opens in a new tab)' });

    expect(link).toHaveAttribute('href', 'https://example.com');
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
  });
});
