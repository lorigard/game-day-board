import type { ReactNode } from 'react';

interface ExternalLinkProps {
  /** The full address of the other site, starting with https://. */
  href: string;
  /** The visible link text. */
  children: ReactNode;
}

/**
 * A link to another website that opens in a new browser tab.
 *
 * Why a component instead of a plain <a>: every link that leaves the site
 * needs the same three things, and a shared component means none of them
 * can be forgotten.
 *
 * 1. target="_blank" opens a new tab, so a visitor who follows a link to
 *    GitHub or the data source still has Game Day Board open behind it.
 * 2. rel="noopener noreferrer" is a security habit. "noopener" stops the new
 *    page from reaching back into this tab through window.opener (modern
 *    browsers now do this by default for target="_blank", but older ones did
 *    not). "noreferrer" also keeps our address out of the other site's logs.
 * 3. The hidden "(opens in a new tab)" text is for screen reader users.
 *    Sighted visitors see a new tab appear; a screen reader user only hears
 *    that something changed, so the accessibility guidelines (WCAG 3.2.5) ask
 *    us to say so up front. The "visually-hidden" class in styles.css keeps
 *    the text readable by screen readers but invisible on screen.
 *
 * Links inside the site (team pages, About) use React Router's <Link>
 * instead, and stay in the same tab, which is what visitors expect.
 */
export function ExternalLink({ href, children }: ExternalLinkProps) {
  return (
    <a href={href} target="_blank" rel="noopener noreferrer">
      {children}
      <span className="visually-hidden"> (opens in a new tab)</span>
    </a>
  );
}
