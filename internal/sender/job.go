package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"runtime"
	"runtime/debug"
	"text/template"
	"time"

	"git.server.lan/pkg/config/realtimeconfig"
	"github.com/google/uuid"
	"github.com/psevdocoder/gentleman-ping-bot/internal/config"
)

const (
	SendMessageJobName = "SendMessage"
)

const (
	messageKind        = 0
	skipInviteMentions = false
)

type Body struct {
	ChatID  int64   `json:"chat_id"`
	Message Message `json:"message"`
}

type Message struct {
	UUID               uuid.UUID `json:"uuid"`
	Text               string    `json:"text"`
	Markup             []any     `json:"markup"`
	Kind               int       `json:"kind"`
	Files              []any     `json:"files"`
	SkipInviteMentions bool      `json:"skip_invite_mentions"`
}

type parser interface {
	GetHeaders() (map[string]string, error)
	GetRequestURL() (string, error)
	GetCookie() (string, error)
}

type apiClient interface {
	SendMessage(ctx context.Context, requestURL string, cookie string, headers map[string]string, messageBody *Body) error
}

type SendMessageJob struct {
	parser      parser
	client      apiClient
	messageText string
	markup      []any
	chatID      int64
	sendEnabled bool
}

func NewSendMessageJob(parser parser, client apiClient) (*SendMessageJob, error) {

	// --- Message text ---
	messageRaw, err := config.GetValue(config.MessageText)
	if err != nil {
		return nil, err
	}
	messageStrRaw, err := messageRaw.String()
	if err != nil {
		return nil, err
	}

	messageStr, err := renderTemplate(messageStrRaw)
	if err != nil {
		return nil, err
	}

	var markup []any
	markupRaw, err := config.GetValue(config.Markup)
	if err == nil {
		markupStr, err := markupRaw.String()
		if err == nil && markupStr != "" {
			if err := json.Unmarshal([]byte(markupStr), &markup); err != nil {
				return nil, err
			}
		}
	}

	chatIDRaw, err := config.GetValue(config.ChatId)
	if err != nil {
		return nil, err
	}
	chatID, err := chatIDRaw.Int64()
	if err != nil {
		return nil, err
	}

	sendEnabledRaw, err := config.GetValue(config.SendEnabled)
	if err != nil {
		return nil, err
	}
	sendEnabled, err := sendEnabledRaw.Bool()
	if err != nil {
		return nil, err
	}

	job := &SendMessageJob{
		parser:      parser,
		client:      client,
		messageText: messageStr,
		markup:      markup,
		chatID:      chatID,
		sendEnabled: sendEnabled,
	}

	config.Watch(config.MessageText, func(newValue, oldValue realtimeconfig.Value) {
		newMessageText, err := newValue.String()
		if err != nil {
			log.Println("Failed to parse new message in live config:", err)
			return
		}

		rendered, err := renderTemplate(newMessageText)
		if err != nil {
			log.Println("Failed to render template:", err)
			return
		}

		job.messageText = rendered
		log.Println("Applied new message text")
	})

	config.Watch(config.Markup, func(newValue, oldValue realtimeconfig.Value) {
		newMarkupStr, err := newValue.String()
		if err != nil {
			log.Println("Failed to parse new markup:", err)
			return
		}

		var newMarkup []any
		if newMarkupStr != "" {
			if err := json.Unmarshal([]byte(newMarkupStr), &newMarkup); err != nil {
				log.Println("Failed to unmarshal markup:", err)
				return
			}
		}

		job.markup = newMarkup
		log.Println("Applied new markup config")
	})

	config.Watch(config.ChatId, func(newValue, oldValue realtimeconfig.Value) {
		newChatID, err := newValue.Int64()
		if err != nil {
			log.Println("Failed to parse new chatID in live config:", err)
			return
		}

		job.chatID = newChatID
		log.Printf("Applied new message chatID to %d", newChatID)
	})

	config.Watch(config.SendEnabled, func(newValue, oldValue realtimeconfig.Value) {
		newSendEnabled, err := newValue.Bool()
		if err != nil {
			log.Println("Failed to parse new send enabled in live config:", err)
			return
		}

		job.sendEnabled = newSendEnabled
		log.Printf("Applied new send enabled config to %t", newSendEnabled)
	})

	return job, nil
}

func (p *SendMessageJob) Name() string {
	return SendMessageJobName
}

func (p *SendMessageJob) Work(ctx context.Context) error {
	if !p.sendEnabled {
		log.Println("SendMessageJob is disabled")
		return nil
	}

	log.Println("Starting sending message...")

	requestURL, err := p.parser.GetRequestURL()
	if err != nil {
		return err
	}

	cookie, err := p.parser.GetCookie()
	if err != nil {
		return err
	}

	headers, err := p.parser.GetHeaders()
	if err != nil {
		return err
	}

	message := Message{
		UUID:               uuid.New(),
		Text:               p.messageText,
		Markup:             p.markup,
		Kind:               messageKind,
		Files:              []any{},
		SkipInviteMentions: skipInviteMentions,
	}

	body := &Body{
		ChatID:  p.chatID,
		Message: message,
	}

	if err := p.client.SendMessage(ctx, requestURL, cookie, headers, body); err != nil {
		return err
	}

	return nil
}

func renderTemplate(input string) (string, error) {
	funcMap := template.FuncMap{
		"NOW": func() string {
			return time.Now().Format(time.RFC3339)
		},

		// It's not a bug, it's a feature :) Поставь звезду, че зря старался ради всех нас?)
		"DEBUG": func() string {
			info, ok := debug.ReadBuildInfo()

			data := map[string]any{
				"go_version": runtime.Version(),
				"os":         runtime.GOOS,
				"arch":       runtime.GOARCH,
			}

			if ok {
				data["module_path"] = info.Main.Path
			}

			b, _ := json.Marshal(data)
			return string(b)
		},
	}

	t, err := template.New("").Funcs(funcMap).Parse(input)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, nil); err != nil {
		return "", err
	}

	return buf.String(), nil
}
