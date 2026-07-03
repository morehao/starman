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
