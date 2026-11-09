package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Bot API 10.1 sendRichMessage renders raw markdown natively (pipe tables included).
// telego v1.10 does not know the method, so it is called over plain HTTP.

const richMessageMaxChars = 32768

var tableDividerRe = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$`)

// hasPipeTable reports whether content contains a GFM table (header + divider row).
func hasPipeTable(content string) bool {
	lines := strings.Split(content, "\n")
	for i := 1; i < len(lines); i++ {
		if tableDividerRe.MatchString(lines[i]) && strings.Contains(lines[i-1], "|") {
			return true
		}
	}
	return false
}

type richSendResult struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
	Result      struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
}

// sendRich sends content via sendRichMessage. Any error means "fall back to the legacy path".
func (c *TelegramChannel) sendRich(
	ctx context.Context,
	chatID int64,
	threadID int,
	replyToID string,
	content string,
) (string, error) {
	payload := map[string]any{
		"chat_id":      chatID,
		"rich_message": map[string]any{"markdown": content},
	}
	if threadID != 0 {
		payload["message_thread_id"] = threadID
	}
	if mid, err := strconv.Atoi(replyToID); err == nil && replyToID != "" {
		// sendRichMessage ignores reply_to_message_id; only reply_parameters works.
		payload["reply_parameters"] = map[string]int{"message_id": mid}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	url := c.richAPIServer + "/bot" + c.bot.Token() + "/sendRichMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.richHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("sendRichMessage: request failed") // no %w: the URL carries the token
	}
	defer resp.Body.Close()

	var res richSendResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("sendRichMessage: decode: %w", err)
	}
	if !res.OK {
		if res.ErrorCode == http.StatusNotFound {
			c.richUnsupported.Store(true) // server without Bot API 10.1: stop trying
		}
		return "", fmt.Errorf("sendRichMessage: %d %s", res.ErrorCode, res.Description)
	}
	return strconv.Itoa(res.Result.MessageID), nil
}

func (c *TelegramChannel) wantsRich(content string) bool {
	return c.tgCfg.RichMessages && !c.richUnsupported.Load() &&
		len([]rune(content)) <= richMessageMaxChars && hasPipeTable(content)
}
