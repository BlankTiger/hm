package main

import (
	conf "blanktiger/hm/configuration"
	"blanktiger/hm/instructions"
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
	baseListItemPaddingLeft = 0
	titleStyle              = lg.NewStyle()
	listItemStyle           = lg.NewStyle().PaddingLeft(baseListItemPaddingLeft)
	selectedListItemStyle   = lg.NewStyle().Foreground(lg.Color(accentColor)).Bold(true).PaddingLeft(baseListItemPaddingLeft)
	paginationStyle         = blist.DefaultStyles().PaginationStyle
	helpStyle               = blist.DefaultStyles().HelpStyle

	listStyle      = lg.NewStyle()
	footerStyle    = lg.NewStyle().Foreground(lg.Color(accentColor))
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

	// this is all global dependencies that are split, so that when we iterate
	// it's like all of them are on a separate line in the DEPENDENCIES file
	flatGlobalDeps []lib.GlobalDependency
	// index in the global dependencies list -> is it selected
	globalDepsSelection map[int]bool

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
		// configsList.Styles.TitleBar.AlignHorizontal(lg.Center)
		// configsList.Styles.TitleBar = titleStyle
		configsList.Styles.HelpStyle = helpStyle
		configsList.AdditionalFullHelpKeys = additionalFullHelpKeys
		configsList.AdditionalShortHelpKeys = additionalShortHelpKeys
	}

	globalDepsSelection := make(map[int]bool)
	globalDepNames := []blist.Item{}
	flatGlobalDeps := []lib.GlobalDependency{}
	depIdx := 0
	for _, dep := range lockfile.GlobalDependencies {
		if dep.Instruction.Method == instructions.Bash {
			flatGlobalDeps = append(flatGlobalDeps, dep)
			globalDepNames = append(globalDepNames, listItem("[*] "+formatGlobalDep(dep)))
			globalDepsSelection[depIdx] = true
			depIdx++
			continue
		}

		for subDepPkg := range strings.SplitSeq(dep.Instruction.Pkg, " ") {
			subDep := dep
			inst := *subDep.Instruction
			subDep.Instruction = &inst
			subDep.Instruction.Pkg = subDepPkg
			flatGlobalDeps = append(flatGlobalDeps, subDep)
			globalDepNames = append(globalDepNames, listItem("[*] "+formatGlobalDep(subDep)))
			globalDepsSelection[depIdx] = true
			depIdx++
		}
	}

	globalDepsList := blist.New(globalDepNames, itemDelegate{}, defaultWidth, defaultListHeight)
	{
		globalDepsList.Title = "Packages (global dependencies) - select ones that should be installed."
		globalDepsList.SetShowStatusBar(false)
		globalDepsList.SetFilteringEnabled(true)
		globalDepsList.Styles.PaginationStyle = paginationStyle
		// globalDepsList.Styles.TitleBar.AlignHorizontal(lg.Center)
		globalDepsList.Styles.HelpStyle = helpStyle
		globalDepsList.AdditionalFullHelpKeys = additionalFullHelpKeys
		globalDepsList.AdditionalShortHelpKeys = additionalShortHelpKeys
	}

	return model{
		lockfile:        lockfile,
		configs:         allConfigs,
		configSelection: selected,

		flatGlobalDeps:      flatGlobalDeps,
		globalDepsSelection: globalDepsSelection,

		listHeight:     defaultListHeight,
		configsList:    configsList,
		globalDepsList: globalDepsList,
	}
}

func (m model) Init() tea.Cmd {
	fmt.Println()
	return nil
}

func (m model) View() string {
	var windowStyle = lg.NewStyle().
		Width(m.termWidth)

	s := ""
	switch m.currentScreen {
	case configsScreen:
		s = m.configsToInstallScreen()
	case globalDepsScreen:
		s = m.pkgsToInstallScreen()
	default:
		panic("invalid screen id")
	}
	return windowStyle.Render(s)
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {

	case tea.WindowSizeMsg:
		windowSize := msg.(tea.WindowSizeMsg)
		m.updateSize(windowSize)
		return m, nil

	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			return m, m.updateAfterSelectingInList()

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

func (m *model) updateSize(windowSize tea.WindowSizeMsg) {
	m.termWidth = windowSize.Width
	m.termHeight = windowSize.Height
	m.configsList.SetWidth(m.termWidth)
	m.configsList.SetHeight(m.termHeight - 10)
	m.globalDepsList.SetWidth(m.termWidth)
	m.globalDepsList.SetHeight(m.termHeight - 10)
	m.configsList.Styles.TitleBar.Width(m.termWidth)
	m.configsList.Styles.Title.Width(m.termWidth)
	m.configsList.Styles.HelpStyle.Width(m.termWidth).Align(lg.Center)
	m.globalDepsList.Styles.TitleBar.Width(m.termWidth)
	m.globalDepsList.Styles.Title.Width(m.termWidth)
	m.globalDepsList.Styles.HelpStyle.Width(m.termWidth).Align(lg.Center)
}

func (m *model) updateAfterSelectingInList() tea.Cmd {
	switch m.currentScreen {
	case configsScreen:
		cur := m.configsList.GlobalIndex()
		m.configSelection[cur] = !m.configSelection[cur]

		updatedItems := []blist.Item{}
		for idx := range m.configsList.Items() {
			if isSelected, ok := m.configSelection[idx]; ok {
				cfgAtIdx := m.configs[idx]
				if isSelected {
					updatedItems = append(updatedItems, listItem("[*] "+cfgAtIdx.Name))
				} else {
					updatedItems = append(updatedItems, listItem("[ ] "+cfgAtIdx.Name))
				}
			}
		}

		return m.configsList.SetItems(updatedItems)

	case globalDepsScreen:
		cur := m.globalDepsList.GlobalIndex()
		m.globalDepsSelection[cur] = !m.globalDepsSelection[cur]

		updatedItems := []blist.Item{}
		for idx := range m.globalDepsList.Items() {
			if isSelected, ok := m.globalDepsSelection[idx]; ok {
				dep := m.flatGlobalDeps[idx]
				if isSelected {
					updatedItems = append(updatedItems, listItem("[*] "+formatGlobalDep(dep)))
				} else {
					updatedItems = append(updatedItems, listItem("[ ] "+formatGlobalDep(dep)))
				}
			}
		}

		return m.globalDepsList.SetItems(updatedItems)

	default:
		panic("we somehow got to an incorrect screen, exiting")
	}
}

func formatGlobalDep(dep lib.GlobalDependency) string {
	return string(dep.Instruction.Method) + ":" + dep.Instruction.Pkg
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
