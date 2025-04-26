package setup

import (
	"strings"

	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/config"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main application state
type Model struct {
	config     *config.Config
	currentTab int
	tabs       []string
	width      int
	height     int
	// Selection states
	skipCategories    map[string]bool
	skipCountTracking bool
	muteAds           bool
	skipAds           bool
	autoplay          bool
}

// InitialModel creates a new model with default values
func InitialModel(cfg *config.Config) Model {
	skipCats := make(map[string]bool)
	for _, cat := range cfg.SkipCategories {
		skipCats[cat] = true
	}

	return Model{
		config:     cfg,
		currentTab: 0,
		tabs: []string{
			"Devices",
			"Skip Categories",
			"Skip Count Tracking",
			"Skip/Mute ads",
			"Channel Whitelist",
			"YouTube API Key",
			"Autoplay",
		},
		skipCategories:    skipCats,
		skipCountTracking: cfg.SkipCountTracking,
		muteAds:           cfg.MuteAds,
		skipAds:           cfg.SkipAds,
		autoplay:          cfg.AutoPlay,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "right", "l":
			m.currentTab = (m.currentTab + 1) % len(m.tabs)
		case "shift+tab", "left", "h":
			m.currentTab = (m.currentTab - 1 + len(m.tabs)) % len(m.tabs)
		case "s":
			m.saveConfig()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *Model) saveConfig() {
	m.config.SkipCategories = make([]string, 0)
	for cat, selected := range m.skipCategories {
		if selected {
			m.config.SkipCategories = append(m.config.SkipCategories, cat)
		}
	}
	m.config.SkipCountTracking = m.skipCountTracking
	m.config.MuteAds = m.muteAds
	m.config.SkipAds = m.skipAds
	m.config.AutoPlay = m.autoplay
}

// View renders the UI
func (m Model) View() string {
	doc := strings.Builder{}

	// Header
	doc.WriteString(styles.Title.Render("iSponsorBlockTV - Setup Wizard") + "\n\n")

	// Tabs
	tabs := make([]string, len(m.tabs))
	for i, tab := range m.tabs {
		if i == m.currentTab {
			tabs[i] = styles.TabActive.Render(tab)
		} else {
			tabs[i] = styles.TabInactive.Render(tab)
		}
	}
	doc.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")

	// Content
	content := m.renderCurrentTab()
	doc.WriteString(styles.Container.Render(content) + "\n")

	// Footer
	doc.WriteString(styles.Footer.Render("q: Exit  s: Save"))

	return doc.String()
}

func (m Model) renderCurrentTab() string {
	switch m.currentTab {
	case 0:
		return m.renderDevicesTab()
	case 1:
		return m.renderSkipCategoriesTab()
	case 2:
		return m.renderSkipCountTrackingTab()
	case 3:
		return m.renderAdSkipMuteTab()
	case 4:
		return m.renderChannelWhitelistTab()
	case 5:
		return m.renderAPIKeyTab()
	case 6:
		return m.renderAutoplayTab()
	default:
		return ""
	}
}

func (m Model) renderDevicesTab() string {
	var s strings.Builder
	s.WriteString(styles.Title.Render("Devices") + "\n")
	s.WriteString(styles.Button.Render("Add Device") + "\n\n")

	if len(m.config.Devices) == 0 {
		s.WriteString(styles.Subtitle.Render("No devices added"))
	} else {
		for _, device := range m.config.Devices {
			name := device.Name
			if name == "" {
				name = device.ScreenID
			}
			s.WriteString(styles.SelectionItem.Render(name) + "\n")
		}
	}
	return s.String()
}

func (m Model) renderSkipCategoriesTab() string {
	var s strings.Builder
	s.WriteString(styles.Title.Render("Skip Categories") + "\n")
	s.WriteString(styles.Subtitle.Render("Select the categories you want to skip") + "\n\n")

	categories := []string{
		"Sponsor",
		"Self-Promotion",
		"Intro",
		"Outro",
		"Interaction",
		"Music Offtopic",
		"Preview",
		"Filler",
	}

	for _, cat := range categories {
		checked := " "
		if m.skipCategories[cat] {
			checked = "x"
		}
		s.WriteString(styles.SelectionItem.Render(
			lipgloss.JoinHorizontal(lipgloss.Left, "["+checked+"]", " "+cat),
		) + "\n")
	}
	return s.String()
}

func (m Model) renderSkipCountTrackingTab() string {
	var s strings.Builder
	s.WriteString(styles.Title.Render("Skip Count Tracking") + "\n")
	s.WriteString(styles.Subtitle.Render(
		"This feature tracks which segments you have skipped to let users know how much their submission has helped others",
	) + "\n\n")

	checked := " "
	if m.skipCountTracking {
		checked = "x"
	}
	s.WriteString(styles.Checkbox.Render(
		lipgloss.JoinHorizontal(lipgloss.Left, "["+checked+"]", " Enable skip count tracking"),
	))
	return s.String()
}

func (m Model) renderAdSkipMuteTab() string {
	var s strings.Builder
	s.WriteString(styles.Title.Render("Skip/Mute ads") + "\n")
	s.WriteString(styles.Subtitle.Render(
		"This feature allows you to automatically mute and/or skip native YouTube ads",
	) + "\n\n")

	skipChecked := " "
	if m.skipAds {
		skipChecked = "x"
	}
	muteChecked := " "
	if m.muteAds {
		muteChecked = "x"
	}

	s.WriteString(styles.Checkbox.Render(
		lipgloss.JoinHorizontal(lipgloss.Left, "["+skipChecked+"]", " Enable skipping ads"),
	) + "\n")
	s.WriteString(styles.Checkbox.Render(
		lipgloss.JoinHorizontal(lipgloss.Left, "["+muteChecked+"]", " Enable muting ads"),
	))
	return s.String()
}

func (m Model) renderChannelWhitelistTab() string {
	var s strings.Builder
	s.WriteString(styles.Title.Render("Channel Whitelist") + "\n")
	s.WriteString(styles.Button.Render("Add Channel") + "\n\n")

	if len(m.config.ChannelWhitelist) == 0 {
		s.WriteString(styles.Subtitle.Render("No channels whitelisted"))
	} else {
		for _, channel := range m.config.ChannelWhitelist {
			s.WriteString(styles.SelectionItem.Render(channel.ID) + "\n")
		}
	}
	return s.String()
}

func (m Model) renderAPIKeyTab() string {
	var s strings.Builder
	s.WriteString(styles.Title.Render("YouTube API Key") + "\n")
	s.WriteString(styles.Subtitle.Render(
		"You can get a YouTube Data API v3 Key from the Google Cloud Console",
	) + "\n\n")

	key := m.config.APIKey
	if key == "" {
		key = "Enter your API key"
	}
	s.WriteString(styles.Input.Render(key))
	return s.String()
}

func (m Model) renderAutoplayTab() string {
	var s strings.Builder
	s.WriteString(styles.Title.Render("Autoplay") + "\n")
	s.WriteString(styles.Subtitle.Render(
		"This feature allows you to enable/disable autoplay",
	) + "\n\n")

	checked := " "
	if m.autoplay {
		checked = "x"
	}
	s.WriteString(styles.Checkbox.Render(
		lipgloss.JoinHorizontal(lipgloss.Left, "["+checked+"]", " Enable autoplay"),
	))
	return s.String()
}
