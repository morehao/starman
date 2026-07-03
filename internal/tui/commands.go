package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/tui/components/footer"
)

type parsedCommand struct {
	Name  string
	Args  []string
	Flags map[string]string
}

func parseCommand(input string) parsedCommand {
	input = strings.TrimSpace(input)
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return parsedCommand{}
	}

	cmd := parsedCommand{Name: parts[0], Flags: make(map[string]string)}
	for i := 1; i < len(parts); i++ {
		arg := parts[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if strings.Contains(key, "=") {
				kv := strings.SplitN(key, "=", 2)
				cmd.Flags[kv[0]] = kv[1]
			} else {
				cmd.Flags[key] = "true"
			}
		} else {
			cmd.Args = append(cmd.Args, arg)
		}
	}
	return cmd
}

func (m *Model) executeCommand(cmd parsedCommand) tea.Cmd {
	switch cmd.Name {
	case "q", "quit":
		return tea.Quit
	case "help":
		m.showHelp = !m.showHelp
		return nil
	case "star":
		if len(cmd.Args) < 1 {
			m.setError("usage: :star owner/repo")
			return nil
		}
		return m.cmdStar(cmd.Args[0])

	case "unstar":
		if len(cmd.Args) < 1 {
			m.setError("usage: :unstar owner/repo")
			return nil
		}
		return m.cmdUnstar(cmd.Args[0])

	case "sync":
		if cmd.Flags["full"] == "true" {
			return m.startFullSync()
		}
		return m.startSync()
	default:
		m.footer.SetTask(&footer.TaskInfo{
			Status:  2,
			Message: "unknown command: " + cmd.Name,
		})
		return clearAfterDelay("cmd-err")
	}
}

func (m *Model) startFullSync() tea.Cmd {
	cfg := m.ctx.Config
	if cfg == nil {
		return nil
	}
	token := config.ResolveToken(cfg, "")
	if token == "" || cfg.GitHub.Username == "" {
		m.footer.SetTask(&footer.TaskInfo{
			Status:  2,
			Message: "sync requires github token and username",
		})
		return clearAfterDelay("sync-err")
	}

	taskID := "sync-full-" + time.Now().Format("150405")
	m.tasks.start(taskID, "sync --full")

	return tea.Batch(
		func() tea.Msg { return TaskStartedMsg{TaskID: taskID, Name: "sync --full"} },
		func() tea.Msg {
			gh := github.New(token)
			repos, err := gh.ListStarred(context.Background(), cfg.GitHub.Username)
			if err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "sync --full", Message: "sync failed", Err: err}
			}
			if err := m.ctx.Store.UpsertReposOnSync(context.Background(), repos, true); err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "sync --full", Message: "sync failed", Err: err}
			}
			return TaskFinishedMsg{
				TaskID:  taskID,
				Name:    "sync --full",
				Message: fmt.Sprintf("synced %d repos (full)", len(repos)),
			}
		},
	)
}

func (m *Model) cmdStar(fullName string) tea.Cmd {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		m.setError("invalid repo: " + fullName)
		return nil
	}
	cfg := m.ctx.Config
	token := config.ResolveToken(cfg, "")
	if token == "" {
		m.setError("GitHub token required")
		return nil
	}
	taskID := "star-cmd-" + time.Now().Format("150405")
	m.tasks.start(taskID, "star "+fullName)
	return tea.Batch(
		func() tea.Msg { return TaskStartedMsg{TaskID: taskID, Name: "star " + fullName} },
		func() tea.Msg {
			gh := github.New(token)
			if err := gh.Star(context.Background(), parts[0], parts[1]); err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "star", Message: "star failed", Err: err}
			}
			return TaskFinishedMsg{TaskID: taskID, Name: "star", Message: "starred " + fullName}
		},
	)
}

func (m *Model) cmdUnstar(fullName string) tea.Cmd {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		m.setError("invalid repo: " + fullName)
		return nil
	}
	cfg := m.ctx.Config
	token := config.ResolveToken(cfg, "")
	if token == "" {
		m.setError("GitHub token required")
		return nil
	}
	taskID := "unstar-cmd-" + time.Now().Format("150405")
	m.tasks.start(taskID, "unstar "+fullName)
	return tea.Batch(
		func() tea.Msg { return TaskStartedMsg{TaskID: taskID, Name: "unstar " + fullName} },
		func() tea.Msg {
			gh := github.New(token)
			ctx := context.Background()
			if err := gh.Unstar(ctx, parts[0], parts[1]); err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "unstar", Message: "unstar failed", Err: err}
			}
			existing, err := m.ctx.Store.GetRepository(ctx, fullName)
			if err == nil {
				existing.StarredAt = ""
				_ = m.ctx.Store.UpsertRepository(ctx, existing)
			}
			return TaskFinishedMsg{TaskID: taskID, Name: "unstar", Message: "unstarred " + fullName}
		},
	)
}
