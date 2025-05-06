package main

import (
	conf "blanktiger/hm/configuration"
	"blanktiger/hm/lib"
	"fmt"
	"io"
	"os"
	"slices"

	blist "github.com/charmbracelet/bubbles/list"
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

type listItem string

func (l listItem) FilterValue() string {
	return string(l)
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                              { return 1 }
func (d itemDelegate) Spacing() int                             { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *blist.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m blist.Model, index int, item blist.Item) {
	i, ok := item.(listItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s", i)
	prefixSelected := "-> "
	prefixNormal := "   "

	fn := listItemStyle.Render
	if index == m.Index() {
		str = prefixSelected + str
		fn = selectedListItemStyle.Render
	} else {
		// TODO: maybe just highlight the arrow instead of the whole line
		str = prefixNormal + str
	}

	fmt.Fprint(w, fn(str))
}

// styles
var (
	baseListItemPaddingLeft = 20
	titleStyle              = lg.NewStyle().PaddingLeft(20)
	listItemStyle           = lg.NewStyle().PaddingLeft(baseListItemPaddingLeft)
	selectedListItemStyle   = lg.NewStyle().Foreground(lg.Color(accentColor)).Bold(true).PaddingLeft(baseListItemPaddingLeft)
	paginationStyle         = blist.DefaultStyles().PaginationStyle
	helpStyle               = blist.DefaultStyles().HelpStyle

	listStyle      = lg.NewStyle().AlignHorizontal(lg.Center)
	footerStyle    = lg.NewStyle().Foreground(lg.Color(accentColor)).AlignHorizontal(lg.Center)
	selectedStyle  = lg.NewStyle().Bold(true).Foreground(lg.Color(accentColor))
	listEntryStyle = lg.NewStyle()
)

// TODO: set up logging, logger should output everything to a file

type model struct {
	lockfile      *lib.Lockfile
	configs       []lib.Config
	selected      map[int]bool
	currentScreen screen
	cursor        int
	dbgMsg        string

	mList      blist.Model
	listHeight int
	termWidth  int
	termHeight int
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

	allConfigs := slices.Concat(lockfile.Configs, lockfile.HiddenConfigs)
	allConfigNames := []blist.Item{}
	for _, cfg := range lockfile.Configs {
		item := listItem("[*] " + cfg.Name)
		allConfigNames = append(allConfigNames, item)
	}
	for _, cfg := range lockfile.HiddenConfigs {
		item := listItem("[ ] " + cfg.Name)
		allConfigNames = append(allConfigNames, item)
	}

	defaultWidth := 20
	defaultListHeight := 15
	mList := blist.New(allConfigNames, itemDelegate{}, defaultWidth, defaultListHeight)
	mList.Title = "Configs - select the ones you wanna copy/symlink."
	mList.SetShowStatusBar(false)
	// TODO: make this work, currently if I have list filtered to one item,
	// toggling that item actually toggles first item from the unfiltered list
	mList.SetFilteringEnabled(false)
	mList.Styles.PaginationStyle = paginationStyle
	mList.Styles.TitleBar.AlignHorizontal(lg.Center)
	mList.Styles.HelpStyle = helpStyle

	return model{
		lockfile: lockfile,
		configs:  allConfigs,
		selected: selected,

		listHeight: defaultListHeight,
		mList:      mList,
	}
}

func (m model) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m model) View() string {
	var borderedWindowStyle = lg.NewStyle().
		Border(lg.RoundedBorder(), true).
		BorderForeground(lg.Color(accentColor)).
		Width(m.termWidth).
		MarginLeft(widthOffset / 2)
		// MarginTop(10 * m.termHeight / 100)

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

func (m model) configsToInstallScreen() string {
	list := m.mList.View()
	var listStyle = listStyle.Width(m.termWidth)
	var footerStyle = footerStyle.Width(m.termWidth)
	return lg.JoinVertical(lg.Top, listStyle.Render(list), footerStyle.Render("\nSelect by pressing: <space>\nAccept by pressing: <enter>"))
}

func (m *model) nextScreen() {
	newScreenId := int(m.currentScreen) + 1
	if isValidScreen(newScreenId) {
		m.currentScreen = screen(newScreenId)
	}
}

var widthOffset = 50

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {

	case tea.WindowSizeMsg:
		windowSize := msg.(tea.WindowSizeMsg)
		m.termWidth = windowSize.Width - widthOffset
		m.termHeight = windowSize.Height
		m.mList.SetWidth(m.termWidth)
		m.mList.SetHeight(m.termHeight - 10)
		return m, nil

	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {

		case "ctrl+c":
			return m, tea.Quit

		case "j", "down":
			m.cursor = min(m.cursor+1, len(m.configs)-1)

		case "k", "up":
			m.cursor = max(m.cursor-1, 0)

		case " ", "tab":
			updatedListItems := m.updateList()
			cmd := m.mList.SetItems(updatedListItems)
			return m, cmd

		case "enter":
			m.nextScreen()

		}
	}

	var cmd tea.Cmd
	m.mList, cmd = m.mList.Update(msg)
	return m, cmd
}

func (m model) updateList() (res []blist.Item) {
	cur := m.mList.Cursor()
	m.selected[cur] = !m.selected[cur]

	for idx := range m.mList.Items() {
		if isSelected, ok := m.selected[idx]; ok {
			cfgAtIdx := m.configs[idx]
			if isSelected {
				res = append(res, listItem("[*] "+cfgAtIdx.Name))
			} else {
				res = append(res, listItem("[ ] "+cfgAtIdx.Name))
			}
		}
	}

	return res
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
