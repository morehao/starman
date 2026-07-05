package discovery

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/morehao/starman/internal/github"
)

type TrendingRepo struct {
	Rank        int
	FullName    string
	Description string
	URL         string
	Stars       int
	Forks       int
	Language    string
	Topics      []string
}

type TrendingOpts struct {
	Since  string
	Lang   string
	Top    int
	Source string
}

type Service struct {
	gh     *github.Client
	http   *http.Client
	rssURL string
}

func NewService(gh *github.Client) *Service {
	return &Service{
		gh:   gh,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

var rssURLMap = map[string]string{
	"daily":   "https://mshibanami.github.io/GitHubTrendingRSS/daily/all.xml",
	"weekly":  "https://mshibanami.github.io/GitHubTrendingRSS/weekly/all.xml",
	"monthly": "https://mshibanami.github.io/GitHubTrendingRSS/monthly/all.xml",
}

func (s *Service) Trending(ctx context.Context, opts TrendingOpts) ([]*TrendingRepo, error) {
	if opts.Source == "search" {
		return s.trendingViaSearch(ctx, opts)
	}
	return s.trendingViaRSS(ctx, opts)
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

var (
	starsRe = regexp.MustCompile(`⭐\s*([\d,]+)`)
	forksRe = regexp.MustCompile(`🍴\s*([\d,]+)`)
	linkRe  = regexp.MustCompile(`github\.com/([^/]+)/([^/?#]+)`)
)

func parseRSS(xmlData []byte) ([]*TrendingRepo, error) {
	var feed rssFeed
	if err := xml.Unmarshal(xmlData, &feed); err != nil {
		return nil, fmt.Errorf("parse RSS XML: %w", err)
	}
	var repos []*TrendingRepo
	for i, item := range feed.Channel.Items {
		repo := parseRSSItem(item, i+1)
		if repo != nil {
			repos = append(repos, repo)
		}
	}
	return repos, nil
}

func parseRSSItem(item rssItem, rank int) *TrendingRepo {
	match := linkRe.FindStringSubmatch(item.Link)
	if len(match) < 3 {
		return nil
	}
	owner := match[1]
	repoName := match[2]
	desc := cleanDescription(item.Description)
	stars := extractNumber(starsRe, item.Description)
	forks := extractNumber(forksRe, item.Description)
	return &TrendingRepo{
		Rank:        rank,
		FullName:    owner + "/" + repoName,
		Description: desc,
		URL:         item.Link,
		Stars:       stars,
		Forks:       forks,
	}
}

func cleanDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	desc = regexp.MustCompile(`⭐\s*[\d,]+\s*\|\s*🍴\s*[\d,]+\s*\|?\s*`).ReplaceAllString(desc, "")
	return strings.TrimSpace(desc)
}

func extractNumber(re *regexp.Regexp, text string) int {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}
	num := 0
	for _, c := range match[1] {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	return num
}

func (s *Service) trendingViaRSS(ctx context.Context, opts TrendingOpts) ([]*TrendingRepo, error) {
	rssURL := s.rssURL
	if rssURL == "" {
		rssURL = rssURLMap[opts.Since]
		if rssURL == "" {
			rssURL = rssURLMap["weekly"]
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rssURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create RSS request: %w", err)
	}
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch RSS: %w (try --source search as fallback)", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("RSS fetch failed: %d (try --source search as fallback)", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read RSS body: %w", err)
	}
	repos, err := parseRSS(data)
	if err != nil {
		return nil, err
	}
	if opts.Top > 0 && opts.Top < len(repos) {
		repos = repos[:opts.Top]
	}
	if s.gh != nil {
		if err := s.enrichRepos(ctx, repos); err != nil {
			return nil, err
		}
	}
	if opts.Lang != "" {
		repos = filterByLang(repos, opts.Lang)
	}
	for i, r := range repos {
		r.Rank = i + 1
	}
	return repos, nil
}

func (s *Service) enrichRepos(ctx context.Context, repos []*TrendingRepo) error {
	for _, r := range repos {
		parts := strings.SplitN(r.FullName, "/", 2)
		if len(parts) != 2 {
			continue
		}
		details, err := s.gh.GetRepository(ctx, parts[0], parts[1])
		if err != nil {
			continue
		}
		r.Language = details.Language
		r.Topics = details.Topics
		if details.StargazersCount > 0 {
			r.Stars = details.StargazersCount
		}
		if details.ForksCount > 0 {
			r.Forks = details.ForksCount
		}
		if details.Description != "" {
			r.Description = details.Description
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(80 * time.Millisecond):
		}
	}
	return nil
}

func filterByLang(repos []*TrendingRepo, lang string) []*TrendingRepo {
	var filtered []*TrendingRepo
	for _, r := range repos {
		if strings.EqualFold(r.Language, lang) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (s *Service) trendingViaSearch(ctx context.Context, opts TrendingOpts) ([]*TrendingRepo, error) {
	days := 7
	switch opts.Since {
	case "daily":
		days = 7
	case "weekly":
		days = 30
	case "monthly":
		days = 90
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	query := fmt.Sprintf("stars:>1000 created:>%s", since)
	ghRepos, err := s.gh.SearchRepositories(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search repositories: %w", err)
	}
	top := opts.Top
	if top <= 0 {
		top = 20
	}
	limit := top
	if limit > len(ghRepos) {
		limit = len(ghRepos)
	}
	var repos []*TrendingRepo
	for i := 0; i < limit; i++ {
		r := ghRepos[i]
		repos = append(repos, &TrendingRepo{
			Rank:        i + 1,
			FullName:    r.FullName,
			Description: r.Description,
			URL:         r.URL,
			Stars:       r.StargazersCount,
			Forks:       r.ForksCount,
			Language:    r.Language,
			Topics:      r.Topics,
		})
	}
	if opts.Lang != "" {
		repos = filterByLang(repos, opts.Lang)
		for i, r := range repos {
			r.Rank = i + 1
		}
	}
	return repos, nil
}
