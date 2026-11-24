# Game Day Board: instructions for AI coding agents

Game Day Board shows this week's college football games and scores for the Big 12 and the SEC, with a page per team. Data comes from the CollegeFootballData.com (CFBD) API. It is a public portfolio and learning project.

## Rules

1. **No secrets in the repository, ever.** The CFBD key lives in a gitignored `.env` locally and in Google Secret Manager in the cloud. The repository is public and its history is permanent.
2. **No personal details in project files.** No personal email addresses, and no names of prospective employers or anything about the owner's job search, in code, docs, or commit messages.
3. **Honesty.** README, About page, and docs describe only what is built and running. Work in progress is called work in progress.
4. **Writing style** for anything a person reads (docs, comments, UI text): US English spelling, no contractions, no em dashes, warm and clear.
5. **Keep `docs/BUILD-GUIDE.md` current.** Record every setup step as it is done: exact commands, what each does, how to check it worked, and why it matters.
6. **CFBD terms:** the read API is this site's own back end, never a public data API (no CORS, not advertised). Keys never go in browser code. Do not poll CFBD more than the schedule in `docs/ARCHITECTURE.md`.

## Layout

| Path | What it is |
| --- | --- |
| `backend/` | One Go module, two programs: `cmd/api` (the site: API plus built React files) and `cmd/ingest` (private; copies CFBD data into Firestore) |
| `backend/internal/` | `football` (domain types), `cfbd` (outside API client with timeouts and retries), `retry`, `store` (Firestore and in-memory), `ingest`, `api`, `web` (single-page app file server), `httpx` (logging, middleware, graceful shutdown) |
| `frontend/` | React + TypeScript, built with Vite. Pages in `src/pages`, components in `src/components`, API client in `src/api` |
| `Dockerfile` | Multi-stage build with two targets: `web` and `ingest` |
| `.github/workflows/ci-cd.yml` | Checks and tests on every push; deploy to Cloud Run on `main` |
| `infra/` | Cloud resource settings that are files (for example the image cleanup policy) |
| `docs/` | `BUILD-GUIDE.md`, `ARCHITECTURE.md`, `AI-WORKFLOW.md` |

## Commands

```powershell
# Back end (from backend/)
go test ./...                     # unit tests
go vet ./...
gofmt -l .                        # lists unformatted files; must print nothing

# Front end (from frontend/)
npm test                          # component and unit tests
npm run lint
npm run typecheck
npm run dev                       # local site; /api calls go to the deployed back end
```

Integration tests (`-tags=integration`) need the Firestore emulator, which needs Java; CI runs them.

## Code style

- Comments explain **why**, not what. Every exported Go name has a doc comment; exported React components and hooks have a short TSDoc comment.
- Clear names, small functions, one job each.
- Anything that talks to the outside world (CFBD, Firestore, the clock) sits behind an interface or a parameter, so tests can use a fake.
- Every requirement-related change has a test. Test names say the expected behavior. No real network or real sleeping in unit tests.
- Requirement IDs (FR-n, NFR-n) and design decision IDs (Dn) refer to the Software Requirements Specification.
