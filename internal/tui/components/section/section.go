package section

import (
	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/tui/context"
)

type SectionConfig struct {
	Title   string
	Filters string
}

type RowData interface {
	GetId() string
	GetTitle() string
	GetUrl() string
	GetColumns() []string
}

type Section interface {
	GetId() int
	GetType() string
	GetConfig() SectionConfig
	CurrRow() RowData
	NextRow() RowData
	PrevRow()
	FirstItem()
	LastItem()
	NumRows() int
	CurrRowIndex() int
	View() string
	Update(msg tea.Msg) (Section, tea.Cmd)
	FetchNextPageSectionRows() []tea.Cmd
	ResetFilters()
	ResetRows()
	SetIsLoading(bool)
	GetIsLoading() bool
	GetTotalCount() int
	IsSearchFocused() bool
	UpdateProgramContext(*context.ProgramContext)
	FilterRows(query string)
	SupportsSearch() bool
	SupportsFilter() bool
}
