package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestExpandPath_AbsolutePath(t *testing.T) {
	path := "/absolute/path/to/project"
	got := expandPath(path)
	if got != path {
		t.Errorf("expandPath(%q) = %q, want unchanged", path, got)
	}
}

func TestExpandPath_TildeExpands(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	got := expandPath("~/myproject")
	want := filepath.Join(home, "myproject")
	if got != want {
		t.Errorf("expandPath(~/myproject) = %q, want %q", got, want)
	}
}

func TestExpandPath_NoTilde(t *testing.T) {
	got := expandPath("/no/tilde/here")
	if got != "/no/tilde/here" {
		t.Errorf("expandPath without tilde changed: %q", got)
	}
}

func TestListDirs_EmptyInput(t *testing.T) {
	if got := listDirs(""); got != nil {
		t.Errorf("listDirs('') = %v, want nil", got)
	}
}

func TestListDirs_NoSeparator(t *testing.T) {
	if got := listDirs("noseparator"); got != nil {
		t.Errorf("listDirs('noseparator') = %v, want nil", got)
	}
}

func TestListDirs_ReturnsSubdirs(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"aaa", "bbb"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := listDirs(dir + string(filepath.Separator))
	if len(got) != 2 {
		t.Errorf("listDirs returned %d entries, want 2: %v", len(got), got)
	}
}

func TestListDirs_IgnoresHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "visible"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := listDirs(dir + string(filepath.Separator))
	if len(got) != 1 {
		t.Errorf("listDirs returned %d entries, want 1 (hidden excluded): %v", len(got), got)
	}
	if filepath.Base(got[0]) != "visible" {
		t.Errorf("expected 'visible', got %q", filepath.Base(got[0]))
	}
}

func TestListDirs_PrefixFilter(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"abc", "abd", "xyz"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got := listDirs(filepath.Join(dir, "ab"))
	if len(got) != 2 {
		t.Errorf("listDirs with prefix 'ab' = %d results, want 2: %v", len(got), got)
	}
}

func TestListDirs_NonExistentDir(t *testing.T) {
	got := listDirs("/this/path/does/not/exist/")
	if got != nil {
		t.Errorf("listDirs for non-existent dir = %v, want nil", got)
	}
}

func folderKey(m FolderModel, key string) FolderModel {
	var msg tea.KeyMsg
	switch key {
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	updated, _ := m.Update(msg)
	return updated
}

func TestFolderUpdate_Menu_CursorDown(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_MENU, cursor: 0}
	m = folderKey(m, "down")
	if m.cursor != 1 {
		t.Errorf("after down cursor = %d, want 1", m.cursor)
	}
}

func TestFolderUpdate_Menu_CursorDownClamped(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_MENU, cursor: 1}
	m = folderKey(m, "down")
	if m.cursor != 1 {
		t.Errorf("cursor should stay at 1 (bottom), got %d", m.cursor)
	}
}

func TestFolderUpdate_Menu_CursorUp(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_MENU, cursor: 1}
	m = folderKey(m, "up")
	if m.cursor != 0 {
		t.Errorf("after up cursor = %d, want 0", m.cursor)
	}
}

func TestFolderUpdate_Menu_CursorUpClamped(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_MENU, cursor: 0}
	m = folderKey(m, "up")
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0 (top), got %d", m.cursor)
	}
}

func TestFolderUpdate_Menu_SelectCurrentDir(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_MENU, cursor: MENU_ITEM_CURRENT, workDir: "/some/dir"}
	m = folderKey(m, "enter")
	if !m.done {
		t.Error("done should be true after selecting current dir")
	}
	if m.selected != "/some/dir" {
		t.Errorf("selected = %q, want /some/dir", m.selected)
	}
}

func TestFolderUpdate_Menu_SelectCustomEntersInputMode(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_MENU, cursor: MENU_ITEM_CUSTOM, workDir: "/some/dir"}
	m = folderKey(m, "enter")
	if m.mode != FOLDER_MODE_INPUT {
		t.Errorf("mode = %d, want FOLDER_MODE_INPUT (%d)", m.mode, FOLDER_MODE_INPUT)
	}
	if m.done {
		t.Error("done should not be set when entering input mode")
	}
}

func TestFolderUpdate_Menu_EscSetsBack(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_MENU}
	m = folderKey(m, "esc")
	if !m.backPressed {
		t.Error("backPressed should be true after esc in menu mode")
	}
}

func TestFolderUpdate_Input_TypeCharacter(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_INPUT, input: "/foo/", inputCursor: 5}
	m = folderKey(m, "b")
	if !strings.Contains(m.input, "b") {
		t.Errorf("input %q should contain typed character 'b'", m.input)
	}
	if m.inputCursor != 6 {
		t.Errorf("inputCursor = %d after typing, want 6", m.inputCursor)
	}
}

func TestFolderUpdate_Input_Backspace(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_INPUT, input: "/foo/b", inputCursor: 6}
	m = folderKey(m, "backspace")
	if m.input != "/foo/" {
		t.Errorf("after backspace input = %q, want '/foo/'", m.input)
	}
	if m.inputCursor != 5 {
		t.Errorf("after backspace cursor = %d, want 5", m.inputCursor)
	}
}

func TestFolderUpdate_Input_BackspaceAtStart(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_INPUT, input: "/foo/", inputCursor: 0}
	m = folderKey(m, "backspace")
	if m.input != "/foo/" {
		t.Errorf("backspace at pos 0 should not change input, got %q", m.input)
	}
}

func TestFolderUpdate_Input_EnterConfirmsValidPath(t *testing.T) {
	m := FolderModel{
		mode:        FOLDER_MODE_INPUT,
		input:       "/home/user/project",
		inputCursor: len([]rune("/home/user/project")),
	}
	m = folderKey(m, "enter")
	if !m.done {
		t.Error("done should be true for valid path with separator")
	}
}

func TestFolderUpdate_Input_EnterBlockedByDanger(t *testing.T) {
	m := FolderModel{
		mode:      FOLDER_MODE_INPUT,
		input:     "/",
		dangerMsg: "Nope.",
	}
	m = folderKey(m, "enter")
	if m.done {
		t.Error("done must not be set when dangerMsg is present")
	}
}

func TestFolderUpdate_Input_EnterBlockedWithoutSeparator(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_INPUT, input: "noseparator"}
	m = folderKey(m, "enter")
	if m.done {
		t.Error("done must not be set for input without separator")
	}
}

func TestFolderUpdate_Input_EnterBlockedEmpty(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_INPUT, input: ""}
	m = folderKey(m, "enter")
	if m.done {
		t.Error("done must not be set for empty input")
	}
}

func TestFolderUpdate_Input_EscBackToMenu(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_INPUT, directInput: false}
	m = folderKey(m, "esc")
	if m.mode != FOLDER_MODE_MENU {
		t.Errorf("mode = %d after esc, want FOLDER_MODE_MENU", m.mode)
	}
	if m.input != "" {
		t.Errorf("input should be cleared after esc, got %q", m.input)
	}
}

func TestFolderUpdate_Input_EscDirectInputSetsBack(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_INPUT, directInput: true}
	m = folderKey(m, "esc")
	if !m.backPressed {
		t.Error("backPressed should be true when directInput=true and esc pressed")
	}
}

func TestFolderUpdate_Input_LeftRightMoveCursor(t *testing.T) {
	m := FolderModel{mode: FOLDER_MODE_INPUT, input: "/foo/bar", inputCursor: 5}
	m = folderKey(m, "left")
	if m.inputCursor != 4 {
		t.Errorf("after left cursor = %d, want 4", m.inputCursor)
	}
	m = folderKey(m, "right")
	if m.inputCursor != 5 {
		t.Errorf("after right cursor = %d, want 5", m.inputCursor)
	}
}
