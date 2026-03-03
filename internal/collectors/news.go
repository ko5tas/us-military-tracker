package collectors

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

// gnewsBaseURL is the base URL for the GNews API. It is a variable so tests
// can replace it with a httptest server URL.
var gnewsBaseURL = "https://gnews.io/api/v4/search"

// DefaultRSSFeeds returns the list of default DoD RSS feed URLs.
// It is a variable (function) so tests can override it.
var DefaultRSSFeeds = func() []string {
	return []string{
		"https://www.defense.gov/DesktopModules/ArticleCS/RSS.ashx?max=20&ContentType=1",
		"https://www.dvidshub.net/rss/news",
		"https://news.usni.org/feed",
		"https://news.usni.org/category/fleet-tracker/feed",
		"https://www.navalnews.com/feed/",
		"https://www.twz.com/feed",
	}
}

// gnewsResponse represents the top-level GNews API response.
type gnewsResponse struct {
	TotalArticles int            `json:"totalArticles"`
	Articles      []gnewsArticle `json:"articles"`
}

// gnewsArticle represents a single article in the GNews API response.
type gnewsArticle struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	URL         string      `json:"url"`
	Source      gnewsSource `json:"source"`
	PublishedAt string      `json:"publishedAt"`
}

// gnewsSource represents the source field of a GNews article.
type gnewsSource struct {
	Name string `json:"name"`
}

// rssDocument represents an RSS 2.0 XML document.
type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

// rssChannel represents the channel element of an RSS feed.
type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

// rssItem represents a single item in an RSS feed.
type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// fetchGNews queries the GNews search API and returns a slice of NewsItems.
func fetchGNews(ctx context.Context, apiKey, baseURL string) ([]models.NewsItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gnews: create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("token", apiKey)
	q.Set("q", "\"aircraft carrier\" OR \"carrier strike group\" OR \"navy deployment\" OR \"US military\"")
	q.Set("lang", "en")
	q.Set("max", "50")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gnews: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gnews: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gnews: read body: %w", err)
	}

	var gnResp gnewsResponse
	if err := json.Unmarshal(body, &gnResp); err != nil {
		return nil, fmt.Errorf("gnews: parse json: %w", err)
	}

	items := make([]models.NewsItem, 0, len(gnResp.Articles))
	for _, a := range gnResp.Articles {
		pubAt, _ := time.Parse(time.RFC3339, a.PublishedAt)
		items = append(items, models.NewsItem{
			Title:       a.Title,
			Description: a.Description,
			URL:         a.URL,
			Source:      a.Source.Name,
			PublishedAt: pubAt,
		})
	}

	return items, nil
}

// fetchRSSFeed fetches and parses an RSS 2.0 XML feed, returning NewsItems.
func fetchRSSFeed(ctx context.Context, feedURL string) ([]models.NewsItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("rss: create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rss: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rss: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rss: read body: %w", err)
	}

	var doc rssDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("rss: parse xml: %w", err)
	}

	channelTitle := doc.Channel.Title
	items := make([]models.NewsItem, 0, len(doc.Channel.Items))
	for _, item := range doc.Channel.Items {
		pubAt, _ := time.Parse(time.RFC1123, item.PubDate)
		if pubAt.IsZero() {
			// Try RFC1123Z as fallback
			pubAt, _ = time.Parse(time.RFC1123Z, item.PubDate)
		}
		items = append(items, models.NewsItem{
			Title:       item.Title,
			URL:         item.Link,
			Description: item.Description,
			Source:      channelTitle,
			PublishedAt: pubAt,
		})
	}

	return items, nil
}

// CollectNews fetches news from GNews API and all default RSS feeds in parallel.
// If gnewsKey is empty, GNews is skipped silently. Failed RSS feeds log a
// warning but do not prevent other feeds from being collected.
func CollectNews(ctx context.Context, gnewsKey string) ([]models.NewsItem, error) {
	var mu sync.Mutex
	var allItems []models.NewsItem
	var wg sync.WaitGroup

	// Fetch GNews if API key is provided
	if gnewsKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := fetchGNews(ctx, gnewsKey, gnewsBaseURL)
			if err != nil {
				log.Printf("WARNING: gnews fetch failed: %v", err)
				return
			}
			mu.Lock()
			allItems = append(allItems, items...)
			mu.Unlock()
		}()
	}

	// Fetch all RSS feeds in parallel
	feeds := DefaultRSSFeeds()
	for _, feedURL := range feeds {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			items, err := fetchRSSFeed(ctx, url)
			if err != nil {
				log.Printf("WARNING: rss feed %s failed: %v", url, err)
				return
			}
			mu.Lock()
			allItems = append(allItems, items...)
			mu.Unlock()
		}(feedURL)
	}

	wg.Wait()

	tagFleetNews(allItems)

	return allItems, nil
}

// fleetKeywords are terms that indicate a news item is about naval fleet movements.
var fleetKeywords = []string{
	"carrier", "strike group", "fleet tracker", "csg", "cvn",
	"deployment", "deployed", "naval", "navy", "warship",
	"destroyer", "cruiser", "amphibious", "uss ",
}

// tagFleetNews scans news items and tags those matching naval/fleet keywords
// with Tag="fleet" so the AI prompt can prioritize them.
func tagFleetNews(items []models.NewsItem) {
	for i := range items {
		text := strings.ToLower(items[i].Title + " " + items[i].Description)
		for _, kw := range fleetKeywords {
			if strings.Contains(text, kw) {
				items[i].Tag = "fleet"
				break
			}
		}
	}
}
