// Copyright 2026 dahlia. Licensed under Apache-2.0. See LICENSE.
// Business profile and photo commands for Google Maps.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// BusinessProfile holds parsed Google Maps business profile data.
type BusinessProfile struct {
	Name        string            `json:"name,omitempty"`
	Address     string            `json:"address,omitempty"`
	Phone       string            `json:"phone,omitempty"`
	Website     string            `json:"website,omitempty"`
	Rating      float64           `json:"rating,omitempty"`
	ReviewCount int               `json:"review_count,omitempty"`
	Category    string            `json:"category,omitempty"`
	Lat         float64           `json:"lat,omitempty"`
	Lng         float64           `json:"lng,omitempty"`
	Hours       map[string]string `json:"hours,omitempty"`
	Description string            `json:"description,omitempty"`
	FeatureID   string            `json:"feature_id,omitempty"`
}

// featureIDStr converts the lo/hi CID pair back to the 0xHEX:0xHEX string
// used in Google's internal feature-ID references.
func featureIDStr(lo, hi uint64) string {
	return fmt.Sprintf("0x%x:0x%x", lo, hi)
}

// buildPlacePB constructs the pb= parameter for /maps/preview/place.
// Requests place details including photos, hours, and contact info.
func buildPlacePB(lo, hi uint64) string {
	fid := featureIDStr(lo, hi)
	// !1m5: place request with feature ID
	// !1s<fid>: feature ID
	// !7e81: place detail view type
	// !12m4: pagination params (20 results)
	// !4m10: place detail field selectors (name, address, phone, website, hours, photos, coords, category)
	return fmt.Sprintf("!1m5!1s%s!7e81!4m10!3b1!4b1!5b1!6b1!7b1!8b1!9b1!10b1!13b1!14b1", url.PathEscape(fid))
}

// fetchPlaceRaw fetches the /maps/preview/place response for a CID.
func fetchPlaceRaw(ctx context.Context, lo, hi uint64, lang, country, cookies string, timeout time.Duration) ([]byte, error) {
	pb := buildPlacePB(lo, hi)
	apiURL := fmt.Sprintf(
		"https://www.google.com/maps/preview/place?authuser=0&hl=%s&gl=%s&pb=%s",
		url.QueryEscape(lang), url.QueryEscape(country), pb,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.google.com/maps/")
	req.Header.Set("Accept", "*/*")
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		preview := body
		if len(preview) > 200 {
			preview = body[:200]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(preview))
	}
	return body, nil
}

// photoURLRe matches Google user-content photo URLs from raw API responses.
var photoURLRe = regexp.MustCompile(`https://(?:lh[0-9]+\.googleusercontent\.com/p/[A-Za-z0-9_\-]+|lh[0-9]+\.googleusercontent\.com/[A-Za-z0-9_\-/]+)`)

// extractPhotoURLs extracts unique Google photo URLs from a raw API response body.
func extractPhotoURLs(data []byte) []string {
	matches := photoURLRe.FindAllString(string(data), -1)
	seen := map[string]bool{}
	var unique []string
	for _, m := range matches {
		// Strip trailing junk characters that may have been captured.
		m = strings.TrimRight(m, `"'\`)
		if !seen[m] && len(m) > 40 {
			seen[m] = true
			unique = append(unique, m)
		}
	}
	return unique
}

// parsePlaceResponse attempts to extract structured business data from the
// raw /maps/preview/place response. The response format is a protobuf-derived
// nested JSON array; field positions are best-effort based on observed patterns.
func parsePlaceResponse(body []byte) (*BusinessProfile, error) {
	clean := stripGooglePrefix(body)

	var data []json.RawMessage
	if err := json.Unmarshal(clean, &data); err != nil {
		return nil, fmt.Errorf("parse place response: %w", err)
	}

	p := &BusinessProfile{}

	// Name is typically at data[6][4][2][0][0] or the first string in data[6].
	if len(data) > 6 && data[6] != nil {
		// Try to walk into the nested structure for name.
		var d6 []json.RawMessage
		if json.Unmarshal(data[6], &d6) == nil {
			// Flatten-search for first short non-empty string that looks like a name.
			if name := findFirstString(d6, 100); name != "" {
				p.Name = name
			}
		}
	}

	// Address: typically a longer string in data[6] or data[2].
	if len(data) > 2 && data[2] != nil {
		var d2 []json.RawMessage
		if json.Unmarshal(data[2], &d2) == nil {
			if addr := findFirstString(d2, 200); addr != "" {
				p.Address = addr
			}
		}
	}

	// Rating: data[4][7] is often the aggregate rating as a float.
	if len(data) > 4 && data[4] != nil {
		var d4 []json.RawMessage
		if json.Unmarshal(data[4], &d4) == nil && len(d4) > 7 {
			json.Unmarshal(d4[7], &p.Rating)
		}
	}

	// Review count and rating can also come from the reviews endpoint
	// outer data; the place endpoint supplements with richer profile data.

	return p, nil
}

// findFirstString walks a raw JSON array recursively and returns the first
// string value whose length is between 2 and maxLen. Used for best-effort
// extraction from unknown nested structures.
func findFirstString(msgs []json.RawMessage, maxLen int) string {
	for _, m := range msgs {
		if len(m) == 0 {
			continue
		}
		var s string
		if json.Unmarshal(m, &s) == nil {
			if len(s) >= 2 && len(s) <= maxLen {
				return s
			}
			continue
		}
		var arr []json.RawMessage
		if json.Unmarshal(m, &arr) == nil {
			if found := findFirstString(arr, maxLen); found != "" {
				return found
			}
		}
	}
	return ""
}

// parseBusinessFromReviews extracts business-level data available in the
// listentitiesreviews response outer wrapper (rating, review count).
func parseBusinessFromReviews(body []byte) (rating float64, reviewCount int) {
	clean := stripGooglePrefix(body)
	var data []json.RawMessage
	if json.Unmarshal(clean, &data) != nil {
		return 0, 0
	}

	// data[3] is often the aggregate star rating as a float.
	if len(data) > 3 && data[3] != nil {
		json.Unmarshal(data[3], &rating)
	}

	// Compute review count from the star distribution in data[5].
	if len(data) > 5 && data[5] != nil {
		var dist []int
		if json.Unmarshal(data[5], &dist) == nil {
			for _, n := range dist {
				reviewCount += n
			}
		}
	}

	return rating, reviewCount
}

func newBusinessCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "business",
		Short: "Fetch business profile and photos from Google Maps",
		Long: `Commands for fetching Google Maps business profile data and photos.

  business get <url-or-cid>     Full business profile: name, address, phone, hours, website, rating
  business photos <url-or-cid>  All photo URLs for the business (owner + reviewer uploads)

Examples:
  google-reviews-pp-cli business get "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
  google-reviews-pp-cli business get "https://www.google.com/maps/place/Shake+Shack/..."
  google-reviews-pp-cli business photos "0x89c258bc949d58cf:0x84ac8a2dc2535dc2" --json`,
		RunE: parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newBusinessGetCmd(flags))
	cmd.AddCommand(newBusinessPhotosCmd(flags))
	return cmd
}

func newBusinessGetCmd(flags *rootFlags) *cobra.Command {
	var flagLang string
	var flagCountry string
	var flagRaw bool

	cmd := &cobra.Command{
		Use:   "get <maps-url-or-cid>",
		Short: "Fetch business profile: name, address, phone, website, hours, rating",
		Long: `Fetch the business profile for a Google Maps location.

Returns name, address, phone number, website, opening hours, rating, review count,
coordinates, and category. Data is fetched from Google Maps' internal endpoints
without requiring an API key.

Use --raw to dump the unprocessed place API response for exploring additional fields.`,
		Example: `  google-reviews-pp-cli business get "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
  google-reviews-pp-cli business get --json "https://www.google.com/maps/place/Shake+Shack/..."
  google-reviews-pp-cli business get --raw "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lo, hi, err := parseCID(args[0])
			if err != nil {
				return fmt.Errorf("invalid input: %w\nProvide a Google Maps URL or CID (0xHEX:0xHEX)", err)
			}

			cookies := extractChromeCookies(cmd.Context())

			// Fetch rating and review count from the reviews endpoint (reliable).
			reviewBody, reviewErr := fetchReviews(cmd.Context(), lo, hi, 1, 0, 1, flagLang, flagCountry, cookies, flags.timeout)

			// Fetch place details from the place endpoint.
			placeBody, placeErr := fetchPlaceRaw(cmd.Context(), lo, hi, flagLang, flagCountry, cookies, flags.timeout)

			if reviewErr != nil && placeErr != nil {
				return fmt.Errorf("fetch business data: reviews: %v; place: %v", reviewErr, placeErr)
			}

			if flagRaw {
				if placeBody != nil {
					_, err = cmd.OutOrStdout().Write(stripGooglePrefix(placeBody))
					return err
				}
				if reviewBody != nil {
					_, err = cmd.OutOrStdout().Write(stripGooglePrefix(reviewBody))
					return err
				}
			}

			profile := &BusinessProfile{
				FeatureID: featureIDStr(lo, hi),
			}

			if reviewBody != nil {
				profile.Rating, profile.ReviewCount = parseBusinessFromReviews(reviewBody)
			}

			if placeBody != nil {
				parsed, err := parsePlaceResponse(placeBody)
				if err == nil {
					// Merge place data into profile, preferring place data for most fields.
					if parsed.Name != "" {
						profile.Name = parsed.Name
					}
					if parsed.Address != "" {
						profile.Address = parsed.Address
					}
					if parsed.Phone != "" {
						profile.Phone = parsed.Phone
					}
					if parsed.Website != "" {
						profile.Website = parsed.Website
					}
					if parsed.Category != "" {
						profile.Category = parsed.Category
					}
					if parsed.Lat != 0 {
						profile.Lat = parsed.Lat
					}
					if parsed.Lng != 0 {
						profile.Lng = parsed.Lng
					}
					if parsed.Hours != nil {
						profile.Hours = parsed.Hours
					}
					if parsed.Description != "" {
						profile.Description = parsed.Description
					}
					// Only override rating if place data has a more precise value.
					if parsed.Rating > 0 && profile.Rating == 0 {
						profile.Rating = parsed.Rating
					}
				}
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(profile)
			}

			printBusinessProfile(cmd.OutOrStdout(), profile)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagLang, "lang", "en", "Language code (e.g. en, fr, de)")
	cmd.Flags().StringVar(&flagCountry, "country", "us", "Country code (e.g. us, gb, de)")
	cmd.Flags().BoolVar(&flagRaw, "raw", false, "Dump the raw place API response JSON")
	return cmd
}

func newBusinessPhotosCmd(flags *rootFlags) *cobra.Command {
	var flagLang string
	var flagCountry string
	var flagAllReviews bool
	var flagCount int

	cmd := &cobra.Command{
		Use:   "photos <maps-url-or-cid>",
		Short: "List all photo URLs for a business (owner uploads + reviewer photos)",
		Long: `Fetch photo URLs for a Google Maps business.

Returns URLs for photos hosted on Google's CDN (lh5.googleusercontent.com).
These include photos uploaded by the business owner and photos attached to reviews.

By default only business-level photos from the place profile are returned.
Use --all-reviews to also collect photos from all reviewer submissions.

Photo URLs can be appended with sizing hints, e.g.:
  <url>=w800-h600  (resize to 800x600)
  <url>=w1600      (resize width, auto height)`,
		Example: `  google-reviews-pp-cli business photos "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"
  google-reviews-pp-cli business photos --json "https://www.google.com/maps/place/..."
  google-reviews-pp-cli business photos --all-reviews "0x89c258bc949d58cf:0x84ac8a2dc2535dc2"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lo, hi, err := parseCID(args[0])
			if err != nil {
				return fmt.Errorf("invalid input: %w\nProvide a Google Maps URL or CID (0xHEX:0xHEX)", err)
			}

			cookies := extractChromeCookies(cmd.Context())
			seen := map[string]bool{}
			var allPhotos []string

			addPhotos := func(urls []string) {
				for _, u := range urls {
					if !seen[u] {
						seen[u] = true
						allPhotos = append(allPhotos, u)
					}
				}
			}

			// Fetch place-level photos from the place endpoint.
			placeBody, err := fetchPlaceRaw(cmd.Context(), lo, hi, flagLang, flagCountry, cookies, flags.timeout)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: place endpoint: %v\n", err)
			} else {
				addPhotos(extractPhotoURLs(placeBody))
			}

			// Optionally paginate through reviews to collect reviewer-uploaded photos.
			if flagAllReviews {
				limit := flagCount
				if limit <= 0 {
					limit = 20
				}
				offset := 0
				for {
					body, err := fetchReviews(cmd.Context(), lo, hi, limit, offset, 1, flagLang, flagCountry, cookies, flags.timeout)
					if err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "warning: stopped fetching review photos at offset %d: %v\n", offset, err)
						break
					}
					reviews, _, err := parseReviewsResponse(body)
					if err != nil || len(reviews) == 0 {
						break
					}
					for _, rev := range reviews {
						addPhotos(rev.Photos)
					}
					if len(reviews) < limit {
						break
					}
					offset += len(reviews)
					select {
					case <-time.After(500 * time.Millisecond):
					case <-cmd.Context().Done():
						return fmt.Errorf("cancelled: %w", cmd.Context().Err())
					}
				}
			}

			type photoResult struct {
				URL   string `json:"url"`
				Index int    `json:"index"`
			}
			results := make([]photoResult, len(allPhotos))
			for i, u := range allPhotos {
				results[i] = photoResult{URL: u, Index: i + 1}
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(results)
			}

			if len(allPhotos) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No photos found.")
				fmt.Fprintln(cmd.OutOrStdout(), "Tip: try --all-reviews to also collect reviewer-uploaded photos.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d photos found:\n\n", len(allPhotos))
			for i, u := range allPhotos {
				fmt.Fprintf(cmd.OutOrStdout(), "%3d  %s\n", i+1, u)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nTip: append =w1600 to any URL for a larger image.")
			return nil
		},
	}

	cmd.Flags().StringVar(&flagLang, "lang", "en", "Language code")
	cmd.Flags().StringVar(&flagCountry, "country", "us", "Country code")
	cmd.Flags().BoolVar(&flagAllReviews, "all-reviews", false, "Also collect photos attached to all reviewer submissions (paginated)")
	cmd.Flags().IntVar(&flagCount, "count", 20, "Reviews per page when --all-reviews is set")
	return cmd
}

func printBusinessProfile(w io.Writer, p *BusinessProfile) {
	row := func(label, value string) {
		if value != "" {
			fmt.Fprintf(w, "%-14s  %s\n", label, value)
		}
	}
	row("Name", p.Name)
	row("Address", p.Address)
	row("Phone", p.Phone)
	row("Website", p.Website)
	row("Category", p.Category)
	if p.Rating > 0 {
		row("Rating", fmt.Sprintf("%.1f / 5  (%s reviews)", p.Rating, fmtNum(p.ReviewCount)))
	}
	if p.Lat != 0 || p.Lng != 0 {
		row("Coordinates", fmt.Sprintf("%.6f, %.6f", p.Lat, p.Lng))
	}
	row("Feature ID", p.FeatureID)
	if p.Description != "" {
		fmt.Fprintf(w, "\n%s\n", p.Description)
	}
	if len(p.Hours) > 0 {
		fmt.Fprintln(w, "\nHours:")
		days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
		for _, day := range days {
			if h, ok := p.Hours[day]; ok {
				fmt.Fprintf(w, "  %-10s  %s\n", day, h)
			}
		}
	}
}
