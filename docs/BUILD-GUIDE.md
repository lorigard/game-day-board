# Game Day Board: Build Guide

This guide records every step taken to build Game Day Board, in order, with the exact commands and the reason for each one. It is written so the whole project can be rebuilt from an empty machine by following it from top to bottom.

**How to use this guide**

- Work through the parts in order. Each step says what to do, what each command or option does, how to check it worked, and why it matters.
- Commands are for **PowerShell on Windows** unless a step says otherwise. Lines starting with `#` are comments, not commands.
- Text in angle brackets, such as `<your-github-username>`, is a placeholder. Replace it, brackets included.
- Never paste a real key, password, or token into this file. Steps that involve secrets say where the secret is kept instead.
- The requirements and design decisions are in the Software Requirements Specification ([SRS.md](SRS.md)). Requirement IDs such as FR-2 or NFR-8, and decision IDs such as D4, refer to it.

**Reviewing and committing in VS Code**

After step 0.4, every commit can be made from the VS Code **Source Control** panel (the branch icon on the left, or **Ctrl+Shift+G**) instead of the terminal. It runs the same git commands underneath.

| Terminal command | What it does | In the Source Control panel |
| --- | --- | --- |
| `git diff` | Shows what changed in each file | Click a file under **Changes** for a side-by-side view: old on the left, new on the right |
| `git add <file>` | Stages a file, meaning it goes into the next commit | Hover over the file and click **+** (Stage Changes). It moves to **Staged Changes** |
| `git status` | Lists staged and unstaged files | The panel itself: **Staged Changes** go into the commit, **Changes** do not |
| `git commit -m "..."` | Saves the staged files as one snapshot with a message | Type the message in the box at the top, then click **Commit** |
| `git push` | Uploads new commits to GitHub | Click **Sync Changes**, which appears after a commit |

Stage files one at a time with **+** rather than letting VS Code stage everything. Files that git ignores, such as `.env`, never appear in the panel.

In the terminal, `git diff` opens long output in a pager, a viewer that shows one screen at a time and waits at a `:` prompt. Press **q** to leave it. Anything typed or pasted while the pager is open is read as key presses, not run as commands.

**Status key:** ✅ done · ⏳ in progress · ⬜ not started

---

## Contents

- [Part 0: Setup (Milestone 0)](#part-0-setup-milestone-0)
  - [0.1 Install the tools](#01-install-the-tools)
  - [0.2 Create the local repository](#02-create-the-local-repository)
  - [0.3 Keep your email address private in commits](#03-keep-your-email-address-private-in-commits)
  - [0.4 Create the GitHub repository and push](#04-create-the-github-repository-and-push)
  - [0.5 Google Cloud: account security and free-tier check](#05-google-cloud-account-security-and-free-tier-check)
  - [0.6 Google Cloud: billing account, project, and budget alert](#06-google-cloud-billing-account-project-and-budget-alert)
  - [0.7 CollegeFootballData.com API key](#07-collegefootballdatacom-api-key)
  - [0.8 Remaining setup items](#08-remaining-setup-items)
- [Part 1: The first version (built 2026-11-23)](#part-1-the-first-version-built-2026-11-23)
- [Troubleshooting](#troubleshooting)
- [Change log for this guide](#change-log-for-this-guide)

---

## Part 0: Setup (Milestone 0)

Goal: a machine with every tool installed, a public GitHub repository with a working first pipeline, a Google Cloud project with a budget alert, and the project's AI instructions and first skill. No app features yet.

### 0.1 Install the tools

**Status:** ✅ done (2026-11-16)

**Which shell:** the commands in this step also work in Git Bash, because `winget`, `go`, `gcloud`, and `gh` are ordinary Windows programs. Later steps use PowerShell-only commands such as `Set-Location`, so this guide uses PowerShell throughout.

**What each tool is for**

| Tool | What it is | Why this project needs it |
| --- | --- | --- |
| Git | Version control: records every change to the code | Already installed. Tracks history and pushes to GitHub |
| Node.js and npm | JavaScript runtime and package manager | Already installed (Node 24). Builds and tests the React front end |
| Go | The Go language compiler and tools | Builds and tests the back-end services |
| Google Cloud CLI (`gcloud`) | Command-line access to Google Cloud | Creates and manages the cloud project, services, and secrets |
| GitHub CLI (`gh`) | Command-line access to GitHub | Creates the repository and checks pipeline runs without leaving the terminal |
| Docker | Builds and runs containers | Not needed until Milestone 4 (deploy). Install it then |

**Steps**

1. Check what is already installed. Any tool that prints a version can be skipped.

   ```powershell
   git --version
   node --version
   go version
   gcloud version
   gh --version
   ```

   **What it does:** each command asks one tool to print its version. A version number means the tool is installed and Windows can find it; "not recognized" means it is missing.

2. Install the missing tools with `winget`, the package manager built into Windows. Each command may open a permission prompt; accept it.

   ```powershell
   winget install --id GoLang.Go -e
   winget install --id Google.CloudSDK -e
   winget install --id GitHub.cli -e
   ```

   **What it does:** `winget install` downloads the official installer for each tool and runs it. `--id` picks the exact package, and `-e` means "exact match only", so winget never guesses a similarly named package.

3. **Close every terminal and VS Code completely, then reopen them.** Installers add the tools to the system PATH (the list of folders Windows searches for commands), and open windows do not see that change.

4. Check that each tool now answers:

   ```powershell
   go version
   gcloud version
   gh --version
   ```

   **Expected:** each prints a version number. If one says "not recognized", see [Troubleshooting](#troubleshooting).

**Versions used when this guide was written (2026-11-16):**

| Tool | Version |
| --- | --- |
| Go | go1.27.4 windows/amd64 |
| Google Cloud SDK | 593.0.0 (core 2026.11.06, bq 2.1.42, gsutil 5.38) |
| GitHub CLI | 2.104.0 (2026-11-12) |
| Node.js | 24 (installed before this project) |

Newer versions should work. If a later step behaves differently from what this guide says, compare your version with this table first.

### 0.2 Create the local repository

**Status:** ✅ done (2026-11-16)

**Steps**

1. Create the project folder and start a git repository whose main branch is called `main`:

   ```powershell
   New-Item -ItemType Directory C:\Source\VSCode\game-day-board
   Set-Location C:\Source\VSCode\game-day-board
   git init -b main
   ```

   **What it does:**

   - `New-Item -ItemType Directory` creates the folder.
   - `Set-Location` moves the terminal into it (like `cd`).
   - `git init` turns the folder into a git repository by adding a hidden `.git` folder, where git keeps the history. `-b main` names the first branch `main`.

2. Add these files (their full contents are in the repository):

   - `.gitignore`: tells git which files never to track. The most important entries are the secret files (`.env`, key files, service account files). It goes in **before** any other file, so a secret can never be committed by accident.
   - `.gitattributes`: stores text files with Unix line endings (LF). Windows uses CRLF, but the pipeline runs on Linux; without this, the same file can look different on each and cause noisy diffs.
   - `README.md`: says only what is built. Right now that is nothing, so it says "work in progress".
   - `docs/BUILD-GUIDE.md`: this guide.

3. Check that git sees the files and ignores nothing unexpected:

   ```powershell
   git status
   ```

   **What it does:** `git status` lists files that are new, changed, or ready to commit. "Untracked" means git sees the file but is not recording it yet. Files matched by `.gitignore` do not appear at all.

   **Expected:** the four files above are listed as untracked.

**Why it matters:** the repository will be public from the first commit, and everything in its history is public forever, even if it is deleted later. If a secret is ever committed, deleting the file is not enough: the secret must be canceled (revoked) and replaced.

### 0.3 Keep your email address private in commits

**Status:** ✅ done (2026-11-16). First commit made with the noreply address.

Every git commit records the author's email address, and anyone can read it in a public repository. GitHub provides a private "noreply" address to use instead.

**Steps**

1. On GitHub, open **Settings**, then **Emails**.
2. Turn on **Keep my email addresses private**. This makes GitHub use a private stand-in address for anything it does on your behalf, such as edits made on the website. GitHub then shows your noreply address. It looks like `<number>+<your-github-username>@users.noreply.github.com`.
3. Also turn on **Block command line pushes that expose my email**. This makes GitHub refuse a push that contains your real address, as a safety net.
4. In the project folder, set the noreply address for **this repository only** (without `--global`, so other projects are not affected):

   ```powershell
   Set-Location C:\Source\VSCode\game-day-board
   git config user.email "<your-noreply-address>"
   git config user.email   # prints the address back, to check it
   ```

   **What it does:** `git config user.email` sets the email address that git writes into every new commit made in this folder. It is stored in `.git/config`, which is never pushed.

5. Make the first commit:

   ```powershell
   git add .
   git commit -m "Initial project setup: README, build guide, gitignore, gitattributes"
   git log --format="%an <%ae>" -1   # shows the author name and email of that commit
   ```

   **What it does:**

   - `git add .` stages every new or changed file in the folder, meaning it marks them to go into the next commit.
   - `git commit -m "..."` records the staged files as one saved snapshot in the history, with the message after `-m`.
   - `git log` shows the history. `--format="%an <%ae>"` prints only the author name and email, and `-1` shows only the latest commit.

   **Expected:** the last command shows the noreply address, not your personal email.

### 0.4 Create the GitHub repository and push

**Status:** ✅ done (2026-11-16). Created with Option A, then pushed.

There are two ways to do this. Option A uses the GitHub website and was the one used for this project. Option B does the same thing from the terminal with the GitHub CLI.

#### Option A: on the GitHub website (used for this project)

1. Sign in at github.com. Click the **+** in the top right corner, then **New repository**.
2. Fill in the form:

   | Field | Value | Why |
   | --- | --- | --- |
   | Owner | your account | |
   | Repository name | `game-day-board` | Matches the local folder and the app name |
   | Description | `College football scores and team pages for the Big 12 and the SEC. A learning project in Go, React, and Google Cloud. Work in progress.` | Honest about what it is today |
   | Visibility | **Public** | The site will link to it, and interviewers can read it |
   | Add README | **Off** | The local folder already has one |
   | Add .gitignore | **No .gitignore** | The local folder already has one |
   | Add license | **No license** | Chosen later, on purpose (see the note below) |

   Leaving all three "Add" options off creates an **empty** repository. If GitHub created its own first commit, the local and online histories would not match, and the first push would be refused.

3. Click **Create repository**. GitHub shows a "Quick setup" page. Copy the HTTPS address, which looks like `https://github.com/<your-github-username>/game-day-board.git`.

4. After the first commit (step 0.3), connect the local folder to GitHub and push:

   ```powershell
   Set-Location C:\Source\VSCode\game-day-board
   git remote add origin https://github.com/<your-github-username>/game-day-board.git
   git remote -v                # shows the address for fetch and push, to check it
   git push -u origin main
   ```

   - `git remote add origin` saves the GitHub address under the name `origin`, the usual name. It only stores the address on this machine; nothing is sent yet.
   - `git push` uploads the local commits that GitHub does not have yet.
   - `-u` links the local `main` branch to `origin/main`, so later pushes need only `git push`.
   - The first push opens a browser window from Git Credential Manager (included with Git for Windows) to sign in to GitHub. It then remembers the sign-in, so no password or token is stored in the project.

5. Check that the push worked, from the terminal:

   ```powershell
   git status
   git log --oneline origin/main   # lists the commits GitHub now has
   ```

   **Expected:** `git status` says `Your branch is up to date with 'origin/main'`, which means the local `main` and the GitHub `main` point to the same commit. The log lists the same commits as `git log --oneline`.

6. Refresh the repository page. **Expected:** the README shows, and the commit author is your GitHub account, not a personal email address.

**Tip:** before the first push, check that no commit carries a personal address. This lists the author and committer email of every commit:

```powershell
git log --format="%h %ae %ce"
```

`%h` is the short commit ID, `%ae` the author's email, and `%ce` the committer's email (usually the same person). Every line should show only the noreply address.

**About the license:** with no license, people can read public code but have no permission to reuse it. That is fine while building. A common choice for a portfolio project is the MIT License, which lets others reuse the code with credit. It can be added at any time from the repository page (**Add file → Create new file**, name it `LICENSE`, and pick a template).

#### Option B: from the terminal with the GitHub CLI

1. Sign in to GitHub from the terminal. Choose **GitHub.com**, **HTTPS**, and **Login with a web browser** when asked.

   ```powershell
   gh auth login
   gh auth status   # confirms which account you are signed in as
   ```

   **What it does:** `gh auth login` opens a browser to sign in, then stores a sign-in token in your Windows user profile so later `gh` commands can act as you.

2. Create the public repository from the local folder, and push the first commit to it, in one command:

   ```powershell
   Set-Location C:\Source\VSCode\game-day-board
   gh repo create game-day-board --public --source . --remote origin --push
   ```

   - `gh repo create game-day-board` creates a repository with that name on GitHub, and `--public` makes it readable by anyone.
   - `--source .` uses the current folder as the repository.
   - `--remote origin` names the link to GitHub `origin`, the usual name.
   - `--push` uploads the existing commit.

   Creating the repository this way, rather than on the GitHub website, means GitHub does not add its own README or license, so the local and online histories match.

3. Check that it worked:

   ```powershell
   gh repo view --web   # opens the repository in the browser
   ```

   **Expected:** the README shows on the repository page.

### 0.5 Google Cloud: account security and free-tier check

**Status:** ✅ done (2026-11-17)

**How Google Cloud is organized**

| Layer | What it is | Rough Azure equivalent |
| --- | --- | --- |
| Google account | Your sign-in. It is who you are | Your Microsoft account |
| Cloud Billing account | The payment method. Budgets and alerts attach here, and free-tier limits are counted here | A billing account or subscription |
| Project | Holds everything the app uses: services, database, secrets, permissions. Each project links to one billing account | A resource group, but each project also turns on its own APIs and sets its own permissions |

Every Google Cloud API is off in a new project. Each one (Cloud Run, Firestore, and so on) is turned on before first use.

**Steps**

1. **Turn on 2-Step Verification** for the Google account that will own the billing account: go to myaccount.google.com, then **Security**, then **2-Step Verification**. Save the backup codes somewhere safe, outside this repository. (It was already on for this project.)

   **What it does:** after the password, Google also asks for a second proof that it is you, such as a prompt on your phone. Backup codes are one-time codes that get you in if the phone is lost.

   **Why it matters:** this account will control a payment method and every cloud resource. A password alone is not enough.

2. **Read the current free terms** on Google's own page before signing up, because they change: https://docs.cloud.google.com/free/docs/free-cloud-features. Also check https://cloud.google.com/scheduler/pricing and https://firebase.google.com/pricing, which are not covered on that page.

**What the pages said on 2026-11-17**

Two different offers are easy to mix up. The **Free Trial** is temporary credit for new customers. The **Free Tier** is a monthly allowance of free usage that continues after the trial. This project's cost goal (NFR-11) depends on the Free Tier.

Free Trial:

- $300 credit for 90 days. No charges during the trial. A card is needed at signup, for identity checks only (a temporary hold of $0 to $1).
- Only for people who have never been a paying user of Google Cloud, Google Maps Platform, or Firebase, and have never had the trial before.
- **If the account is not upgraded to Paid by day 90, the billing account closes and every resource stops.** After a 30-day grace period, the resources are deleted. Upgrading keeps the remaining credit (until day 90) and the free tier.
- During the trial, the credit brings the net cost to $0, so a budget alert only works if it is set to **ignore credits** (step 0.6).

Free Tier (monthly, no end date; Google may change it with 30 days' notice; unused amounts do not roll over):

| Service | Free limit | This project's expected use | Notes |
| --- | --- | --- | --- |
| Cloud Run | 2 million requests; 180,000 vCPU-seconds; 360,000 GB-seconds; 1 GB outbound from North America | A few thousand requests | Far under |
| Firestore | 1 GiB storage; 50,000 reads, 20,000 writes, 20,000 deletes **per day** per project | About 1,500 writes on a Saturday | Reads are the one to watch: one home page view reads about 30 games |
| Cloud Scheduler | 3 jobs per **billing account** (across all projects; a paused job still counts). $0.10 per extra job per month | 2 or 3 jobs | Billed per job, not per run, so running often costs nothing extra |
| Secret Manager | 6 active secret versions; 10,000 access operations | 1 secret | Far under |
| Artifact Registry | **0.5 GB storage** | Two images per push to main | **The main cost risk:** old images pile up. A cleanup policy is added in Milestone 4 |
| Logging | 50 GiB of logs per project | Tiny | Far under |
| Firebase Hosting | 10 GB storage; 360 MB data transfer **per day** | A few hundred KB per first visit | Fine. A project with a billing account counts as Firebase's Blaze (pay as you go) plan, which still includes these no-cost amounts |

### 0.6 Google Cloud: billing account, project, and budget alert

**Status:** ⏳ in progress

**Decision: start with the Free Trial, and upgrade to Paid before it ends.** The trial cannot charge anything while the project is being learned and built. The app must stay live after day 90, so the account is upgraded before then (step 6). Upgrading keeps the remaining credit and the free tier.

**Steps**

1. **Sign up for the Free Trial** at https://console.cloud.google.com/freetrial, signed in with the Google account from step 0.5. In November 2026 the form asked only for the country (United States), acceptance of the terms, and a card. The card gets a temporary hold of $0 to $1, not a charge. Do **not** click **Activate** or **Upgrade** yet.

   **What it does:** signing up creates a billing account, the thing that pays for usage, loaded with the $300 credit. The card is only checked, not charged.

   Google then creates a Free Trial billing account named "My Billing Account" and a project named "My First Project". This project does not use "My First Project"; it gets its own project in step 3.

   **Check:** in the console, open the menu, then **Billing**. **Expected:** the overview says "Free trial account" and shows $300 of credit with an end date.

   **Done 2026-11-17. Trial end date: February 16, 2027.**

2. **Sign the Google Cloud CLI in to your account** (opens a browser window to choose the account and allow access):

   ```powershell
   gcloud auth login
   gcloud auth list   # the active account is marked with an asterisk
   ```

   **What it does:** `gcloud auth login` lets the command-line tool act as your Google account. The sign-in token is kept in your Windows user profile, outside the repository. `gcloud auth list` shows which accounts are signed in. This works like `az login` for Azure.

3. **Create the project.** The project ID is permanent, must be unique across all of Google Cloud, and appears in public addresses (for example, the default site address `<project-id>.web.app`). The display name is only a label and can be changed later.

   ```powershell
   gcloud projects create game-day-board --name="Game Day Board"
   gcloud projects list   # check: the new project is listed
   ```

   **What it does:** `gcloud projects create` makes an empty project with that ID. `--name` sets the display name shown in the console. `gcloud projects list` lists every project your account can see.

   If the ID is already in use, pick another one, such as `<your-initials>-game-day-board`, and use it everywhere this guide says `game-day-board`. This project got `game-day-board` (2026-11-17).

   **Why from the command line:** design decision D11 records every cloud resource as a command that can be rerun, rather than clicks in the console.

4. **Link the project to the billing account** (in the console). Choose **Game Day Board** in the project picker at the top, then go to **☰ menu → Billing**. If the new project is not in the picker yet (see Troubleshooting), open this address instead, which selects it for you:

   https://console.cloud.google.com/billing/linkedaccount?project=game-day-board

   It says "This project has no billing account". Click **Link a billing account**, choose **My Billing Account**, then click **Set account**.

   **What it does:** linking tells Google which billing account pays for this project's usage. Without a link, most services in the project refuse to start, even free ones.

   Then make `game-day-board` the default project for the Google Cloud CLI, so later commands do not need `--project` each time:

   ```powershell
   gcloud config set project game-day-board
   gcloud billing projects describe game-day-board --format="value(billingEnabled)"
   ```

   **What it does:** `gcloud config set project` saves a default project in your local `gcloud` settings. `gcloud billing projects describe` shows the project's billing details, and `--format="value(billingEnabled)"` prints only the one field that says whether billing is on.

   **Expected:** the second command prints `True`.

   **Does linking billing risk a charge during the trial?** No. A Free Trial billing account is not billable: if usage ever reached the $300 credit, the account would close rather than charge the card. Charges are possible only after upgrading to Paid (step 6).

   **Tip:** check the project name at the top of the console before changing anything. It is easy to change "My First Project" by mistake.

5. **Create a $5 budget alert** (in the console). Go to **☰ menu → Billing → Budgets & caps**, then click **Create new**. The wizard has four parts:

   | Part | Setting | Why |
   | --- | --- | --- |
   | Define | **Alerts only**. Name: `Game Day Board monthly` | "Spend cap enforcement" pauses services, and the screen did not say which services it covers. Revisit at the upgrade (step 6) |
   | Scope | Time range **Monthly**; Projects **All projects**; Services **All services** | Free-tier limits are counted per billing account, so the whole account is watched |
   | Scope, Savings | Open **Other savings** with its arrow, and uncheck **Promotional credits** only. Leave everything else checked, including **Free tier credits** | The trial credit is a promotional credit. If it is counted, the cost always shows $0 and the alert never fires. Free-tier credits stay on, so free usage does not count toward the $5 |
   | Amount | Budget type **Specified amount**; Target amount **5** | |
   | Actions | Thresholds **50%, 90%, 100%**, each triggered on **Actual**; **Email alerts to billing admins and users** checked | These were the defaults. As billing admin, you receive the emails |

   Click **Finish**.

   **Check:** **Budgets & caps** lists "Game Day Board monthly", Monthly, "7 credits" under **Applies to**, and alerts at 50%, 90%, and 100%.

   **Why it matters:** a budget is a warning, not a limit; Google emails but stops nothing. During the trial nothing can be charged anyway. After the upgrade, these emails are the early warning (NFR-11).

   **Note for the upgrade:** in November 2026, enforced spend caps were changed to reset automatically at the start of each month, resuming paused services. That makes a cap more attractive once the account is Paid.

6. **Upgrade to a Paid billing account before the trial ends.** Reminder set for February 1, 2027. If this is missed, the billing account closes and every resource stops (step 0.5). In the console, go to the Welcome page and click **Activate**. *To come.*

### 0.7 CollegeFootballData.com API key

**Status:** ✅ done (2026-11-17)

**What an API key is:** a long random string that identifies you to the CFBD service. The ingest service sends it with every request, so CFBD knows whose monthly quota to count. Anyone who has the key can use up that quota, so it is treated as a secret.

**Steps**

1. On collegefootballdata.com, click **Request a free API key**. In the form, **Primary use** is a short survey question; this project chose **App builder**. Click **Submit**. CFBD emails the key.

   **Do not paste the key into this guide, a chat, or any project file.** It is stored locally in step 2.

2. **Store the key locally in a `.env` file.**

   **What a `.env` file is:** a plain text file of `NAME=value` lines that programs load as environment variables. It does the same job as User Secrets (`dotnet user-secrets`) in .NET: the secret stays on your machine, outside the code. The Go services read the key with `os.Getenv("CFBD_API_KEY")`, like `Environment.GetEnvironmentVariable` in C#. In Google Cloud, Secret Manager supplies the same variable, so the code does not change.

   The repository includes `.env.example`, which lists every setting by name with no values. It is committed so anyone rebuilding the project can see which settings they need. `.gitignore` blocks `.env` but allows `.env.example`.

   In the VS Code **Explorer**, right-click empty space at the top level of the project, choose **New File**, and name it `.env`. Put in one line, with your key after the `=`: no spaces around `=` and no quotation marks.

   ```
   CFBD_API_KEY=<your-key>
   ```

   Save it. **Expected:** VS Code shows `.env` in gray, which means git ignores it.

3. **Check that git ignores the file**, without printing the key:

   ```powershell
   git check-ignore -v .env
   git status --short
   ```

   **What it does:** `git check-ignore -v` prints which `.gitignore` line matches a file; it prints nothing if the file is not ignored. `git status --short` lists new and changed files, one per line.

   **Expected:** the first command prints `.gitignore:3:.env	.env`. The second does not list `.env`.

   **If the key is ever committed or shared by mistake:** deleting it is not enough, because git history is public. Request a new key from CFBD and stop using the old one.

**What the tiers and terms pages said on 2026-11-17** (https://collegefootballdata.com/api-tiers and https://collegefootballdata.com/terms)

| Fact | Effect on this project |
| --- | --- |
| Free tier: **1,000 calls a month**, no card needed | The plan uses about 250 (NFR-7), under half |
| **Live scoreboard is not in the free tier.** It starts at the $1 a month tier (Tier 1, 5,000 calls a month) | The app works out "in progress" as "started and not yet marked complete" (SRS 7.1). Decided 2026-11-23: stay on the free tier. A started game is labeled "Kicked off", with a dash for each score, until its final arrives |
| One key and one shared monthly limit cover both college football and college basketball | Only football is used, so no effect |
| Scheduled polling and server-side caching are encouraged; excessive polling is prohibited | Matches design decision D2: ingest on a timer, serve from Firestore |
| Keys must not be put in a public repository or in browser code | The key lives only on the server side (ingest service), never in the React app |
| Going over the quota may suspend the key | Why NFR-7 asks for a call count and an alert at 60 percent. The count is logged on every run; the alert is not built yet (step 1.9) |
| No service-level agreement; features can change without notice | Another reason to serve from our own copy (D2, NFR-4) |
| Attribution is appreciated but not required; suggested text "Data provided by CollegeFootballData.com" | Shown on every page anyway (FR-10) |
| Running a public mirror, proxy, or substitute API of the data is prohibited | The read API exists only to serve this site's pages. It is described as the site's back end, never advertised as a public data API |

### 0.8 Remaining setup items

- ⬜ Branch protection on `main` (now that the pipeline exists): see step 1.7.
- ⬜ VS Code extensions: Go (`golang.go`) and ESLint (`dbaeumer.vscode-eslint`).
- ⬜ My own clean-code review skill (see `docs/AI-WORKFLOW.md`).

The "hello" endpoint, "hello" page, and first pipeline from the original Milestone 0 list were overtaken by Part 1, which built the whole first version.

---

## Part 1: The first version (built 2026-11-23)

**Status:** ✅ running at https://gdb-web-442687404328.us-central1.run.app

**How this part was done.** To get a working first version online quickly, Claude Code drafted the code, tests, pipeline, and cloud resources in one session, from the SRS, and ran the cloud commands with my signed-in `gcloud`. This part records exactly what was run, in order, so it can be repeated and understood. The plan for the following weeks is to work through each step here myself, one at a time (step 1.10).

Commands in this part are written for **Git Bash** (the `\` at the end of a line continues the command on the next line). In PowerShell, use a backtick `` ` `` instead of `\` at the end of each line.

### 1.1 What was built, and where it lives

| Path | What it is |
| --- | --- |
| `backend/internal/football` | Our own data types (Game, Team, Week) and small rules: game status, win-loss record, current week, which season a date belongs to |
| `backend/internal/cfbd` | The CollegeFootballData client: 10-second timeout per attempt, up to 3 retries with backoff and jitter on timeouts, 429, and 5xx, reads the `X-CallLimit-Remaining` quota header |
| `backend/internal/retry` | The backoff calculation, on its own so it can be tested exactly |
| `backend/internal/store` | The `Store` interface, the Firestore version, and an in-memory version for tests |
| `backend/internal/ingest` | Decides what to fetch (full refresh weekly, otherwise the current week only), keeps Big 12 and SEC games, saves them, records the run's status |
| `backend/internal/api` | The read endpoints under `/api/`, plus `/api/health` and `/api/ready` |
| `backend/internal/web` | Serves the built React files, sending `index.html` for page addresses such as `/teams/2305` |
| `backend/internal/httpx` | JSON logs in Cloud Logging's format, request logging with trace IDs, panic recovery, graceful shutdown |
| `backend/cmd/api`, `backend/cmd/ingest` | The two programs' `main` functions: read settings from environment variables and start the server |
| `frontend/src` | React app: `api/` (fetch and types), `hooks/useApi.ts` (loading, error, success), `components/`, `pages/`, `lib/format.ts` |
| `Dockerfile` | Builds both container images (targets `web` and `ingest`) |
| `.github/workflows/ci-cd.yml` | The pipeline |

**Check it locally** (from the repository root, in PowerShell):

```powershell
cd backend
go test ./...          # every Go unit test; "ok" for each package
cd ..\frontend
npm ci                 # installs exactly the versions in package-lock.json
npm test               # 21 tests
npm run dev            # opens the site locally; its /api calls go to the live back end
```

### 1.2 Turn on the Google Cloud services the app uses

```bash
gcloud services enable run.googleapis.com firestore.googleapis.com \
  cloudscheduler.googleapis.com secretmanager.googleapis.com \
  artifactregistry.googleapis.com iam.googleapis.com iamcredentials.googleapis.com \
  sts.googleapis.com monitoring.googleapis.com logging.googleapis.com \
  cloudresourcemanager.googleapis.com serviceusage.googleapis.com \
  --project=game-day-board
```

**What it does:** every Google Cloud product is an API that starts switched off in a new project. This switches on the ones the app uses: Cloud Run, Firestore, Cloud Scheduler, Secret Manager, Artifact Registry (container images), IAM and the two token services Workload Identity Federation needs (`iamcredentials`, `sts`), and monitoring and logging. Enabling an API costs nothing; only usage is billed. (`firebase` and `firebasehosting` were also enabled while trying Firebase Hosting; see the SRS note on D6.)

### 1.3 Create the database and the image registry

```bash
gcloud firestore databases create --location=us-central1 --type=firestore-native --project=game-day-board

gcloud artifacts repositories create gdb --repository-format=docker \
  --location=us-central1 --description="Game Day Board container images" --project=game-day-board

gcloud artifacts repositories set-cleanup-policies gdb --location=us-central1 \
  --policy=infra/artifact-cleanup-policy.json --no-dry-run --project=game-day-board
```

- The first command creates the project's default Firestore database in Native mode (the document database the app uses), in `us-central1`, the same region as Cloud Run. The free tier applies to this default database.
- The second creates a private Docker registry named `gdb`, where the pipeline stores each built image.
- The third attaches the cleanup policy in `infra/artifact-cleanup-policy.json`: keep the 5 newest images, delete the rest after 7 days. **Why:** the free tier holds only 0.5 GB of images, and every push to `main` adds two (step 0.5).

### 1.4 Store the CFBD key in Secret Manager, without printing it

```bash
grep '^CFBD_API_KEY=' .env | cut -d= -f2- | tr -d '\r\n' \
  | gcloud secrets create cfbd-api-key --replication-policy=automatic --data-file=- --project=game-day-board
```

**What it does:** reads the key's line from `.env`, keeps only the part after `=`, strips the line ending, and pipes it straight into a new secret named `cfbd-api-key`. `--data-file=-` means "read the value from the pipe", so the key never appears on screen, in the shell history, or in a file. In Cloud Run, the ingest service receives it as the `CFBD_API_KEY` environment variable, the same name `.env` uses locally.

### 1.5 Create one identity per job, with only the permissions that job needs

A **service account** is an identity for a program rather than a person. Each part of the app gets its own, so a mistake or a break-in in one part cannot reach the rest (NFR-10).

```bash
for sa in gdb-api gdb-ingest gdb-scheduler gdb-deployer; do
  gcloud iam service-accounts create $sa --display-name="Game Day Board $sa" --project=game-day-board
done
```

| Account | Permission granted | Command |
| --- | --- | --- |
| `gdb-api` | Read Firestore | `gcloud projects add-iam-policy-binding game-day-board --member=serviceAccount:gdb-api@game-day-board.iam.gserviceaccount.com --role=roles/datastore.viewer --condition=None` |
| `gdb-ingest` | Read and write Firestore | same command with `gdb-ingest` and `--role=roles/datastore.user` |
| `gdb-ingest` | Read the one secret (not every secret) | `gcloud secrets add-iam-policy-binding cfbd-api-key --member=serviceAccount:gdb-ingest@game-day-board.iam.gserviceaccount.com --role=roles/secretmanager.secretAccessor` |
| `gdb-deployer` | Deploy and manage Cloud Run services | project binding with `--role=roles/run.admin` |
| `gdb-deployer` | Use the project's APIs on its own behalf | project binding with `--role=roles/serviceusage.serviceUsageConsumer` |
| `gdb-deployer` | Push images to the `gdb` registry only | `gcloud artifacts repositories add-iam-policy-binding gdb --location=us-central1 --member=serviceAccount:gdb-deployer@game-day-board.iam.gserviceaccount.com --role=roles/artifactregistry.writer` |
| `gdb-deployer` | Start services that run as `gdb-api` and `gdb-ingest` | `gcloud iam service-accounts add-iam-policy-binding gdb-api@game-day-board.iam.gserviceaccount.com --member=serviceAccount:gdb-deployer@game-day-board.iam.gserviceaccount.com --role=roles/iam.serviceAccountUser` (and the same for `gdb-ingest`) |
| `gdb-scheduler` | Call the private ingest service | step 1.8, after the service exists |

`--condition=None` means the permission always applies, with no extra rule such as a time limit.

### 1.6 Let GitHub Actions deploy without a stored key (Workload Identity Federation)

**The idea:** when a GitHub Actions job runs, GitHub can issue it a short-lived signed token that says "I am a run of the repository lorigard/game-day-board". Google Cloud is told to trust GitHub's signature, but only for this repository, and to exchange the token for credentials of `gdb-deployer` that expire within an hour. Nothing secret is stored in GitHub. This is design decision D10.

```bash
# 1. A "pool" groups outside identities that Google Cloud will trust.
gcloud iam workload-identity-pools create github --location=global \
  --display-name="GitHub Actions" --project=game-day-board

# 2. A "provider" says who signs the tokens (GitHub) and which tokens to accept.
gcloud iam workload-identity-pools providers create-oidc github-actions \
  --location=global --workload-identity-pool=github \
  --display-name="GitHub Actions OIDC" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref" \
  --attribute-condition="assertion.repository == 'lorigard/game-day-board'" \
  --project=game-day-board

# 3. Allow runs of this repository, and nothing else, to act as gdb-deployer.
gcloud iam service-accounts add-iam-policy-binding gdb-deployer@game-day-board.iam.gserviceaccount.com \
  --role=roles/iam.workloadIdentityUser \
  --member="principalSet://iam.googleapis.com/projects/442687404328/locations/global/workloadIdentityPools/github/attribute.repository/lorigard/game-day-board" \
  --project=game-day-board
```

- `--attribute-condition` is the safety catch: a token from any other repository is refused, even though every repository's tokens are signed by GitHub.
- `442687404328` is the project **number** (not the ID). Find it with `gcloud projects describe game-day-board --format="value(projectNumber)"`.
- The workflow uses the provider's full name: `projects/442687404328/locations/global/workloadIdentityPools/github/providers/github-actions`.

### 1.7 The pipeline

`.github/workflows/ci-cd.yml` has four jobs. The first three run on every push and pull request, side by side:

| Job | What it checks |
| --- | --- |
| Go checks and tests | Formatting (`gofmt`), `go vet`, unit tests with the race detector, then integration tests against the Firestore emulator |
| React checks and tests | ESLint, TypeScript type check, Vitest tests, production build |
| Scan for committed secrets | Gitleaks reads the whole git history for anything shaped like a key or password |

The fourth, **Build, deploy, and verify**, runs only on a push to `main`, and only if the other three passed:

1. Signs in to Google Cloud through Workload Identity Federation (step 1.6).
2. Builds both images and pushes them to Artifact Registry, tagged with the commit ID.
3. Deploys `gdb-ingest` as a private service with the secret attached.
4. Deploys the new `gdb-web` version with **no traffic** and a private "candidate" address, checks `/api/health` there, and only then moves visitors to it. If the check fails, visitors stay on the previous version.
5. Smoke-tests the live address.

**To do: branch protection.** On GitHub: **Settings → Branches → Add branch ruleset** (or **Add rule**) for `main`, require the three check jobs to pass before merging, and block force pushes. After that, changes go through pull requests.

### 1.8 Run ingest on a schedule (Cloud Scheduler)

The pipeline deploys the services; the schedule is set up once, by hand, after the first deploy, because it needs the ingest service's address.

```bash
# The ingest service's address (Cloud Run gives each service one).
INGEST_URL=$(gcloud run services describe gdb-ingest --region us-central1 --project game-day-board --format='value(status.url)')

# Allow the scheduler's identity, and only it, to call the private ingest service.
gcloud run services add-iam-policy-binding gdb-ingest --region us-central1 --project game-day-board \
  --member=serviceAccount:gdb-scheduler@game-day-board.iam.gserviceaccount.com --role=roles/run.invoker

# Every 6 hours, all week.
gcloud scheduler jobs create http gdb-ingest-regular --location us-central1 --project game-day-board \
  --schedule="0 */6 * * *" --time-zone="Etc/UTC" --uri="$INGEST_URL/run" --http-method=POST \
  --oidc-service-account-email=gdb-scheduler@game-day-board.iam.gserviceaccount.com \
  --oidc-token-audience="$INGEST_URL" \
  --attempt-deadline=120s --max-retry-attempts=2 --min-backoff=60s

# Every 30 minutes from noon to midnight Central time on Saturdays, when most games are played.
gcloud scheduler jobs create http gdb-ingest-saturday --location us-central1 --project game-day-board \
  --schedule="*/30 12-23 * * 6" --time-zone="America/Chicago" --uri="$INGEST_URL/run" --http-method=POST \
  --oidc-service-account-email=gdb-scheduler@game-day-board.iam.gserviceaccount.com \
  --oidc-token-audience="$INGEST_URL" \
  --attempt-deadline=120s --max-retry-attempts=2 --min-backoff=60s

# Run once now instead of waiting for the schedule.
gcloud scheduler jobs run gdb-ingest-regular --location us-central1 --project game-day-board
```

- `--schedule` uses cron format: minute, hour, day of month, month, day of week. `0 */6 * * *` is "minute 0 of every 6th hour"; `*/30 12-23 * * 6` is "every 30 minutes, hours 12 to 23, on Saturday (day 6)".
- `--oidc-service-account-email` makes the scheduler sign each call as `gdb-scheduler`. Cloud Run checks that signature and refuses anything else; `--oidc-token-audience` names which service the signature is for.
- `--max-retry-attempts=2 --min-backoff=60s`: if a run fails, the scheduler tries again after a minute, then again later.
- Two jobs fit inside the free tier's 3 jobs per billing account (step 0.5).

**Check:** in the console, **☰ menu → Cloud Scheduler** lists both jobs, and **Last run** shows "Success" after the manual run. In **☰ menu → Cloud Run → gdb-ingest → Logs**, the run's line reads "ingest run succeeded" with `gamesSaved` and `callsRemaining`. The first run on 2026-11-23 did a full refresh: 236 games saved, 992 CFBD calls left that month.

**Check that the service is private:** `curl -X POST -d '' <ingest URL>/run` from your own machine answers **403 Forbidden**, because the call carries no signature (NFR-9).

### 1.9 Get an email if the site goes down

```bash
# Ping /api/health from several places around the world every 5 minutes.
gcloud monitoring uptime create gdb-web-health --project=game-day-board \
  --resource-type=uptime-url \
  --resource-labels=host=gdb-web-442687404328.us-central1.run.app,project_id=game-day-board \
  --protocol=https --path=/api/health --period=5 --timeout=10

# Where alerts go: an email "notification channel".
gcloud beta monitoring channels create --display-name="Game Day Board owner email" \
  --type=email --channel-labels=email_address=<your-email> --project=game-day-board

# The alert rule itself, from infra/uptime-alert-policy.json.
gcloud beta monitoring policies create --policy-from-file=infra/uptime-alert-policy.json \
  --notification-channels=<channel name printed by the previous command> --project=game-day-board
```

- An **uptime check** is a robot visitor. Google calls the health endpoint from outside, the way a real visitor would, so it catches problems a check from inside the project would miss.
- The **alert policy** in `infra/uptime-alert-policy.json` fires when the check fails from more than one location, so a single unlucky ping does not wake anyone. The check's ID (`gdb-web-health-K6ZhAm6RrfI`) is printed by the first command and must match the one in the policy file.
- The email address lives only in Google Cloud, never in this repository.

**Check:** **☰ menu → Monitoring → Uptime checks** lists `gdb-web-health` with green ticks; **Monitoring → Alerting** lists "Game Day Board site is down".

**Not built yet:** an alert for stale data (ingest failing quietly while the site stays up), and an alert on the CFBD call count (NFR-7). Both are Milestone 5 items.

### 1.10 Plan: rebuilding it myself

Working through Part 1 in this order, one sitting each, with the [SRS](SRS.md) beside it:

1. Go basics through `internal/football`: types, methods, table-driven tests.
2. `internal/retry` and `internal/cfbd`: timeouts, retries, fake servers with `httptest`.
3. `internal/store` and `internal/ingest`: interfaces, Firestore, idempotent saves, the emulator.
4. `internal/api`, `internal/web`, `internal/httpx`: routing, JSON, health checks, logging, shutdown.
5. The React app: components, the `useApi` hook, Mock Service Worker tests.
6. The Dockerfile and the pipeline.
7. The cloud resources in steps 1.2 to 1.9, in the console first, then the commands.

---

## Troubleshooting

| Problem | Likely cause | Fix |
| --- | --- | --- |
| `go`, `gcloud`, or `gh` is "not recognized" right after installing | The terminal was open before the install, so it has the old PATH | Close all terminals and VS Code, then reopen. If it still fails, sign out of Windows and back in |
| A project made with `gcloud projects create` does not show in the console's project picker | The picker reads from a search index that can take a while to update | Open the project directly with `?project=<project-id>` at the end of a console address, as in step 0.6. It appears in the picker later |
| `gcloud` fails with status `429` and `Quota exceeded ... for consumer 'project_number:32555940559'` | No default project is set, so `gcloud` uses a project Google shares among many users, and its per-minute allowance ran out | Wait a minute and retry. Setting your own default project (`gcloud config set project <project-id>`, step 0.6) usually prevents it |
| `winget` asks to accept source agreements | First use of winget on this machine | Type `Y` and press Enter |
| `git push` says `fatal: 'origin' does not appear to be a git repository` | The remote was never added | Run `git remote add origin <address>` (step 0.4), then check with `git remote -v` |
| A push is rejected with a message about a private email | Step 0.3 was skipped, or the commit was made before the noreply address was set | Set the address (step 0.3), then fix the last commit with `git commit --amend --reset-author --no-edit` and push again |

---

## Change log for this guide

| Date | Change |
| --- | --- |
| 2026-11-16 | Guide started. Part 0 steps 0.1 to 0.4 written |
| 2026-11-16 | Steps 0.2 and 0.3 done; first commit made with the noreply address |
| 2026-11-16 | Step 0.4: added Option A (GitHub website), the path used for this project |
| 2026-11-16 | Step 0.4 done: remote added and first push made; added the push checks and the email check tip |
| 2026-11-16 | Step 0.1 done: Go, Google Cloud CLI, and GitHub CLI installed; versions recorded |
| 2026-11-17 | Step 0.5 started: Google Cloud layers explained, 2-Step Verification checked, free trial and free-tier limits recorded from Google's pages |
| 2026-11-17 | Step 0.5: Firebase Hosting limits added. Step 0.6 started: trial-then-upgrade decision recorded; Free Trial signed up (ends 2027-02-16) |
| 2026-11-17 | Step 0.6: gcloud signed in; project `game-day-board` created |
| 2026-11-17 | Step 0.6: billing linked in the console; default gcloud project set; two troubleshooting rows (picker delay, 429 from the shared project) |
| 2026-11-17 | Step 0.6: $5 budget alert created (alerts only, promotional credits excluded) |
| 2026-11-17 | Step 0.5 done. Added "What it does" notes for every command and option in steps 0.1 to 0.6 |
| 2026-11-17 | Step 0.7 started: key requested; CFBD tiers and terms recorded |
| 2026-11-17 | Step 0.7 done: key stored in a gitignored `.env`; `.env.example` added |
| 2026-11-17 | Added "Reviewing and committing in VS Code" to the start of the guide |
| 2026-11-23 | Part 1 added: the first version built and deployed in one session by Claude Code; every cloud command recorded (services, Firestore, registry, secret, service accounts, Workload Identity Federation, pipeline, scheduler, uptime alert) |
| 2026-11-25 | Review round with Lori: links to other sites open in a new tab (`ExternalLink`); new game card design (navy strip with FINAL, UPCOMING, or KICKED OFF and the date; "at Kansas · Memorial Stadium" line; a dash for scores of games that have kicked off); "one service for pages and API" written up as a decision in ARCHITECTURE.md; teaching comments added through the Go and React code ("Go note", "React note", "TypeScript note"); the in-memory test store now reports unknown calls remaining as -1, like Firestore; stale `/readyz`, CDN, and Firebase mentions fixed; the TV network idea added to the SRS as future feature F1 |
| 2026-11-25 | SRS published as `docs/SRS.md` and linked from the README, About page, and docs. Game cards: every team name is blue, and only an underline marks a link to a team page; losing names stay blue (only the losing score is softened); a hover message explains teams with no page, and a note above the games says which names are links, for phones |
