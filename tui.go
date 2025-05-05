package main

import (
	conf "blanktiger/hm/configuration"
	"blanktiger/hm/lib"
	"fmt"
	"os"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
)

// IDEAS:
// - I want a list of all configs with information on if they would be copied over
//   with an ability to change that fact by selecting or unselecting them in a list
//
// - I want a list of all packages that would be installed, etc same thing as above,
//   same thing for packages that would be uninstalled
//
// - I want a list of all global dependencies that would be installed
//
// - An editor of global dependencies list (they would be permanently stored on disk
//   after we are done editing)
//
// - An option to permanently hide/unhide a config from being copied (makes it so that
//   I don't have to manually hide them)

type screen int

const (
	configsToInstall screen = iota
)

func isValidScreen(screenId int) bool {
	switch screenId {
	case int(configsToInstall):
		return true

	default:
		return false
	}
}

type model struct {
	lockfile      *lib.Lockfile
	configs       []lib.Config
	selected      map[int]bool
	currentScreen screen
	cursor        int
	marginLeft    int
	marginTop     int
	dbgMsg        string
}

func initModel(lockfile *lib.Lockfile) model {
	selected := make(map[int]bool)
	for idx := range lockfile.Configs {
		selected[idx] = true
	}
	offset := len(lockfile.Configs)
	for idx := range lockfile.HiddenConfigs {
		selected[idx+offset] = false
	}

	return model{
		lockfile: lockfile,
		configs:  slices.Concat(lockfile.Configs, lockfile.HiddenConfigs),
		selected: selected,
	}
}

func (m model) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m model) View() string {
	s := ""
	switch m.currentScreen {
	case configsToInstall:
		s = m.configsToInstallScreen()
	default:
		panic("invalid screen id")
	}
	return borderedWindowStyle.Render(s)
}

const accentColor = "#17d87e"

var borderedWindowStyle = lg.NewStyle().
	Border(lg.RoundedBorder(), true).
	BorderForeground(lg.Color(accentColor))

var titleStyle = lg.NewStyle().
	PaddingLeft(8).
	PaddingRight(8).
	AlignHorizontal(lg.Center)

var footerStyle = lg.NewStyle().
	Inherit(titleStyle).
	Foreground(lg.Color(accentColor))

var selectedStyle = lg.NewStyle().
	Bold(true).
	Foreground(lg.Color(accentColor))

var listEntryStyle = lg.NewStyle()

func (m model) configsToInstallScreen() string {
	titleTxt := "Configs - select the ones you wanna copy/symlink."
	title := titleStyle.Render(titleTxt)
	res := title + "\n\n\n"

	list := ""
	for idx, cfg := range m.configs {
		if isSelected, ok := m.selected[idx]; ok {
			selectionLine := ""
			prefix := "\t\t   "
			if idx == m.cursor {
				prefix = selectedStyle.Render("\t\t-> ")
			}
			if isSelected {
				selectionLine = fmt.Sprintf("[*] %s\n", cfg.Name)
			} else {
				selectionLine = fmt.Sprintf("[ ] %s\n", cfg.Name)
			}
			list += listEntryStyle.Render(prefix + selectionLine)
		}
	}
	res += list

	padding := strings.Repeat(" ", (len(title) - len(titleTxt)))
	res += footerStyle.Render("\n" + padding + "Select by pressing: <space>\n" + padding + "Accept by pressing: <enter>")
	res += m.dbgMsg

	return res
}

func (m *model) nextScreen() {
	newScreenId := int(m.currentScreen) + 1
	if isValidScreen(newScreenId) {
		m.currentScreen = screen(newScreenId)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {

	case tea.WindowSizeMsg:
		windowSize := msg.(tea.WindowSizeMsg)

		// view := m.View()
		// height := strings.Count(view, "\n")
		// width := 0
		// for line := range strings.SplitSeq(view, "\n") {
		// 	width = max(len(line), width)
		// }
		borderedWindowStyle = borderedWindowStyle.
			Width(windowSize.Width).
			Height(windowSize.Height).
			Margin(50)

		return m, tea.ClearScreen

	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {

		case "ctrl+c":
			return m, tea.Quit

		case "j", "down":
			m.cursor = min(m.cursor+1, len(m.configs)-1)

		case "k", "up":
			m.cursor = max(m.cursor-1, 0)

		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]

		case "enter":
			m.nextScreen()

		}
	}

	return m, nil
}

func tuiMain(c *conf.Configuration) error {
	lockAfter, err := lib.CreateLockBasedOnConfigs(c)
	if err != nil {
		c.Logger.Info("something went wrong while trying to create a new lockfile based on your config files in your source directory", "err", err)
		return err
	}

	// config/DEPENDENCIES file parsing
	globalDependencies, err := lib.ParseGlobalDependencies(c.SourceCfgDir)
	if err != nil {
		c.Logger.Error("couldn't parse global dependencies file", "path", c.SourceCfgDir, "err", err)
		return err
	}

	lockAfter.GlobalDependencies = globalDependencies

	{
		m := initModel(lockAfter)
		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			os.Exit(1)
		}
	}

	return nil
}
