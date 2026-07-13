package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

type RSSChannel struct {
	Title string    `xml:"title"`
	Items []RSSItem `xml:"item"`
}

type RSSFeed struct {
	Channel RSSChannel `xml:"channel"`
}

func FetchRSSFeeds(feedURLs []string) (string, error) {
	if len(feedURLs) == 0 {
		return "", nil
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	var summary string

	for _, url := range feedURLs {
		resp, err := client.Get(url)
		if err != nil {
			summary += fmt.Sprintf("❌ Error al leer feed %s: %v\n", url, err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var feed RSSFeed
		if err := xml.Unmarshal(body, &feed); err != nil {
			summary += fmt.Sprintf("❌ Error al procesar XML de %s: %v\n", url, err)
			continue
		}

		summary += fmt.Sprintf("📰 **%s**\n", feed.Channel.Title)
		
		// Get top 3 items
		limit := 3
		if len(feed.Channel.Items) < 3 {
			limit = len(feed.Channel.Items)
		}
		
		for i := 0; i < limit; i++ {
			item := feed.Channel.Items[i]
			summary += fmt.Sprintf("- %s (%s)\n", item.Title, item.Link)
		}
		summary += "\n"
	}

	return summary, nil
}
