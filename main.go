package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	list        list.Model
	catalog     int
	current     *Item
	audiolinks  []string
	destination string
	progress    int
	err         error
}

type updateCatalog []list.Item
type checkItem []string
type downloadAudio int

type errMsg struct{ err error }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case updateCatalog:
		items := msg
		m.list.SetItems(items)
		m.catalog = len(items)
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
		case "ctrl+c", "Esc":
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
					err := DownloadFile(m.audiolinks[0], m.destination)
					if err != nil {
						return errMsg{err}
					}
					return downloadAudio(0)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height / 2)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	filledBox := lipgloss.
		NewStyle().Width(m.list.Width() - 5).
		Foreground(lipgloss.Color("#04B575")).
		Border(lipgloss.DoubleBorder()).
		Render
	unfilledBox := lipgloss.
		NewStyle().
		Width(m.list.Width() - 5).
		Border(lipgloss.NormalBorder()).
		Render

	currentState := make([]string, 0)

	if m.catalog > 0 {
		currentState = append(currentState, filledBox(fmt.Sprintf("Found %d products", m.catalog)))
	} else {
		currentState = append(currentState, unfilledBox("Loading products"))
	}

	if m.current != nil {
		currentState = append(currentState, filledBox(fmt.Sprintf("Product: %s", m.current.name)))
	} else {
		currentState = append(currentState, unfilledBox("Product:"))
	}

	if len(m.audiolinks) > 0 {
		currentState = append(currentState, filledBox(fmt.Sprintf("DownloadLink: %s", m.audiolinks[0])))
	} else {
		currentState = append(currentState, unfilledBox("DownloadLink:"))
	}

	if len(m.destination) > 0 {
		currentState = append(currentState, filledBox(fmt.Sprintf("Destination: %s", m.destination)))
	} else {
		currentState = append(currentState, unfilledBox("Destination:"))
	}

	if m.progress > 0 {
		currentState = append(currentState, filledBox(fmt.Sprintf("Progress: %d / 100", m.progress)))
	} else {
		currentState = append(currentState, unfilledBox("Progress:"))
	}

	if m.catalog == 0 {
		currentState = append(currentState,fmt.Sprint("\n  Loading catalog... Esc to quit\n\n")))
	}

	if len(m.audiolinks) > 0 && len(m.destination) > 0 {
		currentState = append(currentState, "everything selected, press Enter to download")
	}

	s := ""
	s += lipgloss.JoinVertical(
		lipgloss.Top,
		currentState...,
		m.list.View()
	)

	if m.err != nil {
		s += m.list.Styles.StatusBarActiveFilter.Render(fmt.Sprintf("\nerror: %s\n\n", m.err))
	}

	return s
}

func initialModel() model {
	listDelegate := list.NewDefaultDelegate()
	listDelegate.ShowDescription = false
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	destination := home + "/Downloads"
	return model{list: list.New(make([]list.Item, 0), listDelegate, 50, 30), destination: destination}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		items := FindProductsFromService()

		// TODO there has to be a better way for this
		listItems := make([]list.Item, 0)
		for i := range items {
			listItems = append(listItems, items[i])
		}
		return updateCatalog(listItems)
	}
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}
