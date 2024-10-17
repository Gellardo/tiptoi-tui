package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	list        list.Model
	catalog     int
	current     *Item
	audiolink   *Item
	destination string
	progress    int
	err         error
}

type updateCatalog []list.Item
type checkItem []list.Item
type selectedDestination []list.Item
type downloadAudio int

type errMsg struct{ err error }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case updateCatalog:
		items := msg
		m.list.SetItems(items)
		m.list.ResetSelected()
		m.list.ResetFilter()
		m.catalog = len(items)
		log.Printf("got Catalog")
		return m, nil

	case checkItem:
		if len(msg) > 1 {
			m.list.SetItems(msg)
			m.list.ResetSelected()
			m.list.ResetFilter()
		} else {
			link, ok := msg[0].(Item)
			if !ok {
				return m, func() tea.Msg { return errMsg{errors.New("Could not convert selected link to Item")} }
			}
			m.audiolink = &link
			return m, m.triggerSelectDestination()
		}
		return m, nil
	case selectedDestination:
		if len(msg) > 1 {
			m.list.SetItems(msg)
			m.list.ResetSelected()
			m.list.ResetFilter()
		} else {
			link, ok := msg[0].(Item)
			if !ok {
				return m, func() tea.Msg { return errMsg{errors.New("Could not convert destination to Item")} }
			}
			m.destination = link.detailLink
		}
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
			// all information filled out
			if m.audiolink != nil && len(m.destination) > 0 && m.progress == 0 {
				m.progress = 1
				return m, func() tea.Msg {
					err := DownloadFile(m.audiolink.detailLink, m.destination)
					if err != nil {
						// TODO stop / retry?
						return errMsg{err}
					}
					return downloadAudio(0)
				}
			}

			// some list view is open
			i, ok := m.list.SelectedItem().(Item)
			if !ok {
				return m, func() tea.Msg { return errMsg{errors.New("list selection failed?")} }
			}
			if m.current == nil {
				m.current = &i
				m.list.SetItems(nil)
				return m, m.triggerFindAudioLinks()
			} else if m.audiolink == nil {
				m.audiolink = &i
				m.list.SetItems(nil)
				return m, m.triggerSelectDestination()
			} else if m.destination == "" {
				m.destination = i.detailLink
				m.list.SetItems(nil)
				return m, nil
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

	if m.audiolink != nil {
		currentState = append(currentState, filledBox(fmt.Sprintf("DownloadLink: %s", m.audiolink.name)))
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
		currentState = append(currentState, fmt.Sprint("\n  Loading catalog... Esc to quit\n\n"))
	}

	if m.audiolink != nil && len(m.destination) > 0 && m.progress == 0 {
		currentState = append(currentState, lipgloss.NewStyle().Width(60).Height(5).Foreground(lipgloss.Color("#ee2200")).Align(lipgloss.Center, lipgloss.Center).Render("everything selected, press Enter to download"))
	}

	if len(m.list.Items()) > 0 {
		currentState = append(currentState, m.list.View())
	}

	s := ""
	s += lipgloss.JoinVertical(
		lipgloss.Top,
		currentState...,
	)

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
	return func() tea.Msg {
		items := FindProductsFromService()

		var listItems []list.Item
		for _, item := range items {
			listItems = append(listItems, item)
		}
		return updateCatalog(listItems)
	}
}

func (m model) triggerFindAudioLinks() tea.Cmd {
	return func() tea.Msg {
		items, err := FindAudioLink(m.current.detailLink)
		if err != nil {
			return errMsg{err}
		}

		var listItems []list.Item
		for _, item := range items {
			listItems = append(listItems, item)
		}
		return checkItem(listItems)
	}
}

func (m model) triggerSelectDestination() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return errMsg{err}
		}
		destination := home + "/Downloads"

		paths := possibleDownloadLocations()

		var listItems []list.Item
		for _, item := range paths {
			listItems = append(listItems, item)
		}
		listItems = append(listItems, Item{name: "Downloads", detailLink: destination})
		return selectedDestination(listItems)
	}
}

func possibleDownloadLocations() []Item {
	cmd := exec.Command("mount")
	out, err := cmd.Output()

	var list []Item
	if err != nil {
		return list
	}

	for _, line := range strings.Split(string(out), "\n") {
		if subline := strings.Split(line, " "); len(subline) >= 3 &&
			(strings.HasPrefix(subline[2], "/Volumes/") ||
				strings.HasPrefix(subline[2], "/media/") ||
				strings.HasPrefix(subline[2], "/run/media/")) {
			path := strings.Split(subline[2], "/")
			list = append(list, Item{name: path[len(path)-1], detailLink: subline[2]})
		}
	}
	return list
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}
