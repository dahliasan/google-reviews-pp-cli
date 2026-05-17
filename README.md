# google-reviews-pp-cli

Fetch Google Maps reviews, rating summaries, and full business profile data — **no API key required**.

Reverse-engineered from Google Maps internal endpoints (`listentitiesreviews`, `preview/place`).
Reads cookies from your local Chrome profile automatically (macOS/Linux).

Printed by [@dahliasan](https://github.com/dahliasan) using [Printing Press](https://github.com/mvanhorn/cli-printing-press).

## Install

```bash
go install github.com/dahliasan/google-reviews-pp-cli/cmd/google-reviews-pp-cli@latest
```

Requires Go 1.21+. The binary is placed in `$GOPATH/bin` (usually `~/go/bin`).

### Pre-built binaries

Download from the [latest release](https://github.com/dahliasan/google-reviews-pp-cli/releases/latest) for macOS (arm64/amd64), Linux, and Windows.

On macOS, clear Gatekeeper quarantine after download:
```bash
xattr -d com.apple.quarantine google-reviews-pp-cli
chmod +x google-reviews-pp-cli
```

## Auth

No API key needed. The CLI automatically reads your `NID` cookie from Chrome using `agent-browser`.

**macOS/Linux:** Install [agent-browser](https://github.com/mvanhorn/agent-browser) once, then the CLI handles everything:

```bash
npm install -g @mvanhorn/agent-browser
```

**Manual override:** Set `GOOGLE_NID` to skip agent-browser entirely:

```bash
export GOOGLE_NID="your-nid-cookie-value"
```

To find your NID: open DevTools on google.com → Application → Cookies → `NID`.

## Usage

### Reviews

Pass a Google Maps URL or raw CID (`0xHEX:0xHEX` from the URL's `data=` param):

```bash
# From a Google Maps URL
google-reviews-pp-cli reviews "https://www.google.com/maps/place/...data=!3m5!1s0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# From a CID directly
google-reviews-pp-cli reviews "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# Newest first, fetch all (auto-paginates up to API limit ~300)
google-reviews-pp-cli reviews --sort newest --all "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# JSON output with photos, owner responses, Local Guide badge, visit date
google-reviews-pp-cli reviews --json "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

**Review flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--sort` | `relevant` | `relevant`, `newest`, `highest`, `lowest` |
| `--count` | `20` | Reviews per page (max 20) |
| `--all` | false | Auto-paginate all reviews |
| `--offset` | `0` | Starting offset |
| `--lang` | `en` | Language code (e.g. `fr`, `zh`) |
| `--country` | `us` | Country code (e.g. `au`, `sg`) |
| `--raw` | false | Dump raw API JSON for debugging |

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
  "review_url": "https://www.google.com/maps/reviews/...",
  "photos": ["https://lh5.googleusercontent.com/p/..."],
  "owner_response": "Thank you for the kind words!",
  "is_local_guide": true,
  "visit_date": "Visited March 2025"
}
```

### Rating summary

```bash
google-reviews-pp-cli summary "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

Output:
```
Total: 7,713 reviews

5 ★   64.3%  ████████████  4,958
4 ★   24.5%  ████          1,885
3 ★    7.4%  █               573
2 ★    2.3%                  175
1 ★    1.6%                  122
```

### Business profile

Fetch the full Google Maps business profile — name, address, phone, website, hours, rating, coordinates, category, and description:

```bash
google-reviews-pp-cli business get "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# JSON output
google-reviews-pp-cli business get --json "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

**JSON business profile object:**
```json
{
  "name": "Example Business",
  "address": "123 Main St, Sydney NSW",
  "phone": "+61 2 9999 0000",
  "website": "https://example.com",
  "rating": 4.8,
  "review_count": 312,
  "category": "Barber shop",
  "lat": -33.8688,
  "lng": 151.2093,
  "hours": {
    "Monday": "9:00 AM – 6:00 PM",
    "Tuesday": "9:00 AM – 6:00 PM"
  },
  "description": "...",
  "feature_id": "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
}
```

### Business photos

Fetch photo URLs from a business's Google Maps profile:

```bash
# Profile photos
google-reviews-pp-cli business photos "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# Include photos uploaded by reviewers (paginates through reviews)
google-reviews-pp-cli business photos --all-reviews "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# JSON array of URLs
google-reviews-pp-cli business photos --json "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

Photo URLs are in `lh5.googleusercontent.com` format and accept size suffixes:
- `=w1600` — width 1600px
- `=w800-h600` — width × height

## Finding a CID

The CID is the business identifier embedded in Google Maps URLs. Look for the `data=` segment:

```
https://www.google.com/maps/place/.../@.../data=!3m5!1s0x89c258bc949d58cf:0x84ac8a2dc2535dc2!...
                                                      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                                      This is the CID
```

The format is `0xLO:0xHI` — two 64-bit hex integers.

## Commands

| Command | Description |
|---------|-------------|
| `reviews <cid>` | Fetch reviews (table or JSON) |
| `summary <cid>` | Rating distribution |
| `business get <cid>` | Full business profile (name, address, phone, hours, etc.) |
| `business photos <cid>` | Photo URLs from the business profile |
| `doctor` | Check auth and connectivity |

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Output as JSON |
| `--agent` | false | JSON + compact + no prompts (for automation) |
| `--dry-run` | false | Print request URL without fetching |
| `--timeout` | `30s` | Request timeout |

## Automation

For headless / CI use, set `GOOGLE_NID` instead of relying on agent-browser:

```bash
export GOOGLE_NID="531=..."
google-reviews-pp-cli reviews --agent "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

The NID cookie is valid for approximately 6 months before requiring manual refresh.

## Known Gaps

- **Full pagination:** Google's API uses internal cursor tokens; `--all` fetches ~300 reviews before the API returns 500. Cursor extraction from the response body would unlock full pagination.
- **Place search:** Finding a CID by business name (rather than copy-pasting from a Maps URL) is not yet implemented.

## License

Apache-2.0. See [LICENSE](LICENSE).
