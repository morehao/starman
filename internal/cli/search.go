package cli

import (
	"context"
	"encoding/json"
	"fmt"
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
}

func newSearchCmd() *cobra.Command {
	var jsonOut bool
	var limit int
	var lang string
	var category string
	var sortBy string
	var platform string
	var tags []string
	var minStars int
	var maxStars int
	var analyzed, noAnalyzed, analysisFailed, noVector bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "AI-powered hybrid search (vector + keyword) with three-tier fallback",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()
			ctx := context.Background()

			aiKey := config.ResolveAIKey(cfg, "")
			embeddingKey := config.ResolveEmbeddingKey(cfg, "")

			var aiClient *ai.Client
			if aiKey != "" {
				aiClient = ai.NewClient(cfg.AI.BaseURL, aiKey, cfg.AI.Model)
			}

			var embeddingClient *ai.EmbeddingClient
			if !noVector {
				embeddingClient = ai.NewEmbeddingClient(cfg.Embedding.BaseURL, embeddingKey, cfg.Embedding.Model)
			}
			svc := ai.NewServiceWithEmbedding(aiClient, nil, embeddingClient)

			searchIndex := ai.NewSearchIndex()
			svc.SetSearchIndex(searchIndex)

			var analyzedPtr *bool
			if analyzed || noAnalyzed {
				v := analyzed
				analyzedPtr = &v
			}
			var analysisFailedPtr *bool
			if analysisFailed {
				v := true
				analysisFailedPtr = &v
			}

			opts := ai.SearchOpts{
				Language:       lang,
				Category:       category,
				Platform:       platform,
				Tags:           tags,
				MinStars:       minStars,
				MaxStars:       maxStars,
				Sort:           sortBy,
				Limit:          limit,
				Analyzed:       analyzedPtr,
				AnalysisFailed: analysisFailedPtr,
			}

			result, err := svc.Search(ctx, args[0], s, opts)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Search mode: %s (%d results)\n", result.Mode, len(result.Hits))

			cliOpts := searchOpts{Lang: lang, Category: category, Sort: sortBy, Limit: limit}
			hits := filterByCLIOpts(result.Hits, cliOpts)

			if jsonOut {
				return outputSearchJSON(cmd, hits)
			}
			outputSearchTable(cmd, hits)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	cmd.Flags().IntVar(&limit, "limit", 0, "limit number of results (0=no limit)")
	cmd.Flags().StringVar(&lang, "lang", "", "filter by language")
	cmd.Flags().StringVar(&category, "category", "", "filter by category")
	cmd.Flags().StringVar(&sortBy, "sort", "score", "sort by: score|stars|name")
	cmd.Flags().StringVar(&platform, "platform", "", "filter by platform (web/desktop/mobile/cli/library/service)")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "filter by tag (OR logic, can use multiple times)")
	cmd.Flags().IntVar(&minStars, "min-stars", 0, "minimum star count")
	cmd.Flags().IntVar(&maxStars, "max-stars", 0, "maximum star count (0=no limit)")
	cmd.Flags().BoolVar(&analyzed, "analyzed", false, "only show analyzed repos")
	cmd.Flags().BoolVar(&noAnalyzed, "no-analyzed", false, "only show unanalyzed repos")
	cmd.Flags().BoolVar(&analysisFailed, "analysis-failed", false, "only show analysis-failed repos")
	cmd.Flags().BoolVar(&noVector, "no-vector", false, "disable vector search, use text search only")
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

func outputSearchTable(cmd *cobra.Command, hits []*ai.SearchHit) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-6s %-40s %s\n", "SCORE", "REPO", "DESCRIPTION")
	for _, h := range hits {
		desc := h.Repo.Description
		if len(desc) > 50 {
			desc = desc[:50] + "..."
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-6.1f %-40s %s\n", h.Score, h.Repo.FullName, desc)
	}
}

func outputSearchJSON(cmd *cobra.Command, hits []*ai.SearchHit) error {
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
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
