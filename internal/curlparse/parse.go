package curlparse

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	urlRE     = regexp.MustCompile(`curl\s+['"]([^'"]+)['"]`)
	headersRE = regexp.MustCompile(`-H\s+(?:'([^']*)'|"([^"]*)")`)
	cookieRE  = regexp.MustCompile(`(?:-b|--cookie)\s+(?:'([^']*)'|"([^"]*)")`)
)

type CurlRequest struct {
	Method  string
	URL     string
	Headers http.Header
	Body    Body
}

type Body struct {
	ChatID  int     `json:"chat_id"`
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

type Parser struct {
	curlCMD string
}

func NewParser(curl string) *Parser {
	// убираем переносы строк с "\"
	curl = strings.ReplaceAll(curl, "\\\n", " ")
	curl = strings.ReplaceAll(curl, "\n", " ")

	return &Parser{curlCMD: curl}
}

func (p *Parser) GetRequestURL() (string, error) {
	m := urlRE.FindStringSubmatch(p.curlCMD)
	if len(m) < 2 {
		return "", fmt.Errorf("url not found in curl command")
	}
	return m[1], nil
}

func (p *Parser) GetHeaders() (map[string]string, error) {
	matches := headersRE.FindAllStringSubmatch(p.curlCMD, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("headers not found in curl command")
	}

	headers := make(map[string]string)

	for _, m := range matches {
		var headerLine string
		if m[1] != "" {
			headerLine = m[1]
		} else {
			headerLine = m[2]
		}

		parts := strings.SplitN(headerLine, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		headers[key] = value
	}

	return headers, nil
}

func (p *Parser) GetCookie() (string, error) {
	m := cookieRE.FindStringSubmatch(p.curlCMD)
	if len(m) == 0 {
		return "", fmt.Errorf("cookie not found in curl command")
	}

	if m[1] != "" {
		return m[1], nil
	}
	if m[2] != "" {
		return m[2], nil
	}

	return "", fmt.Errorf("cookie not found")
}
