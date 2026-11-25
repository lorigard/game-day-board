import { ExternalLink } from '../components/ExternalLink';
import { REPO_URL } from '../components/Layout';

/**
 * Explains what the site does and, just as plainly, what it does not do yet
 * (FR-11). Keep this page in step with what is actually built.
 */
export function AboutPage() {
  return (
    <article className="prose">
      <h1>About Game Day Board</h1>
      <p>
        Game Day Board shows Big 12 and SEC college football games and scores, week by week, with a page for each team
        showing its schedule, results, and record. It opens on the current week, and you can step back through the
        season or ahead to games still to come.
      </p>
      <p>
        It is a learning project. I built it to practice a stack that is new to me: a Go back end, a React front end, and
        Google Cloud hosting, with automated tests and a deployment pipeline. It was built with the help of Claude Code,
        an AI coding assistant.
      </p>

      <h2>How it works</h2>
      <ul>
        <li>
          A small Go service runs on a schedule. It fetches games from{' '}
          <ExternalLink href="https://collegefootballdata.com">CollegeFootballData.com</ExternalLink>, with timeouts and retries, and saves a
          copy in a Firestore database.
        </li>
        <li>A second Go service reads that copy and answers this site’s requests, so visitors never wait on the outside data source.</li>
        <li>
          This page is a React app. Google Cloud Run runs the two Go services, and the second one also serves the pages of
          this site.
        </li>
        <li>Every change runs the tests in GitHub Actions, and only a change that passes them is deployed.</li>
      </ul>

      <h2>What it does not do</h2>
      <ul>
        <li>
          Scores are not live. The free data plan has no live scoreboard, and the data is refreshed every few hours, more
          often on Saturdays. A game that has kicked off is labeled “Kicked off”, with a dash in place of each score,
          until its result arrives.
        </li>
        <li>It covers only the Big 12 and the SEC.</li>
        <li>It has no accounts, favorites, or notifications, and nothing related to betting.</li>
      </ul>

      <p>
        The code is on <ExternalLink href={REPO_URL}>GitHub</ExternalLink>, along with the{' '}
        <ExternalLink href={`${REPO_URL}/blob/main/docs/SRS.md`}>requirements and design decisions</ExternalLink> written
        before building, the <ExternalLink href={`${REPO_URL}/blob/main/docs/ARCHITECTURE.md`}>architecture diagram</ExternalLink>,
        and a step-by-step build guide.
      </p>
    </article>
  );
}
