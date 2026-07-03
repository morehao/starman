package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/app"
	"github.com/morehao/starman/internal/config"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var jsonOut bool
	var limit int
	var lang string
	var category string
	var sortBy string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "AI-powered three-tier degraded search (vector → LLM semantic → text)",
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

			searchIndex := ai.NewSearchIndex()
			svc.SetSearchIndex(searchIndex)

			action := app.NewSearchAction(s, svc)
			res, err := action.Run(ctx, args[0], app.SearchOpts{Lang: lang, Category: category, Sort: sortBy, Limit: limit})
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "%s\n", res.Summary)

			if jsonOut {
				return outputSearchJSON(res.Hits)
			}
			outputSearchTable(res.Hits)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	cmd.Flags().IntVar(&limit, "limit", 0, "limit number of results (0=no limit)")
	cmd.Flags().StringVar(&lang, "lang", "", "filter by language")
	cmd.Flags().StringVar(&category, "category", "", "filter by category")
	cmd.Flags().StringVar(&sortBy, "sort", "score", "sort by: score|stars|name")
	return cmd
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
