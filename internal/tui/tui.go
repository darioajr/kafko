// Package tui implements the interactive bubbletea UI behind `kafko tui`.
//
// Two scenes:
//
//	topicsScene   — list of topics, [enter] to start consuming
//	messagesScene — live tail of a topic, [/] to filter, [esc] to go back
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/darioajr/kafko/internal/kafka"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type scene int

const (
	sceneTopics scene = iota
	sceneMessages
)

type topicItem struct {
	name       string
	partitions int
}

func (t topicItem) Title() string       { return t.name }
func (t topicItem) Description() string { return fmt.Sprintf("%d partitions", t.partitions) }
func (t topicItem) FilterValue() string { return t.name }

type recordMsg struct{ rec *kgo.Record }
type errMsg struct{ err error }
type topicsLoadedMsg struct{ items []list.Item }

type model struct {
	clientOpts kafka.ClientOptions

	scene scene
	width int

	topics  list.Model
	filter  textinput.Model
	stream  viewport.Model
	records []*kgo.Record

	consumer       *kgo.Client
	consumerCancel context.CancelFunc
	recordCh       chan kafka.Message
	currentTopic   string
	filterActive   bool
	height         int

	err error
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	footerStyle = lipgloss.NewStyle().Faint(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	keyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	metaStyle   = lipgloss.NewStyle().Faint(true)
)

// Run launches the TUI and blocks until the user exits.
func Run(clientOpts kafka.ClientOptions) error {
	ti := textinput.New()
	ti.Placeholder = "filter substring (enter to apply, esc to clear)"
	ti.Prompt = "/ "

	l := list.New(nil, list.NewDefaultDelegate(), 40, 20)
	l.Title = "kafko — topics"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	m := model{
		clientOpts: clientOpts,
		scene:      sceneTopics,
		topics:     l,
		filter:     ti,
		stream:     viewport.New(80, 20),
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return loadTopicsCmd(m.clientOpts)
}

func loadTopicsCmd(opts kafka.ClientOptions) tea.Cmd {
	return func() tea.Msg {
		c, err := kafka.NewClient(opts)
		if err != nil {
			return errMsg{err}
		}
		defer c.Close()
		adm := kadm.NewClient(c)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		td, err := adm.ListTopics(ctx)
		if err != nil {
			return errMsg{err}
		}
		names := make([]string, 0, len(td))
		for n := range td {
			names = append(names, n)
		}
		sort.Strings(names)
		items := make([]list.Item, 0, len(names))
		for _, n := range names {
			items = append(items, topicItem{name: n, partitions: len(td[n].Partitions)})
		}
		return topicsLoadedMsg{items: items}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.topics.SetSize(msg.Width-2, msg.Height-4)
		m.stream.Width = msg.Width - 2
		m.stream.Height = msg.Height - 6
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case topicsLoadedMsg:
		m.topics.SetItems(msg.items)
		return m, nil

	case recordMsg:
		m.records = append(m.records, msg.rec)
		// Keep last 1000 to bound memory.
		if len(m.records) > 1000 {
			m.records = m.records[len(m.records)-1000:]
		}
		m.refreshStream()
		return m, awaitRecordCmd(m.recordCh)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	switch m.scene {
	case sceneTopics:
		var cmd tea.Cmd
		m.topics, cmd = m.topics.Update(msg)
		return m, cmd
	case sceneMessages:
		if m.filterActive {
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.refreshStream()
			return m, cmd
		}
		var cmd tea.Cmd
		m.stream, cmd = m.stream.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.stopConsumer()
		return m, tea.Quit
	}

	switch m.scene {
	case sceneTopics:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			it, ok := m.topics.SelectedItem().(topicItem)
			if !ok {
				return m, nil
			}
			cmd, err := m.startConsumer(it.name)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.scene = sceneMessages
			m.records = m.records[:0]
			m.currentTopic = it.name
			m.refreshStream()
			return m, cmd
		}
		var cmd tea.Cmd
		m.topics, cmd = m.topics.Update(msg)
		return m, cmd

	case sceneMessages:
		if m.filterActive {
			switch msg.String() {
			case "esc":
				m.filterActive = false
				m.filter.SetValue("")
				m.refreshStream()
				return m, nil
			case "enter":
				m.filterActive = false
				m.refreshStream()
				return m, nil
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.refreshStream()
			return m, cmd
		}
		switch msg.String() {
		case "esc", "q":
			m.stopConsumer()
			m.scene = sceneTopics
			return m, nil
		case "/":
			m.filterActive = true
			m.filter.Focus()
			return m, textinput.Blink
		case "c":
			m.records = m.records[:0]
			m.refreshStream()
			return m, nil
		}
		var cmd tea.Cmd
		m.stream, cmd = m.stream.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) startConsumer(topic string) (tea.Cmd, error) {
	ctx, cancel := context.WithCancel(context.Background())
	c, err := kafka.NewConsumer(m.clientOpts, kafka.ConsumeOptions{
		Topics:    []string{topic},
		Partition: -1,
		Offset:    -1,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	ch := make(chan kafka.Message, 256)
	go kafka.PollLoop(ctx, c, ch)

	m.consumer = c
	m.consumerCancel = cancel
	m.recordCh = ch

	return awaitRecordCmd(ch), nil
}

func (m *model) stopConsumer() {
	if m.consumerCancel != nil {
		m.consumerCancel()
		m.consumerCancel = nil
	}
	if m.consumer != nil {
		m.consumer.Close()
		m.consumer = nil
	}
}

func awaitRecordCmd(ch <-chan kafka.Message) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		if msg.Err != nil {
			return errMsg{msg.Err}
		}
		return recordMsg{rec: msg.Record}
	}
}

func (m *model) refreshStream() {
	filter := m.filter.Value()
	var b strings.Builder
	for _, r := range m.records {
		val := string(r.Value)
		if filter != "" && !strings.Contains(val, filter) && !strings.Contains(string(r.Key), filter) {
			continue
		}
		meta := metaStyle.Render(fmt.Sprintf("[%s/%d@%d %s]",
			r.Topic, r.Partition, r.Offset, r.Timestamp.Format("15:04:05.000")))
		key := ""
		if len(r.Key) > 0 {
			key = keyStyle.Render(string(r.Key)) + " "
		}
		b.WriteString(meta)
		b.WriteString(" ")
		b.WriteString(key)
		b.WriteString(val)
		b.WriteString("\n")
	}
	m.stream.SetContent(b.String())
	m.stream.GotoBottom()
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n\npress ctrl+c to exit"
	}
	switch m.scene {
	case sceneTopics:
		return titleStyle.Render("kafko") + "\n" + m.topics.View() + "\n" +
			footerStyle.Render("[enter] tail topic   [/] filter   [q] quit")
	case sceneMessages:
		header := titleStyle.Render("kafko · "+m.currentTopic) +
			"  " + footerStyle.Render(fmt.Sprintf("(%d messages)", len(m.records)))
		body := m.stream.View()
		var footer string
		if m.filterActive {
			footer = m.filter.View()
		} else {
			f := m.filter.Value()
			suffix := ""
			if f != "" {
				suffix = "  filter=" + f
			}
			footer = footerStyle.Render("[esc] back   [/] filter   [c] clear   [q] quit" + suffix)
		}
		return header + "\n" + body + "\n" + footer
	}
	return ""
}
