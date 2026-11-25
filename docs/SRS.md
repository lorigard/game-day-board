# Game Day Board: Software Requirements Specification

Lori Gardner · Written 2026-11-16, before any code

> **About this document.** This is the plan written before building: the requirements, and the reason for every design choice. It is kept as written, so it shows what was decided and why, at the time it was decided. The first version was built on 2026-11-23, and a few things turned out differently. Each difference is listed, with its reason, in [What was built differently](#what-was-built-differently) at the end. For how the system works today, see [ARCHITECTURE.md](ARCHITECTURE.md).

## 1. Introduction

Game Day Board is a small web app that shows this week's college football games and scores, and a page for each team. It is a learning and portfolio project, built with a Go back end, a React front end, and Google Cloud hosting.

### 1.1 Purpose

This document says what the app must do, how well it must do it, and why each design choice was made. It is the reference for every milestone, and a script for explaining the project in an interview.

### 1.2 Scope

- **In scope:** a scheduled job that copies game data from a public football API into our own database, a Go API that serves that data, a React site that displays it, automated tests, a CI/CD pipeline, health checks, logging, and simple monitoring.
- **Out of scope:** anything betting related, user accounts, live play-by-play, and NFL data (see section 11).
- **Why it exists:** to practice a modern cloud stack (Go services, React, Google Cloud) and to close specific skill gaps: AI-assisted development with skills and agents, automated testing, CI/CD, availability, clean code, and architecture diagrams.

### 1.3 Definitions

| Term | Plain meaning |
| --- | --- |
| CFBD | CollegeFootballData.com, the public data source we read from |
| Ingest | Pulling data from an outside source and saving our own copy |
| Cloud Run | A Google Cloud service that runs a container for us and scales it up and down, even to zero |
| Firestore | A Google Cloud document database (stores records as documents, a little like JSON files) |
| Container | A packaged app plus everything it needs to run, so it behaves the same everywhere |
| CI/CD | Continuous integration (test every change automatically) and continuous delivery or deployment (release tested changes automatically) |
| Idempotent | Safe to run twice: the second run changes nothing extra |
| Health check | A small endpoint that answers "am I working?" so a platform or monitor can tell |
| Skill | A packaged set of instructions an AI agent loads for one kind of task |
| Subagent | A separate AI agent that a main agent hands part of a task to |

### 1.4 References

- [CollegeFootballData.com terms of use](https://collegefootballdata.com/terms)
- [CollegeFootballData.com API getting started](https://api.collegefootballdata.com/getting-started)

## 2. Overall description

The app is a self-contained system with one outside dependency, the CFBD API. Visitors reach it from a link on lori-gardner.com.

### 2.1 Product perspective

It mirrors, at a very small size, what a sports data team does: take in game data from a provider, keep a reliable copy, and serve it quickly to many users. Users never wait on the outside provider, because the app reads from its own copy.

### 2.2 Users

| User | What they want |
| --- | --- |
| A visitor (for example, an interviewer) | See this week's games and scores, open a team page, and find the source code and diagram |
| The developer (Lori) | Change the code safely: tests run on every push, and a green pipeline deploys |
| The operator (also Lori) | Know when the app is down or the data is stale, and find out why from the logs |

### 2.3 Constraints

- **Cost:** stay inside Google Cloud free tier limits, with a budget alert as a safety net.
- **Data source quota:** the CFBD free tier allows a limited number of calls a month (reported as about 1,000; to confirm on the CFBD tiers page in Milestone 0). The design must stay well under it.
- **Data terms:** use data only as the CFBD terms allow. Attribution ("Data provided by CollegeFootballData.com") is shown on every page, as their terms recommend.
- **Secrets:** no key or password is ever committed to the repository.
- **Stack:** Go for the back end and React with TypeScript for the front end, because they are the stack this project sets out to learn.
- **Time:** part-time work over a few weeks, so the scope stays small.

### 2.4 Assumptions

- The CFBD API stays available and its free tier keeps roughly its current limits.
- The college season is in its final weeks (bowl games run into January), so live scores are available for part of the build. Past seasons are always available for testing.
- Lori will create a new Google Cloud project for this app, under her own account (to confirm).

## 3. Functional requirements

Each requirement has an ID so tests, commits, and pull requests can point to it. "Must" items are the finished project; "Should" items are added only if time allows.

| ID | Priority | Requirement | Milestone |
| --- | --- | --- | --- |
| FR-1 | Must | On a schedule, the ingest service fetches games for the current season and week from CFBD for a configured list of conferences, and saves them to Firestore. | 2 |
| FR-2 | Must | Running ingest twice for the same week leaves the same data (idempotent: each game is saved under its CFBD game ID, so a repeat overwrites rather than duplicates). | 2 |
| FR-3 | Must | Ingest fetches the list of teams for the configured conferences, so team names, conferences, and colors come from the data, not from code. | 2 |
| FR-4 | Must | The API returns the games for a given season and week, as JSON. | 2 |
| FR-5 | Must | The API returns one team and its season schedule with results, as JSON. | 2 |
| FR-6 | Must | The home page shows this week's games, grouped by conference: teams, kickoff time in the visitor's time zone, and the score once a game has started or finished. | 3 |
| FR-7 | Must | A visitor can step to the previous or next week. | 3 |
| FR-8 | Must | Each team name links to a team page with its schedule, results, and win-loss record. | 3 |
| FR-9 | Must | Every page shows when the data was last updated, so stale data is visible rather than hidden. | 3 |
| FR-10 | Must | Every page shows "Data provided by CollegeFootballData.com" and a link to an About page. | 3 |
| FR-11 | Must | The About page explains what the app does, links to the source code and the architecture diagram, and states plainly what is not built. | 6 |
| FR-12 | Must | When the API cannot be reached or returns an error, the page shows a clear, friendly message and a retry button, never a blank screen. | 3 |
| FR-13 | Should | A visitor can filter the home page to one conference. | 3 |
| FR-14 | Should | During game windows (Saturdays, and bowl dates), ingest runs more often than at other times, within the CFBD quota. | 5 |

**Decision: which conferences.** The default list is the Big 12 and the SEC (two conferences, roughly 32 teams). That keeps the data small and the calls few. The list is a setting, not code, so changing it needs no rebuild.

## 4. Non-functional requirements

These say how well the app must work. Each one has a check, so "done" is never a matter of opinion.

| ID | Area | Requirement | How we check it |
| --- | --- | --- | --- |
| NFR-1 | Availability | The API exposes `/healthz` (is the process alive?) and `/readyz` (can it reach Firestore?). | Integration test; Cloud Run uses them |
| NFR-2 | Availability | Every outgoing call has a timeout: 10 seconds to CFBD, 5 seconds to Firestore. | Unit test with a slow fake server |
| NFR-3 | Availability | Calls to CFBD are retried up to 3 times with growing waits (backoff plus a little randomness, called jitter), but only for errors worth retrying: timeouts, 429 (too many requests), and 5xx (server errors). A 4xx caused by our own request is not retried. | Unit tests, one per case |
| NFR-4 | Availability | If ingest fails, the API keeps serving the last good data, and the page shows its age (FR-9). | Integration test |
| NFR-5 | Availability | The service shuts down gracefully: on a stop signal it finishes requests in progress before exiting. | Unit test of the shutdown function |
| NFR-6 | Performance | API responses for games and teams return in under 300 milliseconds at the 95th percentile, once the service is warm. A first request after idle (a "cold start") may take up to about 2 seconds. | Cloud Monitoring latency chart |
| NFR-7 | Quota | Ingest uses under half of the CFBD monthly quota. Planned schedule: every 6 hours normally, every 30 minutes during Saturday game windows, about 250 calls a month. | Count in logs; alert at 60 percent |
| NFR-8 | Security | No secret in the repository. The CFBD key lives in Secret Manager; GitHub Actions signs in to Google Cloud with Workload Identity Federation (short-lived credentials, no key file). | Secret scanning in CI; code review |
| NFR-9 | Security | The ingest endpoint is private: only Cloud Scheduler's identity may call it. The read API is public and read-only. | Integration test: an unsigned call is refused |
| NFR-10 | Security | Each service runs as its own service account with the least permissions it needs. | Reviewed in Milestone 4 |
| NFR-11 | Cost | Monthly cost stays at or near $0. A budget alert emails at 50, 90, and 100 percent of $5. Cloud Run is capped at 2 instances per service. | Billing report each milestone |
| NFR-12 | Observability | Logs are structured JSON with a request ID, so one request can be followed through the logs. An uptime check pings `/healthz` and emails on failure. | Seen in Cloud Logging; alert tested once on purpose |
| NFR-13 | Code quality | Clear names, small functions, comments that explain why rather than what, Go doc comments on every exported name, and TSDoc on exported React components and hooks. | Review skill on every change; `go vet`, `golangci-lint`, ESLint in CI |
| NFR-14 | Testability | Code that talks to the outside world (CFBD, Firestore, the clock) sits behind a small interface, so tests can swap in a fake. | Unit tests run without a network |
| NFR-15 | Accessibility | Pages work by keyboard, use real headings and labels, and meet WCAG AA color contrast. | Testing Library queries by role; a manual check |
| NFR-16 | Honesty | The README, the About page, and any description on lori-gardner.com say only what is built and running. | Read through at the end of each milestone |

The numbers in NFR-2, NFR-3, NFR-6, and NFR-7 are starting choices, not measurements. We will revisit them once the app is running and we can see real timings.

## 5. Architecture overview

Two paths run through the system. The read path serves visitors from Firestore; the ingest path refreshes Firestore from CFBD on a timer. The two never wait on each other.

The diagram of the system as built is in [ARCHITECTURE.md](ARCHITECTURE.md).

**Read path:** the browser loads the React files from Firebase Hosting, which forwards `/api/*` calls to the API service, which reads Firestore.

**Ingest path:** Cloud Scheduler makes a signed call to the private ingest service, which reads the CFBD key from Secret Manager, fetches from CFBD with timeouts and retries, and writes to Firestore.

**Around both:** GitHub Actions deploys only tested builds, and Cloud Logging and Monitoring watches both services. Section 6 explains why each piece was chosen.

## 6. Design decisions

Each decision lists what we chose, what else we considered, why, and what we gave up. Interviewers often care more about the "why" and the trade-off than the choice itself.

### D1. Data source: CollegeFootballData.com

- **Considered:** nflverse (NFL data), TheSportsDB (has NFL), CFBD (college).
- **Why:** CFBD's published terms clearly allow this use, the key is free, and the API is well documented. nflverse's code is open source, but it says the NFL data itself belongs to its owners, and I could not confirm TheSportsDB's free-tier terms.
- **Trade-off:** college football, not the NFL. Still American football, and still honest data.

### D2. Keep our own copy, rather than calling CFBD on every page view

- **Considered:** the API acting as a pass-through to CFBD on each request.
- **Why:** a pass-through would burn the monthly quota in a few hundred page views, would be as slow as CFBD, and would go down whenever CFBD does. With our own copy, users are served quickly from Firestore, and a CFBD outage only makes data older, not missing (NFR-4).
- **Trade-off:** scores can be up to 30 minutes old during games. That is fine for a demonstration, and the page says how old the data is.

### D3. Two small services: ingest and API

- **Considered:** one service doing both; a Cloud Run job (a run-to-completion task) for ingest.
- **Why:** splitting them is a small, honest example of microservices. Each scales on its own, and each gets only the permissions it needs: only ingest can read the CFBD key and write to Firestore; the API can only read. Both live in one Go module (`cmd/api` and `cmd/ingest`), sharing code in `internal/`.
- **Why not a job:** a job is a good fit for batch work and worth knowing. A private HTTP service is simpler to test locally and teaches service-to-service authentication, which comes up in interviews. We can switch later without changing the core code.
- **Trade-off:** two deploys instead of one, which the pipeline handles.

### D4. Hosting: Cloud Run, not Kubernetes

- **Considered:** Google Kubernetes Engine (GKE), App Engine, a Compute Engine virtual machine.
- **Why:** Cloud Run runs our container, scales to zero when idle, and has a generous free tier. A GKE cluster costs money every month even when idle, and a virtual machine needs patching and stays running. Cloud Run still teaches the ideas that carry over to Kubernetes: containers, health checks, configuration through environment variables, and scaling.
- **Trade-off:** no hands-on Kubernetes in production. An optional exercise runs the same container on a free local cluster (kind), so that experience is real if we do it.

### D5. Database: Firestore

- **Considered:** Cloud SQL for PostgreSQL (closest to my SQL Server background), storing JSON files in Cloud Storage.
- **Why:** Firestore has a free tier and needs no server; Cloud SQL has no free tier and runs (and bills) all the time. Our data is simple: games and teams, read by key or by season and week. A document database fits that well. There is a local emulator for integration tests.
- **Trade-off:** no SQL joins. We shape documents around the pages that read them (section 7). Learning a NoSQL model is also useful in its own right.

### D6. Front-end hosting: Firebase Hosting, with `/api` routed to Cloud Run

- **Considered:** serving the React files from the Go container; hosting them on Netlify next to my site.
- **Why:** Firebase Hosting is part of Google Cloud, free at this size, and serves files from a content delivery network (copies close to each visitor). A rewrite rule sends `/api/*` to the Cloud Run API, so the browser sees one address. That avoids cross-origin (CORS) setup and keeps the front end and back end deployable on their own.
- **Trade-off:** one more Google product to learn. lori-gardner.com simply links to the app's address.

**Changed 2026-11-23:** adding Firebase to a Google Cloud project needs a one-time step in the Firebase console, so the first version serves the React files from the same Cloud Run service as the API (the alternative listed above), at `https://gdb-web-442687404328.us-central1.run.app`. Moving to Firebase Hosting later changes only the deploy step. In the same change, the health checks moved from `/healthz` and `/readyz` to `/api/health` and `/api/ready`, because Cloud Run reserves some paths that end in "z". The full trade-off is in [ARCHITECTURE.md](ARCHITECTURE.md), under "Decision: one service for the pages and the API".

### D7. Front end: React with TypeScript, built with Vite

- **Considered:** Next.js; plain JavaScript.
- **Why:** Vite gives a plain single-page app with very little hidden machinery, so every part can be explained. TypeScript catches mistakes early, much like C# does. Data loading starts with a small custom hook built on `fetch`, so I learn how it works before reaching for a library such as TanStack Query.
- **Trade-off:** no server-side rendering, which this app does not need.

### D8. Back end: Go standard library

- **Considered:** web frameworks such as Gin or Chi.
- **Why:** since Go 1.22, the standard `net/http` router handles methods and path parameters, and `log/slog` gives structured logging. Fewer dependencies means less to learn and less to update. Most Go teams read standard library code easily.
- **Trade-off:** a little more code for things a framework would give for free, such as middleware chaining. That code is short and worth understanding.

### D9. One repository for everything

- **Considered:** separate repositories for back end and front end.
- **Why:** one place for the code, the diagram, the pipeline, and the docs. One link from my site shows the whole project. The pipeline only rebuilds the part that changed.
- **Trade-off:** none that matter at this size.

### D10. CI/CD: GitHub Actions with Workload Identity Federation

- **Considered:** Google Cloud Build; a service account key stored as a GitHub secret.
- **Why:** GitHub Actions is widely used and I build the pipeline myself, step by step. Workload Identity Federation lets GitHub prove who it is to Google Cloud and receive short-lived credentials, so there is no long-lived key to leak.
- **Trade-off:** a little more setup than pasting a key, and that setup is exactly the part worth learning.

### D11. Cloud setup: a documented script first, Terraform later if time allows

- **Considered:** Terraform (infrastructure as code) from the start; clicking through the console.
- **Why:** a short, commented `gcloud` script records every resource and can be rerun, without learning a fourth new tool at once. The console is used to look, not to change things.
- **Trade-off:** less polished than Terraform. Converting to Terraform is a "Should" item, and is described only if it is done.

### D12. Testing tools

- **Go:** the standard `testing` package with table-driven tests, `httptest` for fake servers, and the Firestore emulator for integration tests.
- **React:** Vitest, React Testing Library (tests what the user sees, by role and label), and Mock Service Worker (fakes the API at the network level, so components are tested as they really run).
- **End to end:** one Playwright test of the deployed site, as a "Should" item.
- **Why these:** each is the common default in its community, so the skills carry over to other teams.

## 7. Data model and interfaces

The database holds three kinds of documents, shaped around the pages that read them. The API has four public read endpoints and one private ingest endpoint.

### 7.1 Firestore collections

| Collection / document ID | Main fields | Why it is shaped this way |
| --- | --- | --- |
| `games/{cfbdGameId}` | season, week, seasonType, startTime (UTC), completed, home and away team ID, name, and points, conferences, updatedAt | Using CFBD's own game ID as the document ID is what makes ingest idempotent (FR-2). Team names are copied in, so the home page needs one query, not one per team. |
| `teams/{cfbdTeamId}` | school, mascot, abbreviation, conference, color, alternateColor, updatedAt | Changes rarely, so ingest refreshes it once a week. |
| `meta/ingest` | lastSuccessAt, lastAttemptAt, lastError, callsThisMonth | Feeds "last updated" (FR-9), the stale-data alert, and the quota count (NFR-7). |

Copying team names into each game is called denormalization. In SQL Server we would join instead; in a document database, copying small, rarely changing fields is the normal trade for faster reads.

Game status (scheduled, in progress, or final) is worked out from `startTime` and `completed`. The free CFBD endpoints do not appear to offer true live status, so "in progress" means "started and not yet marked complete". We will confirm this in Milestone 1.

### 7.2 Outside interface: CFBD API

- Base address `https://api.collegefootballdata.com`, with the key sent as a bearer token in the `Authorization` header.
- Endpoints we expect to use: games by season and week, and teams by conference. Exact paths and fields are confirmed against the official docs in Milestone 1, before any code depends on them.
- Every call goes through one small Go client (`internal/cfbd`), so timeouts, retries, logging, and the quota count live in one place.

### 7.3 Our API

| Method and path | Returns | Who may call |
| --- | --- | --- |
| `GET /api/games?season=2026&week=12&conference=SEC` | Games for that week (conference is optional) | Anyone |
| `GET /api/teams/{id}` | One team | Anyone |
| `GET /api/teams/{id}/schedule?season=2026` | The team's games, results, and record | Anyone |
| `GET /api/weeks/current` | The current season and week, so the home page knows where to start | Anyone |
| `GET /healthz`, `GET /readyz` | Liveness and readiness (NFR-1) | Platform and uptime check |
| `POST /ingest/run` (ingest service) | Runs one ingest pass and reports counts | Cloud Scheduler's identity only (NFR-9) |

Every successful response wraps its data with the time it was last updated: `{"data": ..., "lastUpdated": "2026-11-21T19:30:00Z"}`. Every error uses one shape, `{"error": {"code": "not_found", "message": "No team with ID 9999."}}`, with the matching HTTP status. A consistent error shape means the front end needs only one piece of error-handling code (FR-12).

## 8. Testing strategy

Most tests are small and fast, fewer test pieces working together, and very few test the whole deployed system. That shape is the test pyramid: the lower a test sits, the faster and cheaper it is, so most checking happens there.

| Layer | What it tests | Tools | Where it runs | Examples in this project |
| --- | --- | --- | --- | --- |
| Unit (many) | One function or component, with every outside dependency faked | Go `testing`, `httptest`; Vitest, React Testing Library, Mock Service Worker | On my machine on every save; CI on every push | Retry only on 429 and 5xx (NFR-3); timeout fires (NFR-2); kickoff time shows in local time (FR-6); error message and retry button (FR-12) |
| Integration (some) | Our code with a real dependency running locally | Firestore emulator; Go tests that start the real HTTP server | CI on every push | Ingest twice leaves one copy of each game (FR-2); `/readyz` fails when Firestore is down; the API serves old data when ingest fails (NFR-4); an unsigned call to ingest is refused (NFR-9) |
| End to end (very few) | The deployed site as a visitor uses it | Playwright | CI after each deploy | Home page loads, shows games, a team link opens the team page |

### 8.1 Testing across the development life cycle

- **Requirements:** every requirement in sections 3 and 4 has an ID and a stated check. If we cannot say how to test it, it is not finished being written.
- **Design:** interfaces around CFBD, Firestore, and the clock (NFR-14) are chosen so the code can be tested.
- **Coding:** tests are written with or before the code. For tricky logic such as retries and status rules, the test comes first (test-driven development).
- **Review:** the review skill (section 10) checks that each change has tests and that they test behavior, not internals.
- **CI:** every push runs linting, unit tests, and integration tests. Nothing merges or deploys unless they pass.
- **Release:** after a deploy, the end-to-end test and the uptime check confirm the live site works.

### 8.2 Rules for good tests

- One reason to fail per test, and a name that says the expected behavior, for example `TestClient_DoesNotRetryOn404`.
- No real network in unit tests, and no sleeping on a real clock: time is passed in, so tests run in milliseconds.
- React tests find elements the way a person does (by role, label, or text), not by CSS classes.

## 9. CI/CD and operations

A pull request runs the checks only; a merge to `main` runs the same checks and then deploys. Only a green build reaches production.

### 9.1 Pipeline (GitHub Actions)

1. **Check:** lint (`go vet`, `golangci-lint`, ESLint), type-check (TypeScript), and scan for committed secrets.
2. **Test:** Go unit tests, React component tests, then integration tests against the Firestore emulator. Back end and front end run in parallel.
3. **Stop here for pull requests.** The result shows on the pull request, and `main` is protected so a failing change cannot merge.
4. **Build (on `main` only):** build the two Go containers, tag each with the commit ID, and push them to Artifact Registry. Build the React files.
5. **Deploy:** sign in to Google Cloud through Workload Identity Federation, deploy both Cloud Run services, and deploy the front end to Firebase Hosting.
6. **Verify:** call `/healthz` on the new version, then run the end-to-end test (when built). If the health check fails, roll back to the previous version.

Tagging images with the commit ID means every running version traces back to an exact commit, and rolling back is just redeploying an older tag.

### 9.2 Operations

| Concern | What we set up | What it means in plain words |
| --- | --- | --- |
| Is it up? | Uptime check on `/healthz` every 5 minutes, with an email alert | Google pings the app from outside, the way a visitor would |
| Is the data fresh? | Alert if `meta/ingest.lastSuccessAt` is more than 12 hours old | The app can be up while quietly serving stale data; this catches that |
| What went wrong? | Structured JSON logs with a request ID, viewed in Cloud Logging | Every log line for one request can be pulled up together |
| Is it slow? | Cloud Run's built-in latency and error charts | Shows whether NFR-6 holds |
| Is it costing money? | Budget alert at $5, max 2 instances per service | A warning, not a hard stop, plus a cap on how far it can scale |
| How do I fix it? | A short runbook in the repository: check the logs, roll back, rerun ingest | Steps written down before they are needed |

## 10. Development process

AI drafts, and I own every line: nothing is committed until I have read it, understood it, and seen its tests pass. The workflow is written down in `AI-WORKFLOW.md` in the repository, so it can be explained and repeated.

### 10.1 The workflow, for every change

1. **Plan:** state the requirement IDs the change covers, and agree on the approach before any code is written.
2. **Draft:** Claude Code drafts the code and its tests, following the project's `CLAUDE.md` rules (style, comments, testing).
3. **Review by agent:** the review skill (below) checks the change for clean code, comments, and missing tests.
4. **Review by me:** I read every line, ask about anything I cannot explain, and change what I would write differently.
5. **Test:** run the tests locally, and add any case the review found missing.
6. **Commit:** a small commit whose message names the requirement IDs, then push and let CI confirm.

### 10.2 Skills and agents we build

| What | Kind | What it does | Milestone |
| --- | --- | --- | --- |
| `CLAUDE.md` | Project instructions | Tells any agent the stack, folder layout, coding rules, and test rules for this repository | 0 |
| `clean-code-review` | Skill, written by me | Reviews a change for names, function size, why-not-what comments, error handling, and missing tests; reports findings by file and line | 0 |
| `test-writer` | Subagent | Given a function and its requirement, proposes table-driven tests, including edge cases | 1 |
| Parallel build | Orchestration | One agent plans a feature; two subagents build the API endpoint and the React component at the same time, in separate git worktrees; I review and merge both | 3 |

The orchestration run in Milestone 3 is an experiment. Whatever happens, including what went wrong, gets written down in `AI-WORKFLOW.md`, because an honest account of it is more useful in an interview than a polished one.

### 10.3 Milestones

1. **Milestone 0, setup:** tools installed, repository, Google Cloud project, budget alert, CFBD key, "hello" Go endpoint and React page with one test each, a first pipeline that runs those tests, `CLAUDE.md`, `AI-WORKFLOW.md`, and the review skill.
2. **Milestone 1, CFBD client:** timeouts, retries, error types, unit tests, and the `test-writer` subagent.
3. **Milestone 2, storage and ingest:** Firestore, the ingest service, idempotent saves, the read API, health endpoints, and integration tests.
4. **Milestone 3, React front end:** home and team pages, loading and error states, component tests, and the orchestration experiment.
5. **Milestone 4, deploy pipeline:** containers, Artifact Registry, Cloud Run, Firebase Hosting, Cloud Scheduler, Secret Manager, and Workload Identity Federation.
6. **Milestone 5, availability and monitoring:** structured logs, uptime and freshness alerts, graceful shutdown, scaling caps, and the runbook.
7. **Milestone 6, polish:** the final diagram, an honest README and About page, the link from lori-gardner.com, and interview talking points.

## 11. Out of scope, open questions, and the honesty rule

### 11.1 Out of scope

- Betting lines, odds, or anything betting related, even though CFBD offers some.
- NFL data, until a source with clear terms is found.
- User accounts, favorites, or notifications.
- Live play-by-play or real-time push updates.
- Kubernetes in production (see D4), and Terraform unless it is actually done (see D11).

### 11.2 Open questions

- [x] Repository: `game-day-board` on GitHub. Public from the start, so the no-secrets rule and branch protection apply from the first commit.
- [x] Google Cloud: use the existing Google account, and create a Cloud Billing account in Milestone 0.
- [x] Default conferences: the Big 12 and the SEC.
- [ ] Confirm the CFBD free-tier call quota on its tiers page (Milestone 0).
- [ ] Confirm Google Cloud free-tier limits for Cloud Run, Firestore, Cloud Scheduler, Secret Manager, Artifact Registry, and Firebase Hosting on the official pricing pages (Milestone 0).
- [ ] Confirm the CFBD endpoint paths and whether free endpoints offer any live game status (Milestone 1).

### 11.3 The honesty rule

The README, the About page, lori-gardner.com, my CV, and anything I say in an interview describe this project only as far as it is built and running. Work in progress is called work in progress. "Should" items and optional exercises are mentioned only once they are done.

### 11.4 Future features

Ideas to build after the first version. None is promised, and none appears on the site or in the README until it is built.

| ID | Feature | Notes |
| --- | --- | --- |
| F1 | TV network on each game card for the current week (for example "ESPN" or "FOX"), added 2026-11-25 | CollegeFootballData has a games media endpoint that lists the outlet and type (TV, web, radio) per game. Check first that the free tier includes it. One extra call per ingest run would roughly double the monthly use to about 480 of the 1,000 free calls; fetching it once a day instead keeps the increase near 30 calls. Show it in the card's navy strip or on the "at Kansas · Memorial Stadium" line, only for games not yet played. Networks change late in the week, so show the data's age as the rest of the site does. |

## What was built differently

Added 2026-11-25. The first version was built on 2026-11-23, in one session with Claude Code, rather than milestone by milestone. This table lists every place where it differs from the plan above, so nothing above is read as more than what is running.

| Plan (section) | What was built | Why |
| --- | --- | --- |
| Firebase Hosting serves the React files (D6, 5, 9.1) | The Go service on Cloud Run serves both the React files and the API | Adding Firebase needs a one-time console step. One service is simpler at this size; the trade-off is written up in ARCHITECTURE.md |
| Health checks at `/healthz` and `/readyz` (NFR-1, NFR-12, 7.3, 9) | `/api/health` and `/api/ready` | Cloud Run reserves some paths that end in "z" |
| Ingest endpoint `POST /ingest/run` (7.3) | `POST /run` on the private ingest service, plus `GET /health` | The service has one job, so a shorter path was enough |
| `meta/ingest` holds `callsThisMonth` (7.1) | It holds `lastAttemptAt`, `lastSuccessAt`, `lastFullRefreshAt`, `lastError`, `gamesSaved`, and `callsRemaining`; a `meta/calendar` document holds the season's weeks | CFBD reports the calls left in every response, which is more accurate than counting our own calls |
| About 250 calls a month (NFR-7) | About 240: a full refresh (5 calls) once a week, otherwise 1 call per run | Measured plan in ARCHITECTURE.md |
| Live status uncertain (7.1, 11.2) | Confirmed: the free tier has no live scoreboard. A started game is labeled "Kicked off", with a dash for each score, until the final arrives | Honest about what the data can say |
| CFBD quota to confirm (2.3, 11.2) | Confirmed: 1,000 calls a month on the free tier | |
| `golangci-lint` in CI (NFR-13, 9.1) | `go vet`, ESLint, the TypeScript check, and a gitleaks secret scan run in CI; `golangci-lint` is not added yet | Not done yet |
| `main` is protected (9.1) | Not turned on yet | Planned |
| Playwright end-to-end test (D12, 8, 9.1) | Not built. After each deploy, the pipeline checks the new version's health endpoint before it gets any traffic, then checks the live site again | A "Should" item |
| Stale-data alert and a quota alert (9.2, NFR-7) | Not built. The uptime check and its email alert are running | Planned |
| `clean-code-review` skill written by me, `test-writer` subagent, parallel-build experiment (10.2, 10.3) | Not built yet. `CLAUDE.md` and `AI-WORKFLOW.md` exist | Planned for the step-by-step rebuild |
| A documented `gcloud` script (D11) | Every `gcloud` command is recorded, with what it does, in the build guide (Part 1), rather than in one script | The guide doubles as a tutorial |
| Cloud Run capped at 2 instances per service (NFR-11) | The site is capped at 2 instances, ingest at 1 | Ingest runs one at a time anyway |
| Milestones 0 to 6, one at a time (10.3) | Built in one session, then reviewed and revised | Time: a working link was needed sooner. I am now rebuilding it step by step from the build guide |
