# Game Day Board: Build Guide

This guide records every step taken to build Game Day Board, in order, with the exact commands and the reason for each one. It is written so the whole project can be rebuilt from an empty machine by following it from top to bottom.

**How to use this guide**

- Work through the parts in order. Each step says what to do, the command to run, how to check it worked, and why it matters.
- Commands are for **PowerShell on Windows** unless a step says otherwise. Lines starting with `#` are comments, not commands.
- Text in angle brackets, such as `<your-github-username>`, is a placeholder. Replace it, brackets included.
- Never paste a real key, password, or token into this file. Steps that involve secrets say where the secret is kept instead.
- The requirements and design decisions are in the Software Requirements Specification (SRS). Requirement IDs such as FR-2 or NFR-8, and decision IDs such as D4, refer to it.

**Status key:** ✅ done · ⏳ in progress · ⬜ not started

---

## Contents

- [Part 0: Setup (Milestone 0)](#part-0-setup-milestone-0)
  - [0.1 Install the tools](#01-install-the-tools)
  - [0.2 Create the local repository](#02-create-the-local-repository)
  - [0.3 Keep your email address private in commits](#03-keep-your-email-address-private-in-commits)
  - [0.4 Create the GitHub repository and push](#04-create-the-github-repository-and-push)
  - [0.5 and later: to come](#05-and-later-to-come)
- [Troubleshooting](#troubleshooting)
- [Change log for this guide](#change-log-for-this-guide)

---

## Part 0: Setup (Milestone 0)

Goal: a machine with every tool installed, a public GitHub repository with a working first pipeline, a Google Cloud project with a budget alert, and the project's AI instructions and first skill. No app features yet.

### 0.1 Install the tools

**Status:** ⏳ in progress

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

2. Install the missing tools with `winget`, the package manager built into Windows. Each command may open a permission prompt; accept it.

   ```powershell
   winget install --id GoLang.Go -e
   winget install --id Google.CloudSDK -e
   winget install --id GitHub.cli -e
   ```

   `--id` picks the exact package, and `-e` means "exact match only", so winget never guesses a similarly named package.

3. **Close every terminal and VS Code completely, then reopen them.** Installers add the tools to the system PATH (the list of folders Windows searches for commands), and open windows do not see that change.

4. Check that each tool now answers:

   ```powershell
   go version
   gcloud version
   gh --version
   ```

   **Expected:** each prints a version number. If one says "not recognized", see [Troubleshooting](#troubleshooting).

**Versions used when this guide was written:** to be filled in after installing.

### 0.2 Create the local repository

**Status:** ✅ done (2026-11-16)

**Steps**

1. Create the project folder and start a git repository whose main branch is called `main`:

   ```powershell
   New-Item -ItemType Directory C:\Source\VSCode\game-day-board
   Set-Location C:\Source\VSCode\game-day-board
   git init -b main
   ```

2. Add these files (their full contents are in the repository):

   - `.gitignore`: tells git which files never to track. The most important entries are the secret files (`.env`, key files, service account files). It goes in **before** any other file, so a secret can never be committed by accident.
   - `.gitattributes`: stores text files with Unix line endings (LF). Windows uses CRLF, but the pipeline runs on Linux; without this, the same file can look different on each and cause noisy diffs.
   - `README.md`: says only what is built. Right now that is nothing, so it says "work in progress".
   - `docs/BUILD-GUIDE.md`: this guide.

3. Check that git sees the files and ignores nothing unexpected:

   ```powershell
   git status
   ```

   **Expected:** the four files above are listed as untracked.

**Why it matters:** the repository will be public from the first commit, and everything in its history is public forever, even if it is deleted later. If a secret is ever committed, deleting the file is not enough: the secret must be cancelled (revoked) and replaced.

### 0.3 Keep your email address private in commits

**Status:** ✅ done (2026-11-16). First commit made with the noreply address.

Every git commit records the author's email address, and anyone can read it in a public repository. GitHub provides a private "noreply" address to use instead.

**Steps**

1. On GitHub, open **Settings**, then **Emails**.
2. Turn on **Keep my email addresses private**. GitHub then shows your noreply address. It looks like `<number>+<your-github-username>@users.noreply.github.com`.
3. Also turn on **Block command line pushes that expose my email**. This makes GitHub refuse a push that contains your real address, as a safety net.
4. In the project folder, set the noreply address for **this repository only** (without `--global`, so other projects are not affected):

   ```powershell
   Set-Location C:\Source\VSCode\game-day-board
   git config user.email "<your-noreply-address>"
   git config user.email   # prints the address back, to check it
   ```

5. Make the first commit:

   ```powershell
   git add .
   git commit -m "Initial project setup: README, build guide, gitignore, gitattributes"
   git log --format="%an <%ae>" -1   # shows the author name and email of that commit
   ```

   **Expected:** the last command shows the noreply address, not your personal email.

### 0.4 Create the GitHub repository and push

**Status:** ⬜ not started

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

   - `git remote add origin` saves the GitHub address under the name `origin`, the usual name.
   - `-u` links the local `main` branch to `origin/main`, so later pushes need only `git push`.
   - The first push opens a browser window from Git Credential Manager (included with Git for Windows) to sign in to GitHub. It then remembers the sign-in, so no password or token is stored in the project.

5. Refresh the repository page. **Expected:** the README shows, and the commit author is your GitHub account, not a personal email address.

**About the license:** with no license, people can read public code but have no permission to reuse it. That is fine while building. A common choice for a portfolio project is the MIT License, which lets others reuse the code with credit. It can be added at any time from the repository page (**Add file → Create new file**, name it `LICENSE`, and pick a template).

#### Option B: from the terminal with the GitHub CLI

1. Sign in to GitHub from the terminal. Choose **GitHub.com**, **HTTPS**, and **Login with a web browser** when asked.

   ```powershell
   gh auth login
   gh auth status   # confirms which account you are signed in as
   ```

2. Create the public repository from the local folder, and push the first commit to it, in one command:

   ```powershell
   Set-Location C:\Source\VSCode\game-day-board
   gh repo create game-day-board --public --source . --remote origin --push
   ```

   - `--source .` uses the current folder as the repository.
   - `--remote origin` names the link to GitHub `origin`, the usual name.
   - `--push` uploads the existing commit.

   Creating the repository this way, rather than on the GitHub website, means GitHub does not add its own README or license, so the local and online histories match.

3. Check that it worked:

   ```powershell
   gh repo view --web   # opens the repository in the browser
   ```

   **Expected:** the README shows on the repository page.

### 0.5 and later: to come

These steps are written as they are done:

- Google Cloud: billing account, project, budget alert, free-tier check
- CollegeFootballData.com API key, stored safely
- A "hello" Go endpoint with one test
- A "hello" React page with one test
- The first GitHub Actions pipeline that runs both tests on every push
- Branch protection on `main`
- `CLAUDE.md`, `AI-WORKFLOW.md`, and the clean-code review skill

---

## Troubleshooting

| Problem | Likely cause | Fix |
| --- | --- | --- |
| `go`, `gcloud`, or `gh` is "not recognized" right after installing | The terminal was open before the install, so it has the old PATH | Close all terminals and VS Code, then reopen. If it still fails, sign out of Windows and back in |
| `winget` asks to accept source agreements | First use of winget on this machine | Type `Y` and press Enter |
| A push is rejected with a message about a private email | Step 0.3 was skipped, or the commit was made before the noreply address was set | Set the address (step 0.3), then fix the last commit with `git commit --amend --reset-author --no-edit` and push again |

---

## Change log for this guide

| Date | Change |
| --- | --- |
| 2026-11-16 | Guide started. Part 0 steps 0.1 to 0.4 written |
| 2026-11-16 | Steps 0.2 and 0.3 done; first commit made with the noreply address |
| 2026-11-16 | Step 0.4: added Option A (GitHub website), the path used for this project |
