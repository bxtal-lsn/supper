package components

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bxtal-lsn/supper/internal/sops"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileSelectedMsg is sent when a file is selected
type FileSelectedMsg struct {
	Path string
	Info *sops.FileInfo
}

// DirectoryChangedMsg is sent when the directory changes
type DirectoryChangedMsg struct {
	Path string
}

// FileItem represents a file or directory in the file browser
type FileItem struct {
	Path     string
	Name     string
	IsDir    bool
	IsSOPS   bool
	Size     int64
	ModTime  string
	FileInfo *sops.FileInfo
}

// FilterValue implements list.Item
func (i FileItem) FilterValue() string {
	return i.Name
}

// fileBrowserKeyMap defines the keybindings for the file browser
type fileBrowserKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	GoBack   key.Binding
	GoHome   key.Binding
	GoParent key.Binding
}

// newFileBrowserKeyMap returns the default file browser keybindings
func newFileBrowserKeyMap() fileBrowserKeyMap {
	return fileBrowserKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select/open"),
		),
		GoBack: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "go back"),
		),
		GoHome: key.NewBinding(
			key.WithKeys("~"),
			key.WithHelp("~", "go to home"),
		),
		GoParent: key.NewBinding(
			key.WithKeys(".."),
			key.WithHelp("..", "go to parent"),
		),
	}
}

// FileBrowser is a component for browsing files
type FileBrowser struct {
	list       list.Model
	keys       fileBrowserKeyMap
	currentDir string
	history    []string
	width      int
	height     int
}

// NewFileBrowser creates a new file browser
func NewFileBrowser() *FileBrowser {
	keys := newFileBrowserKeyMap()

	// Get initial directory (current working directory)
	currentDir, err := os.Getwd()
	if err != nil {
		currentDir = "."
	}

	// Create delegate for custom list item rendering
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#1E88E5"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#DDDDDD")).Background(lipgloss.Color("#1E88E5"))

	// Create list model
	listModel := list.New([]list.Item{}, delegate, 0, 0)
	listModel.Title = "File Browser"
	listModel.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#333333")).Padding(0, 1)

	fb := &FileBrowser{
		list:       listModel,
		keys:       keys,
		currentDir: currentDir,
		history:    []string{},
	}

	return fb
}

// Init initializes the component
func (f *FileBrowser) Init() tea.Cmd {
	return f.loadDirectory(f.currentDir)
}

// Update handles events and updates the model
func (f *FileBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
		f.list.SetSize(msg.Width, msg.Height-5)

	case tea.KeyMsg:
		// Handle custom key bindings
		switch {
		case key.Matches(msg, f.keys.GoBack) && len(f.history) > 0:
			// Go back in history
			prev := f.history[len(f.history)-1]
			f.history = f.history[:len(f.history)-1]
			return f, f.loadDirectory(prev)

		case key.Matches(msg, f.keys.GoHome):
			// Go to home directory
			home, err := os.UserHomeDir()
			if err == nil {
				f.history = append(f.history, f.currentDir)
				return f, f.loadDirectory(home)
			}

		case key.Matches(msg, f.keys.GoParent):
			// Go to parent directory
			parent := filepath.Dir(f.currentDir)
			if parent != f.currentDir {
				f.history = append(f.history, f.currentDir)
				return f, f.loadDirectory(parent)
			}

		case key.Matches(msg, f.keys.Enter):
			// Handle selection
			if i, ok := f.list.SelectedItem().(FileItem); ok {
				if i.IsDir {
					// If directory, navigate into it
					f.history = append(f.history, f.currentDir)
					return f, f.loadDirectory(i.Path)
				} else {
					// If file, return selection message
					return f, func() tea.Msg {
						return FileSelectedMsg{
							Path: i.Path,
							Info: i.FileInfo,
						}
					}
				}
			}
		}
	case DirectoryChangedMsg:
		// Handle directory changed externally
		f.history = append(f.history, f.currentDir)
		return f, f.loadDirectory(msg.Path)
	}

	// Update list model
	f.list, cmd = f.list.Update(msg)
	cmds = append(cmds, cmd)

	return f, tea.Batch(cmds...)
}

// View renders the component
func (f *FileBrowser) View() string {
	// Create breadcrumb
	breadcrumb := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Render(fmt.Sprintf(" %s ", f.currentDir))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		breadcrumb,
		f.list.View(),
	)
}

// loadDirectory loads the contents of a directory
func (f *FileBrowser) loadDirectory(dir string) tea.Cmd {
	return func() tea.Msg {
		// Read directory contents
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}

		// Update current directory
		f.currentDir = dir

		// Convert entries to list items
		items := make([]list.Item, 0, len(entries))

		// Add parent directory entry if not at root
		if dir != "/" {
			parentPath := filepath.Dir(dir)
			items = append(items, FileItem{
				Path:  parentPath,
				Name:  "..",
				IsDir: true,
			})
		}

		// Sort directories first, then alphabetically
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
		})

		// Add each entry
		for _, entry := range entries {
			// Skip hidden files
			if strings.HasPrefix(entry.Name(), ".") && entry.Name() != ".." {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Check if it's a SOPS-encrypted file
			var fileInfo *sops.FileInfo
			if !entry.IsDir() {
				fileInfo, _ = sops.GetFileInfo(path)
			}

			items = append(items, FileItem{
				Path:     path,
				Name:     entry.Name(),
				IsDir:    entry.IsDir(),
				IsSOPS:   fileInfo != nil && fileInfo.Encrypted,
				Size:     info.Size(),
				ModTime:  info.ModTime().Format("2006-01-02 15:04:05"),
				FileInfo: fileInfo,
			})
		}

		// Update list with new items
		f.list.SetItems(items)

		return DirectoryChangedMsg{Path: dir}
	}
}

// SetSize sets the size of the component
func (f *FileBrowser) SetSize(width, height int) {
	f.width = width
	f.height = height
	f.list.SetSize(width, height-5)
}

// SetDirectory changes the current directory
func (f *FileBrowser) SetDirectory(dir string) tea.Cmd {
	return f.loadDirectory(dir)
}

// ShortHelp returns keybindings to be shown in the mini help view
func (f *FileBrowser) ShortHelp() []key.Binding {
	return []key.Binding{
		f.keys.Up,
		f.keys.Down,
		f.keys.Enter,
		f.keys.GoBack,
		f.keys.GoHome,
	}
}

// FullHelp returns keybindings for the expanded help view
func (f *FileBrowser) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{f.keys.Up, f.keys.Down},
		{f.keys.Enter, f.keys.GoBack, f.keys.GoHome, f.keys.GoParent},
	}
}

