# Architecture

Visitors read the site's own copy of the data; only the ingest service talks to CollegeFootballData. The two paths never wait on each other.

```mermaid
flowchart LR
  browser["Web browser<br/>React app"]
  subgraph gcp["Google Cloud project: game-day-board"]
    web["gdb-web (Cloud Run, public)<br/>Go: API under /api/ plus the React files"]
    fs[("Firestore<br/>games, teams, calendar,<br/>ingest status")]
    sched["Cloud Scheduler<br/>every 6 hours, and every<br/>30 minutes on Saturdays"]
    ingest["gdb-ingest (Cloud Run, private)<br/>Go: timeouts and retries"]
    secret["Secret Manager<br/>CFBD key"]
    logs["Cloud Logging and Monitoring<br/>JSON logs, uptime check, email alert"]
    ar["Artifact Registry<br/>container images"]
  end
  cfbd["CollegeFootballData API"]
  gha["GitHub Actions<br/>test, build, deploy"]

  browser -->|"pages and /api calls"| web
  web -->|reads| fs
  sched -->|"signed call: POST /run"| ingest
  ingest -->|"reads key"| secret
  ingest -->|"fetches"| cfbd
  ingest -->|"writes"| fs
  gha -->|"pushes images"| ar
  ar -->|"deploys"| web
  ar -->|"deploys"| ingest
  web -.-> logs
  ingest -.-> logs
```

## The two paths

**Read path.** The browser loads the React app from `gdb-web`. React then calls `/api/...` on the same service, which reads Firestore. Every response carries `lastUpdated`, the time of the last successful ingest, so the page always shows how old its data is.

**Ingest path.** Cloud Scheduler signs a call to the private `gdb-ingest` service. Cloud Run refuses any call without a valid signature from the scheduler's service account, so the code does not check it. Ingest reads the CFBD key from Secret Manager (as an environment variable), fetches data, keeps only games involving Big 12 and SEC teams, and saves them under their CFBD game ID, so a repeated run overwrites rather than duplicates.

## How few calls it makes

The CFBD free tier allows 1,000 calls a month.

| Run type | When | Calls |
| --- | --- | --- |
| Full refresh: calendar, teams for 2 conferences, regular and postseason games | First run, then once a week | 5 |
| Current week only | Every other run | 1 |

Schedule: every 6 hours (about 120 runs a month) plus every 30 minutes from noon to midnight Central time on Saturdays (about 100 more). That is roughly 240 calls a month, under a quarter of the quota. Every run records CFBD's own `X-CallLimit-Remaining` count in Firestore and in the logs.

## Availability features

| Feature | Where | Plain words |
| --- | --- | --- |
| Liveness `/api/health` | `api` | "Is the process running?" Checks nothing else, so a database blip never gets healthy instances restarted |
| Readiness `/api/ready` | `api` | "Can it do useful work?" Checks that Firestore answers |
| Timeouts | `cfbd` (10 seconds per attempt), `api` (5 seconds per database call), HTTP server | Nothing waits forever |
| Retries with backoff and jitter | `cfbd`, `retry` | Up to 3 retries on timeouts, 429, and 5xx, waiting longer each time, with randomness so many clients do not retry in step. Honors `Retry-After`. Never retries a 4xx that our own request caused |
| Serve the last good data | `ingest`, `api` | A failed ingest leaves stored data alone and records the error; the site keeps working and shows the data's age |
| Graceful shutdown | `httpx` | On SIGTERM, finish requests in progress before exiting |
| Structured logs with trace IDs | `httpx` | One JSON line per request; every line from one request shares a trace ID in Cloud Logging |
| Verified deploys | CI/CD | A new version gets no traffic until its health check passes |
| Scaling caps and a budget alert | Cloud Run, Billing | At most 2 instances of the site and 1 of ingest; email at 50, 90, and 100 percent of $5 a month |

## Decision: one service for the pages and the API

**What we do.** One Cloud Run service, `gdb-web`, serves both the React files and the API. The Go code in `backend/internal/web` sends `/api/...` requests to the API and every other request to the built React files.

**What the [SRS](SRS.md) first planned (decision D6).** Firebase Hosting for the React files, with `/api` passed through to Cloud Run. This "static files from a CDN, API from a container" split is a very common architecture. We changed course because adding Firebase to a Google Cloud project needs a one-time step in the Firebase console, and then weighed whether the split was worth it at this size. It was not, yet.

**The tradeoff.**

| | One Cloud Run service (what we do) | Firebase Hosting plus Cloud Run (the split) |
| --- | --- | --- |
| Where page files come from | One region (us-central1, Iowa) | A CDN: copies on servers around the world, near each visitor |
| First visit after a quiet spell | About a second while Cloud Run starts an instance | Page frame appears at once, but the scores still wait for Cloud Run, so the total wait barely changes |
| If Cloud Run is down | Nothing loads | The page frame loads, with no games to show |
| Products to set up, secure, and explain | One | Two |
| Deploys per change | One container | Two (files and container), which the pipeline must keep in step |
| Preview addresses for a change | No; the pipeline's candidate revision covers safe releases | Yes, a temporary address per change |
| Custom domain | Possible through Cloud Run domain mapping, with a few more steps | Simple |
| Money | Free at this traffic | Free at this traffic |

**Why one service wins for now.** Visitors are few and mostly in the United States, so a CDN saves little time, and the data still comes from Cloud Run either way. One service is simpler to run, deploy, and explain.

**When we would switch.** Many visitors far from Iowa, a need for the page to load even when the API is down, or a custom domain. Switching changes only the deploy step (build the React files, deploy them to Firebase, add a `/api` rewrite) and the container stops copying the files in. The Go and React code stay the same.

## Services and identities

| Identity (service account) | Used by | Permissions |
| --- | --- | --- |
| `gdb-api` | `gdb-web` | Read Firestore |
| `gdb-ingest` | `gdb-ingest` | Read and write Firestore; read the one secret |
| `gdb-scheduler` | Cloud Scheduler | Call `gdb-ingest` |
| `gdb-deployer` | GitHub Actions, through Workload Identity Federation | Deploy Cloud Run, push images, run as the two service identities above |

Each identity has only what its job needs (NFR-10), so a mistake or a compromise in one part cannot reach the rest.

## Runbook: when something goes wrong

| Symptom | First checks | Fix |
| --- | --- | --- |
| Email: "Game Day Board site is down" | Console: **Cloud Run → gdb-web → Logs** for errors; the latest run under the repository's **Actions** tab | If the latest deploy broke it: **Cloud Run → gdb-web → Revisions**, pick the previous revision, **Manage traffic**, send it 100 percent |
| Site up, but "Data updated" is many hours old | **Cloud Scheduler**: last run result; **Cloud Run → gdb-ingest → Logs**: lines with "ingest run failed" | Fix the cause (for example a revoked key: add a new version of the `cfbd-api-key` secret), then **Cloud Scheduler → gdb-ingest-regular → Force run** |
| Ingest logs show HTTP 401 from CollegeFootballData | The key is wrong or was revoked | Request a new key; `gcloud secrets versions add cfbd-api-key --data-file=-` with the key piped in, as in build guide step 1.4 |
| `callsRemaining` in the ingest logs is low | Something is calling more often than planned | Pause the Saturday job: **Cloud Scheduler → gdb-ingest-saturday → Pause** |
