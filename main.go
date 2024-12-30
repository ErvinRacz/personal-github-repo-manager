package main

import (
	gh "ervinracz/personal-github-repo-manager/ghrepos"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	ghFacade         *gh.GHAPIFacade
	spinner          spinner.Model
	list             list.Model
	choice           string
	quitting         bool
	repos            []*gh.Repo
	currentRepoIndex int
	err              error
	confirmMsg       string
	loading          bool
}

type ghErrMsg error
type ghActionPerformedMsg string
type ghRepoDeletedMsg string
type ghReposFetchedMsg []*gh.Repo
type ghAction func() error

func ghFetchRepos(ghFacade *gh.GHAPIFacade) tea.Cmd {
	return func() tea.Msg {
		repos, err := ghFacade.GetRepos()
		if err != nil {
			return ghErrMsg(err)
		}
		return ghReposFetchedMsg(repos)
	}
}

func ghPerform(action ghAction, successMsg string) tea.Cmd {
	return func() tea.Msg {
		err := action()
		if err != nil {
			return ghErrMsg(err)
		}
		return ghActionPerformedMsg(successMsg)
	}
}

func ghDelete(repo *gh.Repo) tea.Cmd {
	return func() tea.Msg {
		err := repo.Delete()
		if err != nil {
			return ghErrMsg(err)
		}
		return ghRepoDeletedMsg("")
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, ghFetchRepos(m.ghFacade))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case ghRepoDeletedMsg:
		m.loading = false
		m.confirmMsg = "Repo deleted"
		if m.currentRepoIndex > 0 {
			m.currentRepoIndex -= 1
		}
		return m, ghFetchRepos(m.ghFacade)
	case ghReposFetchedMsg:
		m.loading = false
		m.repos = msg
		return m, nil
	case ghActionPerformedMsg:
		m.loading = false
		m.confirmMsg = string(msg)
		return m, nil
	case ghErrMsg:
		m.loading = false
		m.err = msg
		return m, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			switch m.choice {
			case "Next":
				if m.currentRepoIndex < len(m.repos)-1 {
					m.currentRepoIndex += 1
				} else {
					m.quitting = true
					return m, tea.Quit
				}
			case "Previous":
				if m.currentRepoIndex > 0 {
					m.currentRepoIndex -= 1
				} else {
					m.quitting = true
					return m, tea.Quit
				}
			case "Open in browser":
				m.loading = true
				return m, ghPerform(m.repos[m.currentRepoIndex].Open, "Repo opened")
			case "Archive":
				m.loading = true
				return m, ghPerform(m.repos[m.currentRepoIndex].Archive, "Repo archived")
			case "Unarchive":
				m.loading = true
				return m, ghPerform(m.repos[m.currentRepoIndex].Unarchive, "Repo unarchived")
			case "Make it public":
				m.loading = true
				return m, ghPerform(m.repos[m.currentRepoIndex].MakePublic, "Repo made public")
			case "Make it private":
				m.loading = true
				return m, ghPerform(m.repos[m.currentRepoIndex].MakePrivate, "Repo made private")
			case "Delete":
				m.loading = true
				return m, ghDelete(m.repos[m.currentRepoIndex])
			}
			return m, nil
		}
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	if m.loading {
		return fmt.Sprintf("\n\n  %s Loading...", m.spinner.View())
	}
	if m.quitting {
		if m.currentRepoIndex >= len(m.repos)-1 {
			return quitTextStyle.Render("There are no more repos. Have a nice day!")
		}
		return quitTextStyle.Render("Have a nice day!")
	}

	cr := m.repos[m.currentRepoIndex]
	archLabel := "(not archived)"
	if cr.Archived {
		archLabel = "(archived)"
	}

	m.list.Title = fmt.Sprintf("[%d/%d] %s (%s) %s",
		m.currentRepoIndex+1,
		len(m.repos),
		cr.Name,
		cr.Visibility,
		archLabel,
	)
	return "\n" + m.list.View()
}

func main() {
	const no_args_err_msg = "Provide Github API Key as argument"

	var ghApiKey string
	flag.StringVar(&ghApiKey, "ghapikey", "", "Github API Key.")
	var ghOwner string
	flag.StringVar(&ghOwner, "owner", "", "Owner of the Github account")
	var debugLevel bool
	flag.BoolVar(&debugLevel, "debug", false, "Debug level. Default false.")

	flag.Parse()

	if len(ghApiKey) == 0 {
		log.Fatal(no_args_err_msg)
	}

	var logLevel = slog.LevelError
	if debugLevel {
		logLevel = slog.LevelDebug
	}

	slog.SetLogLoggerLevel(logLevel)

	choices := []list.Item{
		item("Next"),
		item("Open in browser"),
		item("Archive"),
		item("Unarchive"),
		item("Make it public"),
		item("Make it private"),
		item("Delete"),
		item("Previous"),
	}

	const defaultWidth = 20

	l := list.New(choices, itemDelegate{}, defaultWidth, listHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	facade := gh.NewGHAPIFacade(gh.WithGHAPIKey(ghApiKey), gh.WithGHOwner(ghOwner))
	p := tea.NewProgram(model{ghFacade: facade, spinner: s, loading: true, list: l, repos: nil, currentRepoIndex: 0})
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
