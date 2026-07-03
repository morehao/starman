package pages

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/backup"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

var backupOps = []string{
	"Export JSON",
	"Import JSON",
	"Push to WebDAV",
	"Pull from WebDAV",
}

type BackupModel struct {
	store         store.Store
	theme         *styles.Theme
	cfg           *config.Config
	enqueue       func(string) string
	cursor        int
	status        string
	width         int
	height        int
	confirmed     bool
	confirmPrompt string
	running       bool
	data          []byte
}

type backupDoneMsg struct {
	id     string
	status string
	err    error
}

func NewBackup(s store.Store, theme *styles.Theme, cfg *config.Config, enqueue func(string) string) *BackupModel {
	return &BackupModel{store: s, theme: theme, cfg: cfg, enqueue: enqueue}
}

func (m *BackupModel) Init() tea.Cmd { return nil }

func (m *BackupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
			m.confirmed = false
			m.confirmPrompt = ""
		case "down", "j":
			m.cursor = min(m.cursor+1, len(backupOps)-1)
			m.confirmed = false
			m.confirmPrompt = ""
		case "enter":
			if m.running {
				return m, nil
			}
			switch m.cursor {
			case 1:
				if !m.confirmed {
					m.confirmed = true
					m.confirmPrompt = "Import may overwrite local data. Press Enter again to confirm."
					return m, nil
				}
				m.confirmed = false
				m.confirmPrompt = ""
				return m, m.runImportCmd()
			case 0:
				return m, m.runExportCmd()
			case 2:
				return m, m.runWebDAVPushCmd()
			case 3:
				return m, m.runWebDAVPullCmd()
			}
		case "esc":
			m.confirmed = false
			m.confirmPrompt = ""
		}
	case backupDoneMsg:
		m.running = false
		m.status = msg.status
		if msg.id != "" {
			return m, func() tea.Msg { return types.TaskDoneMsg{ID: msg.id, Err: msg.err} }
		}
	}
	return m, nil
}

func (m *BackupModel) backupPath() string {
	dir, err := config.DefaultDir()
	if err != nil {
		return "starman-backup.json"
	}
	return filepath.Join(dir, "starman-backup.json")
}

func (m *BackupModel) runExportCmd() tea.Cmd {
	m.running = true
	id := m.enqueue("export")
	return tea.Batch(
		func() tea.Msg { return types.TaskStartedMsg{ID: id, Label: "Export JSON"} },
		func() tea.Msg {
			ctx := context.Background()
			data, err := backup.ExportJSON(ctx, m.store)
			if err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("Export failed: %v", err), err: err}
			}
			path := m.backupPath()
			if err := os.WriteFile(path, data, 0644); err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("Write failed: %v", err), err: err}
			}
			m.data = data
			return backupDoneMsg{id: id, status: fmt.Sprintf("Exported to %s (%d bytes)", path, len(data))}
		},
	)
}

func (m *BackupModel) runImportCmd() tea.Cmd {
	m.running = true
	id := m.enqueue("import")
	return tea.Batch(
		func() tea.Msg { return types.TaskStartedMsg{ID: id, Label: "Import JSON"} },
		func() tea.Msg {
			ctx := context.Background()
			data, err := os.ReadFile(m.backupPath())
			if err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("Read failed: %v", err), err: err}
			}
			if err := backup.ImportJSON(ctx, m.store, data, backup.ImportReplace); err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("Import failed: %v", err), err: err}
			}
			return backupDoneMsg{id: id, status: "Import completed successfully"}
		},
	)
}

func (m *BackupModel) runWebDAVPushCmd() tea.Cmd {
	m.running = true
	id := m.enqueue("webdav-push")
	return tea.Batch(
		func() tea.Msg { return types.TaskStartedMsg{ID: id, Label: "WebDAV Push"} },
		func() tea.Msg {
			ctx := context.Background()
			data, err := backup.ExportJSON(ctx, m.store)
			if err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("Export failed: %v", err), err: err}
			}
			if m.cfg == nil || m.cfg.WebDAV.URL == "" {
				return backupDoneMsg{id: id, status: "WebDAV not configured"}
			}
			pass := config.ResolveWebDAVPassword(m.cfg)
			client := backup.NewWebDAVClient(m.cfg.WebDAV.URL, m.cfg.WebDAV.Username, pass)
			path := m.cfg.WebDAV.Path
			if path == "" {
				path = "/starman-backup.json"
			}
			if err := client.Push(ctx, path, data); err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("WebDAV push failed: %v", err), err: err}
			}
			return backupDoneMsg{id: id, status: fmt.Sprintf("Pushed to WebDAV %s (%d bytes)", path, len(data))}
		},
	)
}

func (m *BackupModel) runWebDAVPullCmd() tea.Cmd {
	m.running = true
	id := m.enqueue("webdav-pull")
	return tea.Batch(
		func() tea.Msg { return types.TaskStartedMsg{ID: id, Label: "WebDAV Pull"} },
		func() tea.Msg {
			ctx := context.Background()
			if m.cfg == nil || m.cfg.WebDAV.URL == "" {
				return backupDoneMsg{id: id, status: "WebDAV not configured"}
			}
			pass := config.ResolveWebDAVPassword(m.cfg)
			client := backup.NewWebDAVClient(m.cfg.WebDAV.URL, m.cfg.WebDAV.Username, pass)
			path := m.cfg.WebDAV.Path
			if path == "" {
				path = "/starman-backup.json"
			}
			data, err := client.Pull(ctx, path)
			if err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("WebDAV pull failed: %v", err), err: err}
			}
			if err := backup.ImportJSON(ctx, m.store, data, backup.ImportReplace); err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("Import failed: %v", err), err: err}
			}
			if err := os.WriteFile(m.backupPath(), data, 0644); err != nil {
				return backupDoneMsg{id: id, status: fmt.Sprintf("Write local backup failed: %v", err), err: err}
			}
			return backupDoneMsg{id: id, status: fmt.Sprintf("Pulled from WebDAV %s (%d bytes)", path, len(data))}
		},
	)
}

func (m *BackupModel) View() string {
	title := m.theme.PageTitle.Render("Backup") + "\n\n"
	var body string
	for i, op := range backupOps {
		line := fmt.Sprintf("  %s", op)
		if i == m.cursor {
			body += m.theme.SidebarActive.Render(line) + "\n"
		} else {
			body += line + "\n"
		}
	}
	if m.confirmPrompt != "" {
		body += "\n" + m.confirmPrompt
	} else if m.status != "" {
		body += "\n" + m.status
	}
	return title + body
}
