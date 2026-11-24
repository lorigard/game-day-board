# How AI is used on this project

This project is built with Claude Code, an AI coding agent, and this page says plainly how. The goal is code I understand line by line, not code I merely have.

## The workflow for every change

1. **Plan.** Start from the requirement it serves (the SRS gives each one an ID, such as FR-6 or NFR-3) and agree on the approach before any code is written.
2. **Draft.** The agent drafts the code and its tests, following `CLAUDE.md`: the stack, the folder layout, the comment style, and the testing rules.
3. **Review by agent.** A review pass checks names, function size, comments, error handling, and missing tests.
4. **Review by me.** I read every line, ask about anything I cannot explain, and change what I would write differently.
5. **Test.** Run the tests locally, and add any case the review found missing.
6. **Commit.** A small commit whose message says what changed and why. CI runs every check again before anything deploys.

## How the first version was built

The planning came first and was done together: the idea, the requirements, the architecture, the cost limits, and every design decision are in the SRS. The Google Cloud account, project, billing, budget alert, and API key were set up by hand, step by step, and recorded in `BUILD-GUIDE.md`.

To get a working first version online quickly, the code, tests, pipeline, and cloud resources were then drafted by Claude Code in a single session, from the SRS. Steps 4 and 5 above are the part still in progress for that first version: I am now working through it one piece at a time, using the build guide, until I can explain and rebuild every part myself.

## What the agent is not trusted to decide alone

- Anything that changes cost, permissions, or what is public. The agent proposes; I approve.
- Claims about the project. The README and About page say only what is built and running.
- Secrets. The agent never sees the API key's value; scripts read it from a file that git ignores.

## Still to do

- Write my own review skill (a packaged set of instructions for the agent) that checks a change for clean code, comments that explain why, and missing tests, then use it on every change.
- Try a task where one agent plans and two subagents build the API endpoint and the React component in parallel, then review and merge both. Record what worked and what did not.
