package fetcher

import (
	"strings"
	"testing"
	"time"
)

const sampleFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <pubDate>Mon, 27 Apr 2026 14:15:00 +0300</pubDate>
      <description>USD 1.0850 GBP 0.8420 EUR 1.0000</description>
    </item>
  </channel>
</rss>`

func TestBankRatesFeedParser_Parse(t *testing.T) {
	expectedDate, _ := time.Parse(time.RFC1123Z, "Mon, 27 Apr 2026 14:15:00 +0300")

	tests := []struct {
		name     string
		body     string
		currency string
		wantRate float64
		wantErr  string
	}{
		{
			name:     "usd",
			body:     sampleFeed,
			currency: "USD",
			wantRate: 1.0850,
		},
		{
			name:     "gbp lowercase input is normalized",
			body:     sampleFeed,
			currency: "gbp",
			wantRate: 0.8420,
		},
		{
			name:     "currency missing",
			body:     sampleFeed,
			currency: "JPY",
			wantErr:  "not found",
		},
		{
			name:     "no items",
			body:     `<?xml version="1.0"?><rss><channel></channel></rss>`,
			currency: "USD",
			wantErr:  "no items",
		},
		{
			name:     "malformed xml",
			body:     `<rss><channel><item`,
			currency: "USD",
			wantErr:  "parse xml",
		},
		{
			name: "bad pubDate",
			body: `<?xml version="1.0"?><rss><channel><item>
				<pubDate>not a date</pubDate>
				<description>USD 1.0</description>
			</item></channel></rss>`,
			currency: "USD",
			wantErr:  "parse pubDate",
		},
	}

	p := NewBankRatesFeedParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, err := p.Parse(strings.NewReader(tt.body), tt.currency)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rate.Rate != tt.wantRate {
				t.Errorf("rate = %v, want %v", rate.Rate, tt.wantRate)
			}
			if rate.Currency != strings.ToUpper(tt.currency) {
				t.Errorf("currency = %q, want %q", rate.Currency, strings.ToUpper(tt.currency))
			}
			if !rate.SourceDate.Equal(expectedDate) {
				t.Errorf("source date = %v, want %v", rate.SourceDate, expectedDate)
			}
		})
	}
}
