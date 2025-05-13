package main

import (
	"blanktiger/hm/configuration"
	conf "blanktiger/hm/configuration"
	"blanktiger/hm/instructions"
	"blanktiger/hm/lib"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"

	blist "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
)

type screen int

const (
	cliArgsScreen screen = iota
	configsScreen
	globalDepsScreen
	// will gather information on wheter the user wants to save info from previous
	// screens to disk (make configs public/private, include/exclude global dependencies)
	userChoicesScreen
)

func isValidScreen(screenId int) bool {
	switch screenId {
	case int(cliArgsScreen), int(configsScreen), int(globalDepsScreen), int(userChoicesScreen):
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
	conf     *configuration.Configuration

	// all configs from lockfile (non hidden + hidden)
	configs []lib.Config
	// index in the configs list -> is it selected
	configSelection map[int]bool

	// this is all global dependencies that are split, so that when we iterate
	// it's like all of them are on a separate line in the DEPENDENCIES file
	flatGlobalDeps []lib.GlobalDependency
	// index in the global dependencies list -> is it selected
	globalDepsSelection map[int]bool

	choiceSelection map[int]bool
	userChoices     choices

	cliArgsSelection map[int]bool

	currentScreen screen
	termWidth     int
	termHeight    int

	cliArgsList    blist.Model
	configsList    blist.Model
	globalDepsList blist.Model
	choicesList    blist.Model
	listHeight     int
}

type choices struct {
	PersistConfigSelection     bool `txt:"Persist config selection"`
	PersistGlobalDepsSelection bool `txt:"Persist global dependencies selection"`
}

func initModel(lockfile *lib.Lockfile, conf *configuration.Configuration) model {
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
	defaultListHeight := 25

	configsList := blist.New(allConfigNames, itemDelegate{}, defaultWidth, defaultListHeight)
	{
		configsList.Title = "Configs - select the ones you want to copy/symlink."
		configsList.SetShowStatusBar(false)
		configsList.SetFilteringEnabled(true)
		configsList.Styles.PaginationStyle = paginationStyle
		configsList.Styles.HelpStyle = helpStyle
		configsList.AdditionalFullHelpKeys = additionalFullHelpKeysGeneral
		configsList.AdditionalShortHelpKeys = additionalShortHelpKeysGeneral
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
		globalDepsList.Styles.HelpStyle = helpStyle
		globalDepsList.AdditionalFullHelpKeys = additionalFullHelpKeysGeneral
		globalDepsList.AdditionalShortHelpKeys = additionalShortHelpKeysGeneral
	}

	cliArgsSelection := make(map[int]bool)
	cliArgsTxt := buildListValuesBasedOnType(*conf)
	cliArgsList := blist.New(cliArgsTxt, itemDelegate{}, defaultWidth, defaultListHeight)
	{
		cliArgsList.Title = "Program flags"
		cliArgsList.SetShowStatusBar(false)
		cliArgsList.SetFilteringEnabled(true)
		cliArgsList.Styles.PaginationStyle = paginationStyle
		cliArgsList.Styles.HelpStyle = helpStyle
		cliArgsList.AdditionalFullHelpKeys = additionalFullHelpKeysGeneral
		cliArgsList.AdditionalShortHelpKeys = additionalShortHelpKeysGeneral
	}

	choiceSelection := make(map[int]bool)
	userChoices := choices{}
	choicesTxt := buildListValuesBasedOnType(userChoices)
	choicesList := blist.New(choicesTxt, itemDelegate{}, defaultWidth, defaultListHeight)
	{
		choicesList.Title = "Additional information"
		choicesList.SetShowStatusBar(false)
		choicesList.SetFilteringEnabled(true)
		choicesList.Styles.PaginationStyle = paginationStyle
		choicesList.Styles.HelpStyle = helpStyle
		choicesList.AdditionalFullHelpKeys = additionalFullHelpKeysLastScreen
		choicesList.AdditionalShortHelpKeys = additionalShortHelpKeysLastScreen
	}

	return model{
		lockfile: lockfile,
		conf:     conf,

		configs:         allConfigs,
		configSelection: selected,

		flatGlobalDeps:      flatGlobalDeps,
		globalDepsSelection: globalDepsSelection,

		userChoices:     userChoices,
		choiceSelection: choiceSelection,

		cliArgsSelection: cliArgsSelection,

		listHeight:     defaultListHeight,
		cliArgsList:    cliArgsList,
		configsList:    configsList,
		globalDepsList: globalDepsList,
		choicesList:    choicesList,
	}
}

// this function skips all values that are not bool in the passed in `value`
func buildListValuesBasedOnType(value any) []blist.Item {
	list := []blist.Item{}

	t := reflect.TypeOf(value)
	v := reflect.ValueOf(value)
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Bool {
			continue
		}

		tagTxt := field.Tag.Get("txt")
		if tagTxt == "exclude" {
			continue
		}

		value := v.Field(i)
		prefix := "[ ] "
		if value.Bool() {
			prefix = "[*] "
		}

		list = append(list, listItem(prefix+tagTxt))
	}

	return list
}

func (m model) Init() tea.Cmd {
	fmt.Println()
	return nil
}

func (m model) View() string {
	list := ""
	switch m.currentScreen {
	case cliArgsScreen:
		list = m.cliArgsList.View()
	case configsScreen:
		list = m.configsList.View()
	case globalDepsScreen:
		list = m.globalDepsList.View()
	case userChoicesScreen:
		list = m.choicesList.View()
	default:
		panic("invalid screen id")
	}

	listStyle := listStyle.Width(m.termWidth)
	windowStyle := lg.NewStyle().Width(m.termWidth)
	return windowStyle.Render(lg.JoinVertical(lg.Top, listStyle.Render(list)))
}

const accentColor = "#17d87e"

func (m *model) nextScreen() tea.Cmd {
	switch m.currentScreen {
	// NOTE: no need for special handling, cli args are saved automatically while updating them
	case cliArgsScreen:
	case configsScreen:
		selectedConfigs := []lib.Config{}
		hiddenConfigs := []lib.Config{}
		for idx, isSelected := range m.configSelection {
			if isSelected {
				selectedConfigs = append(selectedConfigs, m.configs[idx])
			} else {
				hiddenConfigs = append(hiddenConfigs, m.configs[idx])
			}
		}
		m.lockfile.Configs = selectedConfigs
		m.lockfile.HiddenConfigs = hiddenConfigs

	case globalDepsScreen:
		selectedGlobalDeps := []lib.GlobalDependency{}
		for idx, isSelected := range m.globalDepsSelection {
			if isSelected {
				selectedGlobalDeps = append(selectedGlobalDeps, m.flatGlobalDeps[idx])
			}
			// TODO: hidden configs should also probably be preserved
			// else {
			// 	m.hiddenGlobalDeps = append(m.hiddenGlobalDeps, m.flatGlobalDeps[idx])
			// }
		}
		m.lockfile.GlobalDependencies = selectedGlobalDeps

	// NOTE: no need for special handling, choices are always saved when we are updating them
	case userChoicesScreen:
	}

	newScreenId := int(m.currentScreen) + 1
	if isValidScreen(newScreenId) {
		m.currentScreen = screen(newScreenId)
	}

	return nil
}

func (m *model) prevScreen() tea.Cmd {
	newScreenId := int(m.currentScreen) - 1
	if isValidScreen(newScreenId) {
		m.currentScreen = screen(newScreenId)
	} else {
		m.currentScreen = screen(0)
	}

	return nil
}

var shortHelpKeysGeneral = make([]key.Binding, len(helpGeneral))
var longHelpKeysGeneral = make([]key.Binding, len(helpGeneral))

func additionalFullHelpKeysGeneral() []key.Binding {
	return longHelpKeysGeneral
}

func additionalShortHelpKeysGeneral() []key.Binding {
	return shortHelpKeysGeneral
}

var shortHelpKeysLastScreen = make([]key.Binding, len(helpGeneral)+len(helpLastScreen))
var longHelpKeysLastScreen = make([]key.Binding, len(helpGeneral)+len(helpLastScreen))

func additionalFullHelpKeysLastScreen() []key.Binding {
	return longHelpKeysLastScreen
}

func additionalShortHelpKeysLastScreen() []key.Binding {
	return shortHelpKeysLastScreen
}

var sizeUpdates = 0

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {

	case tea.WindowSizeMsg:
		sizeUpdates++
		windowSize := msg.(tea.WindowSizeMsg)
		m.updateSize(windowSize)
		if sizeUpdates > 1 {
			return m, tea.ClearScreen
		} else {
			return m, nil
		}

	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {

		case "ctrl+c", "q":
			// TODO: maybe there is a nicer way to do this with bubbletea (something builtin)
			os.Exit(1)

		case " ":
			return m, m.updateAfterSelectingInList()

		case "enter":
			if !isValidScreen(int(m.currentScreen) + 1) {
				return m, tea.Quit
			}

		case "tab":
			return m, m.nextScreen()

		case "shift+tab":
			return m, m.prevScreen()

		}
	}

	var cmd tea.Cmd
	switch m.currentScreen {
	case cliArgsScreen:
		m.cliArgsList, cmd = m.cliArgsList.Update(msg)
	case configsScreen:
		m.configsList, cmd = m.configsList.Update(msg)
	case globalDepsScreen:
		m.globalDepsList, cmd = m.globalDepsList.Update(msg)
	case userChoicesScreen:
		m.choicesList, cmd = m.choicesList.Update(msg)
	}
	return m, cmd
}

func (m *model) updateSize(windowSize tea.WindowSizeMsg) {
	m.termWidth = windowSize.Width
	{
		m.configsList.SetWidth(m.termWidth)
		m.configsList.Styles.TitleBar.Width(m.termWidth)
		m.configsList.Styles.Title.Width(m.termWidth)
		m.configsList.Styles.HelpStyle.Width(m.termWidth).Align(lg.Center)
	}
	{
		m.globalDepsList.SetWidth(m.termWidth)
		m.globalDepsList.Styles.TitleBar.Width(m.termWidth)
		m.globalDepsList.Styles.Title.Width(m.termWidth)
		m.globalDepsList.Styles.HelpStyle.Width(m.termWidth).Align(lg.Center)
	}
	{
		m.choicesList.SetWidth(m.termWidth)
		m.choicesList.Styles.TitleBar.Width(m.termWidth)
		m.choicesList.Styles.Title.Width(m.termWidth)
		m.choicesList.Styles.HelpStyle.Width(m.termWidth).Align(lg.Center)
	}
}

func (m *model) updateAfterSelectingInList() tea.Cmd {
	switch m.currentScreen {
	case cliArgsScreen:
		cur := m.cliArgsList.GlobalIndex()

		pv := reflect.ValueOf(m.conf)
		v := pv.Elem()

		// toggle fields value
		curValue := v.Field(cur).Bool()
		v.Field(cur).SetBool(!curValue)

		return m.cliArgsList.SetItems(buildListValuesBasedOnType(*m.conf))

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

	case userChoicesScreen:
		cur := m.choicesList.GlobalIndex()

		pv := reflect.ValueOf(&m.userChoices)
		v := pv.Elem()

		// toggle fields value
		curValue := v.Field(cur).Bool()
		v.Field(cur).SetBool(!curValue)

		return m.choicesList.SetItems(buildListValuesBasedOnType(m.userChoices))

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

var helpGeneral = []helpKey{
	{
		shortBinding: key.NewBinding(
			key.WithKeys("Space"),
			key.WithHelp("Space", "Toggle"),
		),
		longBinding: key.NewBinding(
			key.WithKeys("Space"),
			key.WithHelp("Space", "Toggle current option"),
		),
	},
	{
		shortBinding: key.NewBinding(
			key.WithKeys("Tab"),
			key.WithHelp("Tab", "Next"),
		),
		longBinding: key.NewBinding(
			key.WithKeys("Tab"),
			key.WithHelp("Tab", "Go to the next page"),
		),
	},
	{
		shortBinding: key.NewBinding(
			key.WithKeys("Shift+Tab"),
			key.WithHelp("Shift+Tab", "Prev"),
		),
		longBinding: key.NewBinding(
			key.WithKeys("Shift+Tab"),
			key.WithHelp("Shift+Tab", "Go to the previous page"),
		),
	},
}

var helpLastScreen = []helpKey{
	{
		shortBinding: key.NewBinding(
			key.WithKeys("Enter"),
			key.WithHelp("Enter", "Run"),
		),
		longBinding: key.NewBinding(
			key.WithKeys("Enter"),
			key.WithHelp("Enter", "Finish selection and run"),
		),
	},
}

func tuiMain(c *conf.Configuration) error {
	for idx, h := range helpGeneral {
		shortHelpKeysGeneral[idx] = h.shortBinding
		longHelpKeysGeneral[idx] = h.longBinding
		shortHelpKeysLastScreen[idx] = h.shortBinding
		longHelpKeysLastScreen[idx] = h.longBinding
	}

	for idxOffset, h := range helpLastScreen {
		idx := idxOffset + len(helpGeneral) - 1
		shortHelpKeysLastScreen[idx] = h.shortBinding
		longHelpKeysLastScreen[idx] = h.longBinding
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
		m := initModel(lockAfter, c)
		p := tea.NewProgram(m)

		_m, err := p.Run()
		if err != nil {
			return err
		}

		m = _m.(model)
		return executeBasedOnUserSelection(m)
	}
}

func executeBasedOnUserSelection(m model) error {
	c := m.conf
	lockBefore, err := lib.ReadOrCreateLockfile(c.LockfilePath)
	if err != nil {
		c.Logger.Info("encountered an error while trying to read an existing lockfile (probably doesnt exist), creating a new one instead", "err", err)
		lockBefore = &lib.EmptyLockfile
	}

	lockAfter := m.lockfile

	if m.userChoices.PersistConfigSelection {
		err = lockAfter.PersistConfigSelection()
		if err != nil {
			return err
		}
	}

	if m.userChoices.PersistGlobalDepsSelection {
		err = lockAfter.PersistGlobalDepsSelection(c.SourceCfgDir)
		if err != nil {
			return err
		}
	}

	lib.CopyInstallInfo(lockBefore, lockAfter)

	globalDepsChanged := lib.DidGlobalDependenciesChange(&lockBefore.GlobalDependencies, &lockAfter.GlobalDependencies)
	globalDepsInstalled := lib.WereGlobalDependenciesInstalled(&lockAfter.GlobalDependencies)
	if c.Install || c.OnlyInstall || c.Upgrade {
		if globalDepsChanged || !globalDepsInstalled || c.Upgrade {
			err = lib.InstallGlobalDependencies(&lockAfter.GlobalDependencies)
			if err != nil {
				lib.Logger.Error("something went wrong while trying to install global dependencies", "err", err)
				return err
			}
		} else {
			lib.Logger.Info("global dependencies didn't change since last installation, not installing", "depsChanged", globalDepsChanged, "previouslyInstalled", globalDepsInstalled)
		}
	}

	if !c.OnlyUninstall && !c.OnlyInstall {
		toSymlink := lockAfter.Configs

		if c.CopyMode {
			err = lib.Copy(c, toSymlink)
		} else {
			err = lib.Symlink(c, toSymlink)
		}
		if err != nil {
			c.Logger.Error("encountered an error while copying/symlinking", "error", err)
			return err
		}

		toRemove := lockAfter.HiddenConfigs
		err = lib.Remove(c, toRemove)
	} else {
		lib.Logger.Info("skipping copying/symlinking the config, because --only-install or --only-uninstall was passed")
	}

	if (c.Install || c.OnlyInstall || c.Upgrade) && !c.OnlyUninstall {
		infoForUpdate := lib.Install(lockAfter)
		lockAfter.UpdateInstallInfo(infoForUpdate)
	}

	if (c.Uninstall || c.OnlyUninstall) && !c.OnlyInstall {
		infoForUpdate := lib.Uninstall(lockAfter)
		lockAfter.UpdateInstallInfo(infoForUpdate)
	}

	err = lockAfter.Save(c.LockfilePath, c.DefaultIndent)
	if err != nil {
		lib.Logger.Error("something went wrong while trying to save the lockfile", "err", err)
	}

	diff := lib.DiffLocks(*lockBefore, *lockAfter)
	err = diff.Save(c.LockfileDiffPath, c.DefaultIndent)
	if err != nil {
		lib.Logger.Error("something went wrong while trying to save the lockfile diff", "err", err)
	}

	return nil
}
