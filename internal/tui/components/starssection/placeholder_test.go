package starssection

import (
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/morehao/starman/internal/store"
)

// TestPlaceholderStableWidth guards against re-introducing East-Asian
// Ambiguous-width chars (e.g. em dash U+2014) as placeholders for empty
// lang/cat fields. Such chars have locale-dependent runewidth (1 in en_US/C,
// 2 in zh_CN/ja_JP) while the terminal's actual rendering frequently
// disagrees with runewidth, which breaks column alignment for any row whose
// lang/cat is empty. See the bug: em dash caused cat-column misalignment
// when Language == "".
func TestPlaceholderStableWidth(t *testing.T) {
	row := RepoRow{Repo: &store.Repository{FullName: "owner/repo"}}
	cols := row.GetColumns()

	for name, ph := range map[string]string{"lang": cols[2], "cat": cols[3]} {
		r := []rune(ph)[0]
		if runewidth.IsAmbiguousWidth(r) {
			t.Errorf("%s placeholder %q (U+%04X) is East-Asian Ambiguous width; "+
				"runewidth is locale-dependent and breaks column alignment "+
				"on some terminals. Use an ASCII char instead.", name, ph, r)
		}
		if w := runewidth.StringWidth(ph); w != 1 {
			t.Errorf("%s placeholder %q width=%d want 1", name, ph, w)
		}
	}
}
