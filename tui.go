package main

import (
	conf "blanktiger/hm/configuration"
	"blanktiger/hm/lib"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"

	blist "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
)

// IDEAS:
// - [x] I want a list of all configs with information on if they would be copied over
//   with an ability to change that fact by selecting or unselecting them in a list
//
// - [ ] I want a list of all global dependencies that would be installed
//
// - [ ] An editor of global dependencies list (they would be permanently stored on disk
//   after we are done editing)
//
// - [ ] An option to permanently hide/unhide a config from being copied (makes it so that
//   I don't have to manually hide them)

type screen int

const (
	configsScreen screen = iota
	globalDepsScreen
)

func isValidScreen(screenId int) bool {
	switch screenId {
	case int(configsScreen), int(globalDepsScreen):
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
	lockfile *lib.Lockfile

	// all configs from lockfile (non hidden + hidden)
	configs []lib.Config
	// index in the configs list -> is it selected
	configSelection map[int]bool
	// this will be filled after selection is done on the first screen
	// right before going to the next screen
	selectedConfigs []lib.Config
	// global dependency package name -> index in the original lockfile.GlobalDependencies list
	globalDepToIdx map[string]int

	currentScreen screen
	termWidth     int
	termHeight    int

	configsList    blist.Model
	globalDepsList blist.Model
	listHeight     int
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

	configsList := blist.New(allConfigNames, itemDelegate{}, defaultWidth, defaultListHeight)
	{
		configsList.Title = "Configs - select the ones you wanna copy/symlink."
		configsList.SetShowStatusBar(false)
		configsList.SetFilteringEnabled(true)
		configsList.Styles.PaginationStyle = paginationStyle
		configsList.Styles.TitleBar.AlignHorizontal(lg.Center)
		configsList.Styles.HelpStyle = helpStyle
		configsList.AdditionalFullHelpKeys = additionalFullHelpKeys
		configsList.AdditionalShortHelpKeys = additionalShortHelpKeys
	}

	globalDepToIdx := make(map[string]int)
	globalDepNames := []blist.Item{}
	for idx, dep := range lockfile.GlobalDependencies {
		// TODO: check if it works without any spaces
		for subDep := range strings.SplitSeq(dep.Instruction.Pkg, " ") {
			globalDepToIdx[subDep] = idx
			depListItem := listItem("[*] " + subDep)
			globalDepNames = append(globalDepNames, depListItem)
		}
	}

	globalDepsList := blist.New(globalDepNames, itemDelegate{}, defaultWidth, defaultListHeight)
	{
		globalDepsList.Title = "Packages (global dependencies) - select ones that should be installed."
		globalDepsList.SetShowStatusBar(false)
		globalDepsList.SetFilteringEnabled(true)
		globalDepsList.Styles.PaginationStyle = paginationStyle
		globalDepsList.Styles.TitleBar.AlignHorizontal(lg.Center)
		globalDepsList.Styles.HelpStyle = helpStyle
		globalDepsList.AdditionalFullHelpKeys = additionalFullHelpKeys
		globalDepsList.AdditionalShortHelpKeys = additionalShortHelpKeys
	}

	return model{
		lockfile:        lockfile,
		configs:         allConfigs,
		configSelection: selected,
		globalDepToIdx:  globalDepToIdx,

		listHeight:     defaultListHeight,
		configsList:    configsList,
		globalDepsList: globalDepsList,
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
	case configsScreen:
		s = m.configsToInstallScreen()
	case globalDepsScreen:
		s = m.pkgsToInstallScreen()
	default:
		panic("invalid screen id")
	}
	return borderedWindowStyle.Render(s)
}

const accentColor = "#17d87e"

func (m model) configsToInstallScreen() string {
	list := m.configsList.View()
	var listStyle = listStyle.Width(m.termWidth)
	return lg.JoinVertical(lg.Top, listStyle.Render(list))
}

func (m model) pkgsToInstallScreen() string {
	list := m.globalDepsList.View()
	var listStyle = listStyle.Width(m.termWidth)
	return lg.JoinVertical(lg.Top, listStyle.Render(list))
}

func (m *model) nextScreen() {
	newScreenId := int(m.currentScreen) + 1
	if isValidScreen(newScreenId) {
		m.currentScreen = screen(newScreenId)
	}
}

func additionalFullHelpKeys() []key.Binding {
	return longHelpKeys
}

func additionalShortHelpKeys() []key.Binding {
	return shortHelpKeys
}

var widthOffset = 90

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {

	case tea.WindowSizeMsg:
		windowSize := msg.(tea.WindowSizeMsg)
		m.updateSize(windowSize)
		return m, tea.ClearScreen

	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			updatedListItems := m.updateList()
			cmd := m.configsList.SetItems(updatedListItems)
			return m, cmd

		case " ":
			m.nextScreen()

		}
	}

	var cmd tea.Cmd
	switch m.currentScreen {
	case configsScreen:
		m.configsList, cmd = m.configsList.Update(msg)
	case globalDepsScreen:
		m.globalDepsList, cmd = m.globalDepsList.Update(msg)
	}
	return m, cmd
}

func (m model) updateSize(windowSize tea.WindowSizeMsg) {
	m.termWidth = windowSize.Width - widthOffset
	m.termHeight = windowSize.Height
	m.configsList.SetWidth(m.termWidth)
	m.configsList.SetHeight(m.termHeight - 10)
	m.globalDepsList.SetWidth(m.termWidth)
	m.globalDepsList.SetHeight(m.termHeight - 10)
	m.configsList.Styles.HelpStyle.Width(m.termWidth).Align(lg.Center)
}

func (m model) updateList() (res []blist.Item) {
	cur := m.configsList.GlobalIndex()
	m.configSelection[cur] = !m.configSelection[cur]

	for idx := range m.configsList.Items() {
		if isSelected, ok := m.configSelection[idx]; ok {
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

type helpKey struct {
	shortBinding key.Binding
	longBinding  key.Binding
}

var help = []helpKey{
	{
		shortBinding: key.NewBinding(
			key.WithKeys("Tab"),
			key.WithHelp("Tab", "Toggle"),
		),
		longBinding: key.NewBinding(
			key.WithKeys("Tab"),
			key.WithHelp("Tab", "Toggle current option"),
		),
	},
	{
		shortBinding: key.NewBinding(
			key.WithKeys("Space"),
			key.WithHelp("Space", "Next"),
		),
		longBinding: key.NewBinding(
			key.WithKeys("Space"),
			key.WithHelp("Space", "Go to the next page"),
		),
	},
}

var shortHelpKeys = make([]key.Binding, len(help))
var longHelpKeys = make([]key.Binding, len(help))

func tuiMain(c *conf.Configuration) error {
	for idx, h := range help {
		shortHelpKeys[idx] = h.shortBinding
		longHelpKeys[idx] = h.longBinding
	}

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
