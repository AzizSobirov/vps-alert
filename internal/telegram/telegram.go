// Package telegram is a minimal Bot API client covering only what vps-alert needs.
// It is hand written to keep the dependency tree of a root-level agent small.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	token string
	base  string
	http  *http.Client
}

func New(token string) *Client {
	return NewWithURL(token, "https://api.telegram.org")
}

// NewWithURL uses another Bot API server, e.g. a self-hosted one.
func NewWithURL(token, apiURL string) *Client {
	return &Client{
		token: token,
		base:  strings.TrimSuffix(apiURL, "/") + "/bot" + token + "/",
		http:  &http.Client{Timeout: 70 * time.Second},
	}
}

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text"`

	ReplyToMessage *Message `json:"reply_to_message"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type ChatMemberUpdated struct {
	Chat          Chat `json:"chat"`
	From          User `json:"from"`
	NewChatMember struct {
		Status string `json:"status"` // member, administrator, left, kicked, ...
	} `json:"new_chat_member"`
}

type Update struct {
	UpdateID      int64              `json:"update_id"`
	Message       *Message           `json:"message"`
	CallbackQuery *CallbackQuery     `json:"callback_query"`
	MyChatMember  *ChatMemberUpdated `json:"my_chat_member"`
}

type Button struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type Keyboard struct {
	InlineKeyboard [][]Button `json:"inline_keyboard"`
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func (c *Client) call(ctx context.Context, method string, params, out any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+method, bytes.NewReader(body))
	if err != nil {
		return c.scrub(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return c.scrub(err)
	}
	defer resp.Body.Close()

	var r apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("telegram %s: decode response (HTTP %d): %w", method, resp.StatusCode, err)
	}
	if !r.OK {
		if r.Parameters.RetryAfter > 0 {
			return &RetryError{After: time.Duration(r.Parameters.RetryAfter) * time.Second}
		}
		return fmt.Errorf("telegram %s: %s", method, r.Description)
	}
	if out != nil {
		return json.Unmarshal(r.Result, out)
	}
	return nil
}

// scrub removes the bot token from errors, since net/http embeds the full
// request URL in them and they end up in logs.
func (c *Client) scrub(err error) error {
	return errors.New(strings.ReplaceAll(err.Error(), c.token, "<token>"))
}

type RetryError struct{ After time.Duration }

func (e *RetryError) Error() string {
	return fmt.Sprintf("telegram rate limit, retry after %s", e.After)
}

func (c *Client) GetMe(ctx context.Context) (*User, error) {
	var u User
	err := c.call(ctx, "getMe", map[string]any{}, &u)
	return &u, err
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSec int) ([]Update, error) {
	var ups []Update
	err := c.call(ctx, "getUpdates", map[string]any{
		"offset":          offset,
		"timeout":         timeoutSec,
		"allowed_updates": []string{"message", "callback_query", "my_chat_member"},
	}, &ups)
	return ups, err
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, kb *Keyboard) (*Message, error) {
	params := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}
	if kb != nil {
		params["reply_markup"] = kb
	}
	var m Message
	err := c.call(ctx, "sendMessage", params, &m)
	return &m, err
}

// AskReply sends text with a force-reply markup, so the user's answer comes
// back as a reply to it. Replies to the bot's own messages reach it even in
// groups with privacy mode on.
func (c *Client) AskReply(ctx context.Context, chatID int64, text string) (*Message, error) {
	var m Message
	err := c.call(ctx, "sendMessage", map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "HTML",
		"reply_markup": map[string]any{"force_reply": true, "selective": true},
	}, &m)
	return &m, err
}

func (c *Client) EditMessage(ctx context.Context, chatID, messageID int64, text string, kb *Keyboard) error {
	params := map[string]any{
		"chat_id":                  chatID,
		"message_id":               messageID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}
	if kb != nil {
		params["reply_markup"] = kb
	}
	return c.call(ctx, "editMessageText", params, nil)
}

func (c *Client) AnswerCallback(ctx context.Context, id, text string) error {
	return c.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": id, "text": text}, nil)
}

func (c *Client) LeaveChat(ctx context.Context, chatID int64) error {
	return c.call(ctx, "leaveChat", map[string]any{"chat_id": chatID}, nil)
}

func (c *Client) SetMyCommands(ctx context.Context, cmds []BotCommand) error {
	return c.call(ctx, "setMyCommands", map[string]any{"commands": cmds}, nil)
}

// Notifier sends messages to the single allowed chat.
type Notifier struct {
	TG     *Client
	ChatID int64
}

// Send delivers text to the allowed chat, retrying briefly on rate limits and
// network errors. Failures are logged, never returned: a lost notification
// must not stop a backup or the monitor loop.
func (n *Notifier) Send(ctx context.Context, text string) {
	for attempt := 0; attempt < 3; attempt++ {
		_, err := n.TG.SendMessage(ctx, n.ChatID, text, nil)
		if err == nil {
			return
		}
		wait := time.Duration(attempt+1) * 3 * time.Second
		var re *RetryError
		if errors.As(err, &re) {
			wait = re.After
		}
		log.Printf("notify: %v", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}
