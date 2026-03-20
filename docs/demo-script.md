# Seven Test TUI — Demo Script

> Monday leadership demo. Walk through each section live. ~10 minutes.

## Pre-Demo Checklist

- [ ] AWS SSO session active: `aws sso login --profile seven-dev`
- [ ] `export AWS_PROFILE=seven-dev`
- [ ] `export CLIENT_ID="<your-cognito-client-id>"`
- [ ] Terminal font size bumped up for screen share
- [ ] Prefix environment ready (e.g. `SE7-2001` or `int-dev`)
- [ ] All tests green: `go test ./...`

---

## 1. Launch & Authenticate

- Run `./seven-test-tui`
- Enter Cognito username and password when prompted
- **Talking point**: "No AWS console needed — the TUI handles auth via Cognito and uses your SSO session for DynamoDB writes."

## 2. Choose a Prefix Environment

- Arrow keys to highlight a prefix (e.g. `int-dev`), press `Enter`
- **Dev environment**: leave prefix blank — this connects to `https://dev.api.playtheseven.com` in read-only mode (orange border, `READ-ONLY` badge, write controls hidden)
- **Branch environment**: type a prefix like `SE7-2001` for full read/write access
- **Talking point**: "Dev is view-only — you can browse gameweeks and fixtures but can't modify anything. This protects shared dev data. For testing, use a branch prefix."

## 3. Browse Gameweeks

- Gameweek list loads automatically
- Navigate with `↑`/`↓` or `j`/`k`
- Select a gameweek with `Enter`
- **Talking point**: "Gameweeks are fetched from the API — same data the mobile app sees."

## 4. Make a Gameweek Current

- With a gameweek selected, press `m`
- This updates the gameweek's start date in DynamoDB so the mobile app treats it as the active gameweek
- **Talking point**: "One keypress. No console diving."

## 5. Batch Update All Fixtures

- Press `b` to open the batch modal
- Use `↑`/`↓` to cycle through presets:
  - **Pre-Match** — resets all fixtures: period → `PRE_MATCH`, scores cleared, start date pushed to future
  - **Kickoff** — all fixtures go live: period → `FIRST_HALF`, status → `IN_PLAY`, clock → 0:00
  - **Half Time** — period → `HALF_TIME`, clock → 45:00
  - **Second Half** — period → `SECOND_HALF`, clock → 45:00
  - **Full Time** — period → `FULL_TIME`, clock → 90:00
- Press `Enter` to confirm
- Press `r` to refresh and verify all fixtures updated
- **Talking point**: "One action updates every fixture in the gameweek. Replaces manually editing each row in DynamoDB — saves minutes per test cycle."

## 6. Edit a Single Fixture — Period & Scores

- Select a fixture from the table, press `e`
- In the edit modal use `Tab` to move between fields:
  - Change **period** (cycle through PRE_MATCH → FIRST_HALF → HALF_TIME → SECOND_HALF → FULL_TIME)
  - Set **home score** and **away score** with `↑`/`↓`
  - Adjust **clock time** (minutes and seconds)
- Press `Enter` to save
- Press `r` to refresh and confirm
- **Talking point**: "Individual editing lets you set up specific scenarios — e.g. one match at half time 2-1 while others are still pre-match."

## 7. Add Goals to a Fixture

- Select a live fixture, press `g` to open the goal modal
- For each goal:
  - Pick a **player** from the fixture's squad
  - Set the **time** (minute)
- Save and refresh
- **Talking point**: "Goal scorers flow through to the mobile app's live match view and scoring. This is how we test goal notifications and points calculation."

## 8. Verify on Mobile

- Open the Seven app on a test device
- Show the gameweek reflecting the changes just made
- **Talking point**: "Everything we did in the terminal is immediately visible in the app. The TUI writes directly to the same DynamoDB tables the backend reads from."

---

## Handling Questions

### "Can we add feature X?"

> "We practice TDD — we write a failing test for the new behaviour first, then implement. Let me show you."
>
> Run: `go test ./internal/models/ -v`
>
> Point out table-driven tests for validation, presets, and state transitions.

### "How confident are we in data integrity?"

> "Every fixture update runs through validation before writing — period must be valid, scores non-negative, clock times within range for the period. All covered by unit tests."

### "What's next?"

- Goal scorer management improvements
- Customer team selection (7-player assignment flow)
- Real-time update feedback via DynamoDB Streams
- Close gameweek support (EventBridge integration)

---

## Quick Recovery

| Problem | Fix |
|---|---|
| Auth fails | `aws sso login --profile seven-dev` then restart |
| Stale data | Press `r` to refresh |
| Wrong prefix | Press `Esc` back to prefix selection |
| Fixture won't save | Check status bar for validation error |
