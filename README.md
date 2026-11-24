# Game Day Board

This week's college football games and scores for the Big 12 and the SEC, with a page for each team showing its schedule, results, and record.

**Live site:** https://gdb-web-442687404328.us-central1.run.app

**Status:** the first version is running. It is a learning project for a stack that is new to me (Go, React, and Google Cloud), and I am now working through every part of it step by step. See [How AI is used on this project](docs/AI-WORKFLOW.md).

## What it does

- Shows one week's games at a time, grouped by conference, with kickoff times in your own time zone and final scores. Previous and next week buttons, and a conference filter.
- A page per team with its season schedule, results, and win-loss record.
- Every page shows how old its data is.

## What it does not do

- **Scores are not live.** The free data plan has no live scoreboard. Data refreshes every 6 hours, and every 30 minutes on Saturday afternoons and evenings. A game that has kicked off shows "In progress or awaiting final score" until its result arrives.
- Only the Big 12 and the SEC. No accounts, and nothing related to betting.

## How it is built

| Part | Technology |
| --- | --- |
| Back end | Go 1.27, standard library only for HTTP (`net/http`) and logging (`log/slog`); two services in one module |
| Front end | React 19 and TypeScript, built with Vite |
| Data | Firestore, refreshed from the [CollegeFootballData.com](https://collegefootballdata.com) API by a scheduled ingest service |
| Hosting | Google Cloud Run (the site and API in one service; ingest as a private service), Cloud Scheduler, Secret Manager |
| CI/CD | GitHub Actions: Go and React checks, unit, component, and integration tests, and a secret scan on every push; deploys to Cloud Run with Workload Identity Federation, so no cloud key is stored anywhere |

- [Architecture](docs/ARCHITECTURE.md): the diagram, the two data paths, and the availability features (health checks, timeouts, retries with backoff, graceful shutdown, structured logs).
- [Build guide](docs/BUILD-GUIDE.md): every setup step, with commands and reasons, to rebuild the project from an empty machine.

## Tests

| Layer | Where | Run with |
| --- | --- | --- |
| Go unit tests | `backend/internal/...` | `go test ./...` in `backend/` |
| Go integration tests (Firestore emulator) | `backend/internal/store` | In CI; see the workflow |
| React component and unit tests | `frontend/src/...` | `npm test` in `frontend/` |

## Data

Data provided by [CollegeFootballData.com](https://collegefootballdata.com). This site's API exists only to serve this site's pages; it is not a public data service.
