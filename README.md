# google-reviews-pp-cli

Fetch Google Maps reviews and rating summaries — **no API key required**.

Reverse-engineered from the Google Maps internal `listentitiesreviews` endpoint.
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

### Get reviews for a business

Pass a Google Maps URL or raw CID (`0xHEX:0xHEX` from the URL's `data=` param):

```bash
# From a Google Maps URL
google-reviews-pp-cli reviews "https://www.google.com/maps/place/...data=!3m5!1s0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# From a CID directly
google-reviews-pp-cli reviews "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

### Sort and filter

```bash
# Newest reviews first
google-reviews-pp-cli reviews --sort newest "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# Highest rated first
google-reviews-pp-cli reviews --sort highest "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"

# Fetch all available reviews (auto-paginates up to API limit)
google-reviews-pp-cli reviews --all "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
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

### JSON output

```bash
google-reviews-pp-cli reviews --json "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
google-reviews-pp-cli summary --json "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

### Different language/country

```bash
google-reviews-pp-cli reviews --lang fr --country fr "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
```

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
| `doctor` | Check auth and connectivity |

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Output as JSON |
| `--sort` | relevant | `relevant`, `newest`, `highest`, `lowest` |
| `--count` | 20 | Reviews per page (max 20) |
| `--all` | false | Auto-paginate all reviews |
| `--offset` | 0 | Starting offset |
| `--lang` | en | Language code |
| `--country` | us | Country code |
| `--dry-run` | false | Print request URL without fetching |

## Known Gaps

- **Full pagination:** Google's API uses internal cursor tokens; `--all` fetches ~300 reviews before the API returns 500. Cursor extraction from the response body would unlock full pagination.
- **Place search:** Finding a CID by business name (rather than copy-pasting from a Maps URL) is not yet implemented.

## License

Apache-2.0. See [LICENSE](LICENSE).
