package store

var LocalRepoFields = []string{
	"ai_summary",
	"ai_tags",
	"ai_platforms",
	"ai_search_text",
	"ai_category",
	"analyzed_at",
	"analysis_failed",
	"custom_description",
	"custom_tags",
	"custom_category",
	"category_locked",
	"last_released",
	"subscribed_releases",
	"last_release_fetch",
}

var GitHubSourceFields = []string{
	"name",
	"description",
	"url",
	"language",
	"homepage",
	"stargazers_count",
	"forks_count",
	"topics",
	"owner_login",
	"owner_avatar",
	"starred_at",
	"repo_updated_at",
}

func MergeReposOnSync(incoming []*Repository, existing map[string]*Repository) []*Repository {
	merged := make([]*Repository, 0, len(incoming))

	for _, newRepo := range incoming {
		exist, ok := existing[newRepo.FullName]
		if !ok {
			merged = append(merged, newRepo)
			continue
		}

		result := &Repository{}
		*result = *exist

		result.Name = newRepo.Name
		result.Description = newRepo.Description
		result.URL = newRepo.URL
		result.Language = newRepo.Language
		result.Homepage = newRepo.Homepage
		result.StargazersCount = newRepo.StargazersCount
		result.ForksCount = newRepo.ForksCount
		result.Topics = newRepo.Topics
		result.OwnerLogin = newRepo.OwnerLogin
		result.OwnerAvatar = newRepo.OwnerAvatar
		result.StarredAt = newRepo.StarredAt
		result.RepoUpdatedAt = newRepo.RepoUpdatedAt

		merged = append(merged, result)
	}

	return merged
}
