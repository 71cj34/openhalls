# agent_install.md — OpenHalls Setup Instructions for an AI Agent

> **Who should read this:** an AI coding agent with shell access and file access
> to this repo. A human user will point you at this file and expect you to
> finish the installation end-to-end: Python scraper working, data scraped,
> compiled `hallview.exe` downloaded, CLI running.
>
> **You cannot log in to MyTimetable yourself.** You do not have the user's
> McMaster credentials (and must never ask for their password). The one step
> you cannot do alone is obtaining the browser cookie — for that step, ask the
> user to do it and paste the result back to you. Everything else, do yourself.

---

## 1. Goal and done criteria

Starting directory (repo root) is the folder containing `README.md`, `api.py`,
`parse.py`, `requirements.txt`, `cookie.txt`, and the `courses/`, `xml/`,
`state/`, `schedules/`, `src/hallview/` folders.

You are done when ALL of these are true:

1. `courses/<SEM>.json` exists and is non-empty (course catalogue for the
   target semester).
2. `cookie.txt` exists at repo root and contains a fresh, working
   MyTimetable `Cookie` header value (see §4).
3. `xml/<SEM>/` contains ~2000 `*.xml` files (one per course).
4. `state/<SEM>/` contains the matching `*.json` files.
5. `schedules/<SEM>/<SEM>.json` exists (built room schedule).
6. `hallview.exe` (the compiled Go CLI, ~12 MB) sits at **repo root** and
   launches: it shows the logo, the "Data folders" status, and the home menu.
   Databases are built from inside it (menu option `0`).

`SEM` is the semester code, e.g. `3202630` (Fall 2026). Format:
`3` + 4-digit year + `10` (winter) / `20` (spring/summer) / `30` (fall).
So `3202610` = Winter 2026, `3202620` = Summer 2026, `3202630` = Fall 2026.

---

## 2. Prerequisites — check these first

- **OS:** Windows 10/11 (instructions below use PowerShell; `bash` equivalents
  in `README.md` also work).
- **Python 3.10+:** run `python --version`. If missing, STOP and ask the user
  to install it from https://www.python.org/downloads/ (check "Add python to
  PATH"), then continue.
- **Repo root as CWD:** run all commands from repo root. Verify with
  `ls api.py, parse.py, requirements.txt` — all three must exist.
- **Network:** `https://mytimetable.mcmaster.ca` and
  `https://github.com/71cj34/openhalls` must be reachable.
- **Disk/time:** a full semester scrape is ~2000 courses and takes a long
  time (tens of minutes at the default 0.25 s delay). Warn the user before a
  full run.

Install Python deps (from repo root):

```powershell
python -m pip install -r requirements.txt
```

Notes:

- Always use `python -m pip`, not bare `pip` (bare `pip` may point at a
  different Python).
- If `pyfiglet` fails to install/compile, **ignore it** — it is only used by
  the cosmetic `font.py` and is not needed by `api.py` / `parse.py`. The only
  hard dependency is `requests>=2.31`.
- You do NOT need Go. Prefer the prebuilt `.exe` (§6). Only build from source
  if the user explicitly asks (needs Go 1.26+, see `src/hallview/go.mod` and
  `src/hallview/build.bat`).

---

## 3. Step 1 — Course catalogue (no login needed)

This step needs **no cookie**. Do it immediately to confirm Python works and
to discover valid semesters:

```powershell
python api.py --auto
```

- This probes candidate terms and writes `courses/<SEM>.json` for each valid
  one (e.g. `courses/3202630.json`, each several hundred KB).
- If `--auto` finds nothing (network hiccup), retry it once, then fall back
  to an explicit term from the user:
  `python api.py --sem 3202630`
- **Verify:** `courses/<SEM>.json` exists, is valid JSON, and contains entries
  like `{"code": "COMPSCI 1XC3", "title": "..."}`. Ask the user which semester
  they want if more than one was found; default to the latest fall/winter term.

Do NOT proceed to §5 until this file exists — `parse.py` aborts without it
(`Course list not found ... Run step 1 first`).

---

## 4. Step 2 — The cookie (HUMAN step; the hardest part)

### 4a. What the cookie is

`parse.py` calls the authenticated endpoint
`https://mytimetable.mcmaster.ca/api/class-data`. It is not a browser, so the
user must hand it their logged-in browser session: the raw value of the HTTP
`Cookie:` request header from a logged-in MyTimetable request. It is a long
single line (in this repo's history, ~2000 chars) starting something like
`PS_DEVICEFEATURES=...` with many `key=value` pairs separated by `; `.

Rules for the cookie string:

- Copy the **value only** — strip any leading `Cookie:` label.
- It must be **one line** (no newlines). A trailing newline in the file is fine.
- It **expires** (hours–days). If a scrape that used to work suddenly returns
  failures for every course, the cookie is dead — get a fresh one.
- It is a **secret** (it impersonates the user's login). Never print the full
  value to logs/chat, never commit it. It is already in `.gitignore`
  (`cookie.txt`, `.env`). Prefer `--cookie-file` over `--cookie` so the secret
  never lands in shell history.

### 4b. Agent protocol — ask the user, don't improvise

1. If `cookie.txt` already exists at repo root and is > 200 chars, treat it as
   a candidate and **test it** (§5 smoke test) before asking for a new one.
2. Otherwise (or if the test fails), STOP and send the user the exact message
   in the box below. Do not try to guess, fabricate, reuse a sample, or log in
   yourself.

> **Message to send the user (copy verbatim, fill nothing in):**
>
> ---
> I need your MyTimetable browser cookie to download class times. I can't log
> in myself, so please do this (2 minutes, your password never leaves your
> browser):
>
> 1. In Chrome or Edge, go to https://mytimetable.mcmaster.ca and log in.
> 2. Press **F12** to open DevTools → click the **Network** tab.
> 3. On the MyTimetable page, search for any course (e.g. `COMPSCI`) and click
>    it to add it — this makes the page fire requests.
> 4. In the DevTools Network list, click the filter/box and type
>    `class-data`. Click one of the matching requests.
> 5. Go to **Headers** → scroll to **Request Headers** → find `Cookie:`.
>    Right-click its value → **Copy value**.
>    (Firefox: request row → Headers → copy the `Cookie` header the same way.)
> 6. Paste the whole thing back to me here (it's one very long line — send all
>    of it), OR save it yourself as `cookie.txt` in the repo root folder
>    (the same folder as `api.py`) and tell me "cookie file is ready".
>
> Tips: copy the *value only*, not the word `Cookie:`. If DevTools shows no
> `class-data` row, clear the filter, re-run the course search, and look
> again. If the cookie stops working later, we just repeat these steps.
> ---

3. When the user pastes a cookie back, save it yourself (do not ask them to
   create files):
   - Write the pasted string (trimmed) to `cookie.txt` at repo root, single
     line. Overwrite any stale content.
   - Confirm: file length > 200 chars, single line, no `^Cookie:` prefix:
     `python -c "d=open('cookie.txt').read().strip(); print(len(d), d[:40])"`.
     Show the user only the length + first ~20 chars, never the whole secret.
4. Alternative accepted inputs (same secret, different transport — honor any
   of them):
   - `--cookie "..."` CLI arg (discouraged — leaks to bashrc; use only for a
     quick test and then write `cookie.txt`),
   - `$env:MYTIMETABLE_COOKIE="..."` env var,
   - a `.env` file at repo root containing `MYTIMETABLE_COOKIE=...`.
   Resolution order in `parse.py` is: `--cookie` > `--cookie-file` >
   env var > `.env`. Normalize everything to `cookie.txt` + `--cookie-file`
   anyway so later runs just work.

---

## 5. Step 3 — Scrape class data (needs the cookie)

Default pipeline is fetch → process → build. From repo root:

**Smoke test first (always — catches a dead cookie in seconds instead of an
hour in):**

```powershell
python parse.py --semester 3202630 --cookie-file cookie.txt --limit 5 --force
```

- Replace `3202630` with the target semester.
- `--limit 5` fetches only 5 courses; `--force` re-fetches even if XML exists.
- Success looks like `[1/5] Fetched ANTHROP 1AA3 ...` plus
  `Processed 5/5 XML files -> state\3202630`.
- Cookie-dead symptoms: `WARNING ... no class data returned (cookie may be
  expired)`, `no <course>/<offering> found`, `not valid XML`, or every course
  `FAILED`. Fix = fresh cookie (§4), then re-run the smoke test. Do not start
  a full run until the smoke test fetches cleanly.

**Full run (warn the user it takes a while):**

```powershell
python parse.py --semester 3202630 --cookie-file cookie.txt
```

- This fetches ~2000 courses into `xml/<SEM>/`, converts to
  `state/<SEM>/*.json`, and builds `schedules/<SEM>/<SEM>.json`.
- Existing non-empty XML files are skipped by default; `--force` re-fetches.
- Useful variants:
  - `python parse.py --semester 3202630 --no-fetch`
    (XML already downloaded but process/build failed — retry offline, no
    cookie needed),
  - `--delay 0.5` to be gentler on rate limits, or lower it if impatient
    (risk: HTTP 429),
  - `--course-file sample_data.json` for a tiny test list,
  - `--no-process` / `--no-build` to run only selected stages.
- **Verify:** `xml/<SEM>` has ~2000 `.xml` each containing `<course `,
  `state/<SEM>` has the same count of `.json`, and
  `schedules/<SEM>/<SEM>.json` parses as JSON with building names as keys.

If many courses fail at once mid-run: stop, refresh the cookie (§4), re-run
the same command (it resumes — already-fetched files are skipped).

---

## 6. Step 4 — Download the compiled CLI (no build needed)

The Go TUI lives in `src/hallview/` but the user runs the **compiled**
`hallview.exe` from **repo root** (it auto-discovers repo root by looking for
the `schedules/` folder, so root placement matters).

Get the latest release asset from GitHub (`71cj34/openhalls`):

```powershell
# Preferred: GitHub CLI (if installed)
gh release download --repo 71cj34/openhalls --pattern "hallview.exe" --dir . --clobber

# Fallback: PowerShell + GitHub API, no extra tools needed
$rel = Invoke-RestMethod https://api.github.com/repos/71cj34/openhalls/releases/latest
$asset = $rel.assets | Where-Object { $_.name -like "*hallview*.exe" } | Select-Object -First 1
if (-not $asset) { throw "No hallview exe in latest release; see: https://github.com/71cj34/openhalls/releases" }
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile .\hallview.exe
```

- Manual fallback if both fail: open
  https://github.com/71cj34/openhalls/releases/latest in a browser, download
  the `hallview.exe` asset, move it to repo root (next to `api.py`), and tell
  the agent the file is in place.
- **Verify:** `ls hallview.exe` shows a ~10–15 MB file at repo root
  (NOT only `src/hallview/hallview.exe` — that inner copy is a build
  artifact; the runnable one belongs at root).
- Only compile from source if the user insists AND Go is installed:
  `go build -C ./src/hallview -o ..\..\hallview.exe` (requires Go ≥ 1.26 per
  `go.mod`). Then still verify the root-level exe.

---

## 7. Step 5 — Run the CLI

From repo root:

```powershell
.\hallview.exe
```

You must run it from a console session since it is a CLI app. Running it directly will do nothing.

What the user/agent should see and do:

1. ASCII logo, a "Data folders" checklist
   (`Schedules / Courses / State / XML` — all should show OK if §§3–5
   completed), then the `home` menu.
2. **First run: pick `0` (Rebuild Databases).** This builds the SQLite DBs
   under `src/hallview/data/db/` (e.g. `3202630.db`) from the schedules. Wait
   for it to finish.
3. Then use the menu: `1` Search by Room, `2` Search by Time, `3` Search by
   Course, `4` Custom SQL Query, `5` Manage Favorites, `6` Settings
   (e.g. Hide Lab Rooms), `9` Switch Term Database, `8` Exit.
4. CSV exports land in `exports/`; favorites in `src/hallview/data/favs.json`.

If the exe reports missing data folders, you placed it wrong or skipped §5 —
confirm CWD/exe location (repo root) and that `schedules/<SEM>/<SEM>.json`
exists, then pick `0` again.

---

## 8. Troubleshooting (check here before asking the user anything)

| Symptom | Cause → Fix |
|---|---|
| `parse.py: error: fetching needs a cookie` / `Missing browser cookie` | No cookie resolved → do §4, then use `--cookie-file cookie.txt`. |
| `no class data returned (cookie may be expired)` for most/all courses; XML files without `<course `; `no <course>/<offering> found` | Expired cookie → fresh cookie (§4), re-run same command (resumes). |
| `Course list not found: courses/<SEM>.json` | Skipped §3 → `python api.py --sem <SEM>`. |
| `No XML files in xml/<SEM>` with `--no-fetch` | Nothing downloaded yet → run without `--no-fetch` (needs cookie). |
| HTTP 429 / many `FAILED` mid-run | Rate-limited → re-run with `--delay 0.5` or higher; existing files are kept. |
| `pyfiglet ... not compatible` on install | Benign → ignore (only `requests` matters). |
| `hallview.exe` says data folders missing | Wrong exe location/CWD → exe must be at repo root (or cwd = root); check `schedules/` exists. |
| No exe asset in latest release | Releases page changed → check https://github.com/71cj34/openhalls/releases manually or build with Go 1.26+. |
| `go` not recognized | Expected — Go isn't needed; download the exe (§6) instead of building. |

---

## 9. Agent runbook (the order you execute in)

1. CWD = repo root; `python --version` (≥ 3.10); `python -m pip install -r
   requirements.txt` (tolerate `pyfiglet` failure).
2. `python api.py --auto` → confirm `courses/<SEM>.json`; agree on SEM with
   user if ambiguous.
3. Cookie: use existing `cookie.txt` if plausible, else send the §4b message
   and STOP until the user replies. Save their paste to `cookie.txt`.
4. Smoke test: `python parse.py --semester <SEM> --cookie-file cookie.txt
   --limit 5 --force`. On cookie-dead symptoms → back to step 3 (one retry),
   else continue.
5. Full run: `python parse.py --semester <SEM> --cookie-file cookie.txt`
   (warn about duration). Verify `xml/`, `state/`, `schedules/` outputs.
6. Download `hallview.exe` to repo root (§6 commands); verify size/location.
7. Launch check: run `.\hallview.exe`, complete option `0` (Rebuild
   Databases), confirm menu works, then exit (`8`). Report done criteria (§1)
   back to the user with the semester, file counts, and how to re-run the CLI.
8. Never ask for the user's McMaster password; never echo the full cookie;
   never commit `cookie.txt` / `.env` / `*.db` (all gitignored).
