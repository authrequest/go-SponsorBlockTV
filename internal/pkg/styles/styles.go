package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Color scheme
	primaryColor   = lipgloss.Color("#3B3B3B")
	secondaryColor = lipgloss.Color("#1F1F1F")
	accentColor    = lipgloss.Color("#0D99FF")
	textColor      = lipgloss.Color("#FFFFFF")
	errorColor     = lipgloss.Color("#FF0000")

	// Base styles
	Container = lipgloss.NewStyle().
			Padding(1).
			MarginTop(1).
			Width(100).
			Background(secondaryColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(primaryColor)

	Title = lipgloss.NewStyle().
		Bold(true).
		Width(100).
		Padding(0, 1).
		Background(primaryColor).
		Foreground(textColor)

	Subtitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#999999")).
			MarginLeft(1)

	// Tab styles
	TabActive = lipgloss.NewStyle().
			Background(accentColor).
			Foreground(textColor).
			Padding(0, 2)

	TabInactive = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(textColor).
			Padding(0, 2)

	// Selection list styles
	SelectionList = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			MarginTop(1)

	SelectionItem = lipgloss.NewStyle().
			PaddingLeft(1)

	SelectionItemActive = SelectionItem.Copy().
				Foreground(accentColor).
				Bold(true)

	// Checkbox styles
	Checkbox = lipgloss.NewStyle().
			PaddingLeft(1).
			MarginTop(1)

	CheckboxChecked = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// Input styles
	Input = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(0, 1).
		MarginTop(1)

	// Button styles
	Button = lipgloss.NewStyle().
		Background(accentColor).
		Foreground(textColor).
		Padding(0, 2).
		Align(lipgloss.Center).
		MarginTop(1)

	ButtonDanger = Button.Copy().
			Background(errorColor)

	// Footer styles
	Footer = lipgloss.NewStyle().
		Background(primaryColor).
		Foreground(textColor).
		Padding(0, 1).
		Align(lipgloss.Right)

	// Button styles
	ButtonSmall = lipgloss.NewStyle().
			Height(3).
			Padding(0)

	Button100 = lipgloss.NewStyle().
			Width(100)

	// Dialog styles
	Dialog = lipgloss.NewStyle().
		Padding(1, 2).
		Width(35).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8"))

	// Device editor styles
	EditDeviceContainer = lipgloss.NewStyle().
				Padding(1, 2, 0, 2).
				Width(50).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("8"))

	// List styles
	DevicesManager = lipgloss.NewStyle().
			Width(100).
			MaxHeight(70)

	Element = lipgloss.NewStyle().
		Width(100).
		Padding(0, 1, 0, 1)

	ElementName = lipgloss.NewStyle().
			Width(100).
			Align(lipgloss.Left, lipgloss.Center)

	// API Key styles
	APIKeyGrid = lipgloss.NewStyle().
			Padding(1, 1).
			Width(100).
			Height(5)

	// Skip Categories styles
	SkipCategoriesManager = lipgloss.NewStyle().
				Width(100).
				MaxHeight(70)

	// Channel Whitelist styles
	ChannelWhitelistManager = lipgloss.NewStyle().
				Width(100).
				MaxHeight(70)

	// Ad Skip/Mute styles
	AdSkipMuteContainer = lipgloss.NewStyle().
				Padding(1)

	// Autoplay styles
	AutoplayContainer = lipgloss.NewStyle().
				Padding(1)
)
