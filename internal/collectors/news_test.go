package collectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ko5tas/us-military-tracker/internal/models"
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
		expectedQ := "\"aircraft carrier\" OR \"carrier strike group\" OR \"navy deployment\" OR \"US military\""
		if q.Get("q") != expectedQ {
			t.Errorf("expected q=%q, got %q", expectedQ, q.Get("q"))
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

	if len(feeds) != 6 {
		t.Fatalf("expected 6 feeds, got %d", len(feeds))
	}

	expectedFeeds := []string{
		"https://www.defense.gov/DesktopModules/ArticleCS/RSS.ashx?max=20&ContentType=1",
		"https://www.dvidshub.net/rss/news",
		"https://news.usni.org/feed",
		"https://news.usni.org/category/fleet-tracker/feed",
		"https://www.navalnews.com/feed/",
		"https://www.twz.com/feed",
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

func TestTagFleetNews(t *testing.T) {
	items := []models.NewsItem{
		{Title: "USS Ford Carrier Strike Group Arrives in Eastern Mediterranean", Description: "The Gerald R. Ford CSG deployed..."},
		{Title: "Air Force promotes new general", Description: "Ceremony held at Pentagon"},
		{Title: "Navy destroyer conducts patrol", Description: "USS Benfold operates in Pacific"},
		{Title: "Budget proposal for 2027", Description: "Congress reviews defense spending"},
		{Title: "USNI Fleet Tracker: Weekly Update", Description: "CVN-72 Lincoln in Arabian Sea"},
	}

	tagFleetNews(items)

	// Fleet items should be tagged
	if items[0].Tag != "fleet" {
		t.Errorf("item[0] (carrier title) should be tagged fleet, got %q", items[0].Tag)
	}
	if items[2].Tag != "fleet" {
		t.Errorf("item[2] (destroyer/navy) should be tagged fleet, got %q", items[2].Tag)
	}
	if items[4].Tag != "fleet" {
		t.Errorf("item[4] (fleet tracker/CVN) should be tagged fleet, got %q", items[4].Tag)
	}

	// Non-fleet items should NOT be tagged
	if items[1].Tag != "" {
		t.Errorf("item[1] (Air Force promotion) should not be tagged, got %q", items[1].Tag)
	}
	if items[3].Tag != "" {
		t.Errorf("item[3] (budget) should not be tagged, got %q", items[3].Tag)
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple tags", "<p>Hello <b>world</b></p>", "Hello world"},
		{"entities", "AT&amp;T &lt;test&gt; &quot;quoted&quot;", "AT&T <test> \"quoted\""},
		{"nested", "<div><p>USS <em>Lincoln</em> (CVN-72) in <a href='#'>Arabian Sea</a></p></div>", "USS Lincoln (CVN-72) in Arabian Sea"},
		{"whitespace collapse", "<p>Line one</p>  <p>Line two</p>", "Line one Line two"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTMLTags(tt.input)
			if got != tt.want {
				t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFetchRSSFeedWithContentEncoded(t *testing.T) {
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Fleet Tracker</title>
    <item>
      <title>USNI Fleet Tracker: Feb 9, 2026</title>
      <link>https://example.com/fleet-tracker</link>
      <description>These are the approximate positions of the U.S. Navy's deployed carrier strike groups... truncated</description>
      <content:encoded><![CDATA[<p>USS <em>George Washington</em> (CVN-73) is in Yokosuka, Japan.</p><p>USS <em>Abraham Lincoln</em> (CVN-72) is operating in the Arabian Sea.</p><p>USS <em>Gerald R. Ford</em> (CVN-78) is in the Eastern Mediterranean.</p>]]></content:encoded>
      <pubDate>Mon, 09 Feb 2026 16:12:57 +0000</pubDate>
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

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	// Should use content:encoded (full text) instead of truncated description
	desc := items[0].Description
	if !strings.Contains(desc, "Abraham Lincoln") {
		t.Errorf("description should contain full content:encoded text with Lincoln, got: %s", desc)
	}
	if !strings.Contains(desc, "Arabian Sea") {
		t.Errorf("description should contain 'Arabian Sea', got: %s", desc)
	}
	if strings.Contains(desc, "<p>") || strings.Contains(desc, "<em>") {
		t.Errorf("description should have HTML tags stripped, got: %s", desc)
	}
}
