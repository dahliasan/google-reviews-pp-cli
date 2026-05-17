---
name: pp-google-reviews
description: "Fetch Google Maps reviews and rating summaries for any business — no API key required. Pass a Google Maps URL or CID."
author: "dahliasan"
license: "Apache-2.0"
argument-hint: "reviews <maps-url-or-cid> | summary <maps-url-or-cid>"
allowed-tools: "Bash"
metadata:
  openclaw:
    requires:
      bins:
        - google-reviews-pp-cli
---

# Google Reviews — Printing Press CLI

Fetch reviews and rating summaries from Google Maps **without any API key**.
Works by reading your Chrome `NID` cookie via `agent-browser`.

## Prerequisites: Install the CLI

```bash
go install github.com/dahliasan/google-reviews-pp-cli/cmd/google-reviews-pp-cli@latest
```

Also install `agent-browser` for automatic cookie auth (macOS/Linux):
```bash
npm install -g @mvanhorn/agent-browser
```

Or set `GOOGLE_NID` to your NID cookie value to bypass agent-browser entirely.

Verify: `google-reviews-pp-cli --version`

## Finding a CID

The CID is embedded in Google Maps URLs as `0xHEX:0xHEX`:
```
https://www.google.com/maps/place/...data=!3m5!1s0x89c258bc949d58cf:0x84ac8a2dc2535dc2!...
                                          ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
```

You can pass the full URL or just the raw `0xHEX:0xHEX` segment.

## Core Commands

### `reviews` — fetch reviews for a business

```bash
# From a Google Maps URL
google-reviews-pp-cli reviews "https://www.google.com/maps/place/..."

# From a CID directly
google-reviews-pp-cli reviews "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# Newest first, JSON output
google-reviews-pp-cli reviews --sort newest --json "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# Fetch all available reviews (auto-paginates, stops gracefully at API limit ~300)
google-reviews-pp-cli reviews --all --json "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

**Flags:**
- `--sort` — `relevant` (default), `newest`, `highest`, `lowest`
- `--count N` — reviews per page, max 20 (default 20)
- `--all` — auto-paginate
- `--offset N` — starting offset
- `--lang` — language code, e.g. `en`, `zh`, `fr` (default `en`)
- `--country` — country code, e.g. `sg`, `us` (default `us`)
- `--json` — JSON output

**JSON review object:**
```json
{
  "review_id": "...",
  "rating": 5,
  "author": "Jane Smith",
  "author_id": "...",
  "date": "3 months ago",
  "timestamp_ms": 1769986041693,
  "text": "Full review text...",
  "language": "en",
  "review_url": "https://www.google.com/maps/reviews/..."
}
```

### `summary` — rating distribution for a business

```bash
google-reviews-pp-cli summary "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
google-reviews-pp-cli summary --json "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

**JSON output:**
```json
{"stars_5": 154, "stars_4": 3, "stars_3": 1, "stars_2": 1, "stars_1": 1, "total": 160}
```

### `doctor` — verify auth and connectivity

```bash
google-reviews-pp-cli doctor
```

## Agent Mode

Add `--agent` for clean JSON output with no prompts:
```bash
google-reviews-pp-cli reviews --agent "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `google-reviews-pp-cli --help`
2. **Starts with `install`** → install instructions (see Prerequisites)
3. **`reviews <url-or-cid> [flags]`** → fetch reviews
4. **`summary <url-or-cid> [flags]`** → rating distribution
5. **Anything else** → execute as CLI command with `--agent`

## Known Limitations

- `--all` pagination stops at ~300 reviews. Google's API uses internal cursor tokens; offset-based pagination returns 500 beyond a threshold. Full pagination would require cursor extraction.
- No business search by name. You need the CID from a Google Maps URL.
- Requires a Chrome NID cookie. Fresh/unauthenticated requests return empty results. `agent-browser` reads it automatically; `GOOGLE_NID` env var is the manual override.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error |
| 5 | API error (HTTP non-200) |
| 10 | Config / cookie error |
