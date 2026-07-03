package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

type StatItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func newStatsCmd() *cobra.Command {
	var by string
	var top int
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show statistics of synced repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			repos, err := s.ListRepositories(context.Background())
			if err != nil {
				return err
			}
			items := aggregateStats(repos, by)
			if top > 0 && top < len(items) {
				items = items[:top]
			}
			if jsonOut {
				return outputStatsJSON(items)
			}
			outputStatsTable(items, by)
			return nil
		},
	}
	cmd.Flags().StringVar(&by, "by", "language", "dimension: language|category|tag")
	cmd.Flags().IntVar(&top, "top", 10, "show top N items (0=all)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func aggregateStats(repos []*store.Repository, by string) []StatItem {
	counts := make(map[string]int)
	switch by {
	case "language":
		for _, r := range repos {
			lang := r.Language
			if lang == "" {
				lang = "Others"
			}
			counts[lang]++
		}
	case "category":
		for _, r := range repos {
			cat := r.CustomCategory
			if cat == "" {
				cat = r.AICategory
			}
			if cat == "" {
				cat = "其他"
			}
			counts[cat]++
		}
	case "tag":
		for _, r := range repos {
			seen := make(map[string]bool)
			for _, tag := range r.AITags {
				tag = strings.ToLower(tag)
				if !seen[tag] {
					counts[tag]++
					seen[tag] = true
				}
			}
			seen = make(map[string]bool)
			for _, tag := range r.CustomTags {
				tag = strings.ToLower(tag)
				if !seen[tag] {
					counts[tag]++
					seen[tag] = true
				}
			}
		}
	}
	items := make([]StatItem, 0, len(counts))
	for name, count := range counts {
		items = append(items, StatItem{Name: name, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Name < items[j].Name
	})
	return items
}

func outputStatsTable(items []StatItem, by string) {
	header := strings.ToUpper(by)
	fmt.Fprintf(os.Stdout, "%-30s %s\n", header, "COUNT")
	for _, item := range items {
		fmt.Fprintf(os.Stdout, "%-30s %d\n", item.Name, item.Count)
	}
}

func outputStatsJSON(items []StatItem) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
