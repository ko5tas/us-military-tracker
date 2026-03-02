package collectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchGNews(t *testing.T) {
	gnewsJSON := `{
		"totalArticles": 2,
		"articles": [
			{
				"title": "US Navy deploys carrier group to Pacific",
				"description": "The USS Reagan carrier strike group...",
				"url": "https://example.com/navy-deployment",
				"source": {"name": "Defense News"},
				"publishedAt": "2026-03-02T10:00:00Z"
			},
			{
				"title": "Air Force tests new bomber",
				"description": "The B-21 Raider completes test flight...",
				"url": "https://example.com/b21-test",
				"source": {"name": "Military Times"},
				"publishedAt": "2026-03-01T08:30:00Z"
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		q := r.URL.Query()
		if q.Get("token") != "test-api-key" {
			t.Errorf("expected token=test-api-key, got %q", q.Get("token"))
		}
		if q.Get("q") != "US military" {
			t.Errorf("expected q=US military, got %q", q.Get("q"))
		}
		if q.Get("lang") != "en" {
			t.Errorf("expected lang=en, got %q", q.Get("lang"))
		}
		if q.Get("max") != "50" {
			t.Errorf("expected max=50, got %q", q.Get("max"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(gnewsJSON))
	}))
	defer ts.Close()

	items, err := fetchGNews(context.Background(), "test-api-key", ts.URL)
	if err != nil {
		t.Fatalf("fetchGNews returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Check first article
	if items[0].Title != "US Navy deploys carrier group to Pacific" {
		t.Errorf("item[0].Title: got %q, want %q", items[0].Title, "US Navy deploys carrier group to Pacific")
	}
	if items[0].Description != "The USS Reagan carrier strike group..." {
		t.Errorf("item[0].Description: got %q, want %q", items[0].Description, "The USS Reagan carrier strike group...")
	}
	if items[0].URL != "https://example.com/navy-deployment" {
		t.Errorf("item[0].URL: got %q, want %q", items[0].URL, "https://example.com/navy-deployment")
	}
	if items[0].Source != "Defense News" {
		t.Errorf("item[0].Source: got %q, want %q", items[0].Source, "Defense News")
	}
	if items[0].PublishedAt.Year() != 2026 || items[0].PublishedAt.Month() != 3 || items[0].PublishedAt.Day() != 2 {
		t.Errorf("item[0].PublishedAt: got %v", items[0].PublishedAt)
	}

	// Check second article
	if items[1].Title != "Air Force tests new bomber" {
		t.Errorf("item[1].Title: got %q, want %q", items[1].Title, "Air Force tests new bomber")
	}
	if items[1].Source != "Military Times" {
		t.Errorf("item[1].Source: got %q, want %q", items[1].Source, "Military Times")
	}
}

func TestFetchGNewsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := fetchGNews(context.Background(), "test-key", ts.URL)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestFetchRSSFeed(t *testing.T) {
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>DOD News</title>
    <item>
      <title>Exercise Baltic Shield begins</title>
      <link>https://example.com/baltic-shield</link>
      <description>NATO allies begin exercise...</description>
      <pubDate>Mon, 02 Mar 2026 12:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Army modernization update</title>
      <link>https://example.com/army-mod</link>
      <description>New equipment fielding continues</description>
      <pubDate>Sun, 01 Mar 2026 09:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(rssXML))
	}))
	defer ts.Close()

	items, err := fetchRSSFeed(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("fetchRSSFeed returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].Title != "Exercise Baltic Shield begins" {
		t.Errorf("item[0].Title: got %q, want %q", items[0].Title, "Exercise Baltic Shield begins")
	}
	if items[0].URL != "https://example.com/baltic-shield" {
		t.Errorf("item[0].URL: got %q, want %q", items[0].URL, "https://example.com/baltic-shield")
	}
	if items[0].Description != "NATO allies begin exercise..." {
		t.Errorf("item[0].Description: got %q, want %q", items[0].Description, "NATO allies begin exercise...")
	}
	if items[0].Source != "DOD News" {
		t.Errorf("item[0].Source: got %q, want %q", items[0].Source, "DOD News")
	}
	if items[0].PublishedAt.Year() != 2026 || items[0].PublishedAt.Month() != 3 || items[0].PublishedAt.Day() != 2 {
		t.Errorf("item[0].PublishedAt: got %v", items[0].PublishedAt)
	}

	if items[1].Title != "Army modernization update" {
		t.Errorf("item[1].Title: got %q, want %q", items[1].Title, "Army modernization update")
	}
}

func TestFetchRSSFeedError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := fetchRSSFeed(context.Background(), ts.URL)
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

func TestDefaultRSSFeeds(t *testing.T) {
	feeds := DefaultRSSFeeds()

	if len(feeds) != 2 {
		t.Fatalf("expected 2 feeds, got %d", len(feeds))
	}

	expectedFeeds := []string{
		"https://www.defense.gov/DesktopModules/ArticleCS/RSS.ashx?max=20&ContentType=1",
		"https://www.dvidshub.net/rss/news",
	}

	for i, expected := range expectedFeeds {
		if feeds[i] != expected {
			t.Errorf("feed[%d]: got %q, want %q", i, feeds[i], expected)
		}
	}
}

func TestCollectNewsWithGNewsKey(t *testing.T) {
	gnewsJSON := `{
		"totalArticles": 1,
		"articles": [
			{
				"title": "GNews Article",
				"description": "From GNews",
				"url": "https://example.com/gnews",
				"source": {"name": "GNews Source"},
				"publishedAt": "2026-03-02T10:00:00Z"
			}
		]
	}`

	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>RSS Article</title>
      <link>https://example.com/rss</link>
      <description>From RSS</description>
      <pubDate>Mon, 02 Mar 2026 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	gnewsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(gnewsJSON))
	}))
	defer gnewsServer.Close()

	rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(rssXML))
	}))
	defer rssServer.Close()

	// Override defaults for testing
	origFeeds := DefaultRSSFeeds
	origBaseURL := gnewsBaseURL
	DefaultRSSFeeds = func() []string { return []string{rssServer.URL} }
	gnewsBaseURL = gnewsServer.URL
	defer func() {
		DefaultRSSFeeds = origFeeds
		gnewsBaseURL = origBaseURL
	}()

	items, err := CollectNews(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("CollectNews returned error: %v", err)
	}

	if len(items) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(items))
	}

	// Check that we have articles from both sources
	var hasGNews, hasRSS bool
	for _, item := range items {
		if item.Title == "GNews Article" {
			hasGNews = true
		}
		if item.Title == "RSS Article" {
			hasRSS = true
		}
	}

	if !hasGNews {
		t.Error("expected GNews article in results")
	}
	if !hasRSS {
		t.Error("expected RSS article in results")
	}
}

func TestCollectNewsWithoutGNewsKey(t *testing.T) {
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>RSS Only Article</title>
      <link>https://example.com/rss-only</link>
      <description>Only RSS feeds</description>
      <pubDate>Mon, 02 Mar 2026 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(rssXML))
	}))
	defer rssServer.Close()

	origFeeds := DefaultRSSFeeds
	defer func() { DefaultRSSFeeds = origFeeds }()
	DefaultRSSFeeds = func() []string { return []string{rssServer.URL} }

	items, err := CollectNews(context.Background(), "")
	if err != nil {
		t.Fatalf("CollectNews returned error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item (RSS only), got %d", len(items))
	}

	if items[0].Title != "RSS Only Article" {
		t.Errorf("item[0].Title: got %q, want %q", items[0].Title, "RSS Only Article")
	}
}

func TestCollectNewsFailedFeedContinues(t *testing.T) {
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Good Feed</title>
    <item>
      <title>Good Article</title>
      <link>https://example.com/good</link>
      <description>This feed works</description>
      <pubDate>Mon, 02 Mar 2026 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(rssXML))
	}))
	defer goodServer.Close()

	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badServer.Close()

	origFeeds := DefaultRSSFeeds
	defer func() { DefaultRSSFeeds = origFeeds }()
	DefaultRSSFeeds = func() []string { return []string{badServer.URL, goodServer.URL} }

	items, err := CollectNews(context.Background(), "")
	if err != nil {
		t.Fatalf("CollectNews returned error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item (from good feed), got %d", len(items))
	}

	if items[0].Title != "Good Article" {
		t.Errorf("item[0].Title: got %q, want %q", items[0].Title, "Good Article")
	}
}
