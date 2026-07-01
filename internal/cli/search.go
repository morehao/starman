package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/config"
	"github.com/spf13/cobra"
)

type searchOpts struct {
	Lang     string
	Category string
	Sort     string
	Limit    int
	Rerank   bool
}

func newSearchCmd() *cobra.Command {
	var jsonOut bool
	var limit int
	var lang string
	var category string
	var sortBy string
	var rerank bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "AI-powered semantic search with FTS5 full-text index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			aiKey := config.ResolveAIKey(cfg, "")
			if aiKey == "" {
				return fmt.Errorf("AI api_key required")
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			aiClient := ai.NewClient(cfg.AI.BaseURL, aiKey, cfg.AI.Model)
			svc := ai.NewService(aiClient, nil)

			opts := ai.SearchOpts{
				Language: lang,
				Category: category,
				Rerank:   rerank,
				Sort:     sortBy,
				Limit:    limit,
			}

			hits, err := svc.Search(ctx, args[0], s, opts)
			if err != nil {
				return err
			}

			cliOpts := searchOpts{Lang: lang, Category: category, Sort: sortBy, Limit: limit, Rerank: rerank}
			hits = filterByCLIOpts(hits, cliOpts)

			if jsonOut {
				return outputSearchJSON(hits)
			}
			outputSearchTable(hits)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	cmd.Flags().IntVar(&limit, "limit", 0, "limit number of results (0=no limit)")
	cmd.Flags().StringVar(&lang, "lang", "", "filter by language")
	cmd.Flags().StringVar(&category, "category", "", "filter by category")
	cmd.Flags().StringVar(&sortBy, "sort", "score", "sort by: score|stars|name")
	cmd.Flags().BoolVar(&rerank, "rerank", false, "use LLM to rerank top candidates")
	return cmd
}

func filterByCLIOpts(hits []*ai.SearchHit, opts searchOpts) []*ai.SearchHit {
	var filtered []*ai.SearchHit
	for _, h := range hits {
		if opts.Lang != "" && !strings.EqualFold(h.Repo.Language, opts.Lang) {
			continue
		}
		if opts.Category != "" {
			cat := h.Repo.CustomCategory
			if cat == "" {
				cat = h.Repo.AICategory
			}
			if !strings.EqualFold(cat, opts.Category) {
				continue
			}
		}
		filtered = append(filtered, h)
	}
	switch opts.Sort {
	case "stars":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Repo.StargazersCount > filtered[j].Repo.StargazersCount
		})
	case "name":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Repo.FullName < filtered[j].Repo.FullName
		})
	default:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Score > filtered[j].Score
		})
	}
	if opts.Limit > 0 && opts.Limit < len(filtered) {
		filtered = filtered[:opts.Limit]
	}
	return filtered
}

func outputSearchTable(hits []*ai.SearchHit) {
	fmt.Fprintf(os.Stdout, "%-6s %-40s %s\n", "SCORE", "REPO", "DESCRIPTION")
	for _, h := range hits {
		desc := h.Repo.Description
		if len(desc) > 50 {
			desc = desc[:50] + "..."
		}
		fmt.Fprintf(os.Stdout, "%-6.1f %-40s %s\n", h.Score, h.Repo.FullName, desc)
	}
}

func outputSearchJSON(hits []*ai.SearchHit) error {
	type jsonHit struct {
		Score    float64 `json:"score"`
		FullName string  `json:"full_name"`
		Language string  `json:"language"`
		Stars    int     `json:"stars"`
		Category string  `json:"category"`
		Summary  string  `json:"summary"`
	}
	var out []jsonHit
	for _, h := range hits {
		cat := h.Repo.CustomCategory
		if cat == "" {
			cat = h.Repo.AICategory
		}
		out = append(out, jsonHit{
			Score:    h.Score,
			FullName: h.Repo.FullName,
			Language: h.Repo.Language,
			Stars:    h.Repo.StargazersCount,
			Category: cat,
			Summary:  h.Repo.AISummary,
		})
	}
	if out == nil {
		out = []jsonHit{}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
