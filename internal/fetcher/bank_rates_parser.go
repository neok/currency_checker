package fetcher

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type BankRatesFeedParser struct{}

func NewBankRatesFeedParser() *BankRatesFeedParser {
	return &BankRatesFeedParser{}
}

type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []struct {
			PubDate     string `xml:"pubDate"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

func (p *BankRatesFeedParser) Parse(body io.Reader, currency string) (Rate, error) {
	var feed rssFeed
	if err := xml.NewDecoder(body).Decode(&feed); err != nil {
		return Rate{}, fmt.Errorf("parse xml: %w", err)
	}
	return parseRate(feed, strings.ToUpper(currency))
}

func parseRate(feed rssFeed, currency string) (Rate, error) {
	if len(feed.Channel.Items) == 0 {
		return Rate{}, fmt.Errorf("feed has no items")
	}
	item := feed.Channel.Items[0]

	sourceDate, err := time.Parse(time.RFC1123Z, item.PubDate)
	if err != nil {
		return Rate{}, fmt.Errorf("parse pubDate %q: %w", item.PubDate, err)
	}

	rate, ok := extractRate(item.Description, currency)
	if !ok {
		return Rate{}, fmt.Errorf("currency %q not found in feed", currency)
	}

	return Rate{
		Currency:   currency,
		Rate:       rate,
		SourceDate: sourceDate,
	}, nil
}

func extractRate(description, currency string) (float64, bool) {
	parts := strings.Fields(description)
	for i := 0; i+1 < len(parts); i += 2 {
		if parts[i] == currency {
			v, err := strconv.ParseFloat(parts[i+1], 64)
			if err != nil {
				return 0, false
			}
			return v, true
		}
	}
	return 0, false
}
