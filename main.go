package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	list       list.Model
	current    *Item
	audiolinks []string
	progress   int
	err        error
}

type updateCatalog int
type checkItem []string
type downloadAudio int

type errMsg struct{ err error }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case updateCatalog:
		items := FindProductsFromService()
		// TODO there has to be a better way for this
		listItems := make([]list.Item, 0)
		for i := range items {
			listItems = append(listItems, items[i])
		}
		m.list.SetItems(listItems)
		log.Printf("got Catalog")
		return m, nil

	case checkItem:
		m.audiolinks = msg
		return m, nil
	case downloadAudio:
		m.progress = 100
		return m, tea.Quit
	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.current == nil {
				i, ok := m.list.SelectedItem().(Item)
				if ok {
					m.current = &i
				}
				return m, func() tea.Msg {
					links, err := FindAudioLink(m.current.detailLink)
					if err != nil {
						return errMsg{err}
					}
					return checkItem(links)
				}
			} else if len(m.audiolinks) > 0 && m.progress == 0 {
				m.progress = 1
				return m, func() tea.Msg {
					home, err := os.UserHomeDir()
					if err != nil {
						return errMsg{err}
					}
					err = DownloadFile(m.audiolinks[0], home+"/Downloads")
					if err != nil {
						return errMsg{err}
					}
					return downloadAudio(0)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	s := ""
	style := m.list.Styles.FilterPrompt.Render

	if len(m.list.Items()) == 0 {
		s += style(fmt.Sprint("\n  Loading catalog... q to quit\n\n"))
		return s
	}

	s += style(fmt.Sprintf("\nhave %d items\n\n", len(m.list.Items())))

	if m.current == nil {
		s += "\n" + m.list.View()
		return s
	}
	s += style(fmt.Sprintf("\ncurrent: %s\n\n", m.current.name))

	if len(m.audiolinks) == 0 {
		s += style(fmt.Sprintf("\nno audio links\n\n"))
	} else {
		s += style(fmt.Sprintf("\naudio links: %d - Press Enter to download to ~/Downloads\n\n", len(m.audiolinks)))
	}

	s += style(fmt.Sprintf("\nprogress: %d\n\n", m.progress))
	if m.err != nil {
		s += m.list.Styles.StatusBarActiveFilter.Render(fmt.Sprintf("\nerror: %s\n\n", m.err))
	}

	return s
}

func initialModel() model {
	listDelegate := list.NewDefaultDelegate()
	listDelegate.ShowDescription = false
	return model{list: list.New(make([]list.Item, 0), listDelegate, 50, 30)}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg { return updateCatalog(0) }
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}
