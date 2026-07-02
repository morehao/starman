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
	var platform string
	var tags []string
	var minStars int
	var maxStars int
	var analyzed, noAnalyzed, analysisFailed bool
	var hyde, noHyde, noRerank bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "AI-powered three-tier degraded search (vector → LLM semantic → FTS5)",
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
			defer s.Close()
			ctx := context.Background()

			aiKey := config.ResolveAIKey(cfg, "")
			embeddingKey := config.ResolveEmbeddingKey(cfg, "")

			var aiClient *ai.Client
			if aiKey != "" {
				aiClient = ai.NewClient(cfg.AI.BaseURL, aiKey, cfg.AI.Model)
			}

			embeddingClient := ai.NewEmbeddingClient(cfg.Embedding.BaseURL, embeddingKey, cfg.Embedding.Model)
			svc := ai.NewServiceWithEmbedding(aiClient, nil, embeddingClient)

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

			enableHyDE := !noHyde
			if hyde {
				enableHyDE = true
			}
			enableRerank := !noRerank
			if rerank {
				enableRerank = true
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
				EnableHyDE:     enableHyDE,
				EnableRerank:   enableRerank,
				RerankTopK:     30,
			}

			result, err := svc.Search(ctx, args[0], s, opts)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Search mode: %s (%d results)\n", result.Mode, len(result.Hits))

			cliOpts := searchOpts{Lang: lang, Category: category, Sort: sortBy, Limit: limit, Rerank: rerank}
			hits := filterByCLIOpts(result.Hits, cliOpts)

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
	cmd.Flags().StringVar(&platform, "platform", "", "filter by platform (web/desktop/mobile/cli/library/service)")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "filter by tag (OR logic, can use multiple times)")
	cmd.Flags().IntVar(&minStars, "min-stars", 0, "minimum star count")
	cmd.Flags().IntVar(&maxStars, "max-stars", 0, "maximum star count (0=no limit)")
	cmd.Flags().BoolVar(&analyzed, "analyzed", false, "only show analyzed repos")
	cmd.Flags().BoolVar(&noAnalyzed, "no-analyzed", false, "only show unanalyzed repos")
	cmd.Flags().BoolVar(&analysisFailed, "analysis-failed", false, "only show analysis-failed repos")
	cmd.Flags().BoolVar(&hyde, "hyde", false, "enable HyDE query enhancement")
	cmd.Flags().BoolVar(&noHyde, "no-hyde", false, "disable HyDE query enhancement")
	cmd.Flags().BoolVar(&noRerank, "no-rerank", false, "disable LLM reranking")
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
