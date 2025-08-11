package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/krau/remdit/config"
)

type Client struct {
	ctx        context.Context
	conn       *websocket.Conn
	l          *log.Logger
	serverConf config.Server
	editURL    string
	sessionID  string
	filePath   string
}

func NewClient(ctx context.Context, serverConf config.Server, filePath string) *Client {
	return &Client{
		ctx:        ctx,
		serverConf: serverConf,
		filePath:   filePath,
		l:          log.FromContext(ctx).WithPrefix("client"),
	}
}

func (c *Client) CreateSession() error {
	u, err := url.Parse(c.serverConf.Addr)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %w", err)
	}
	if !strings.HasPrefix(u.String(), "http") {
		u, err = url.Parse("http://" + u.String())
		c.serverConf.Addr = u.String()
		if err != nil {
			return fmt.Errorf("failed to parse server URL with http prefix: %w", err)
		}
	}
	u = u.JoinPath("api", "session")
	file, err := os.Open(c.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("document", filepath.Base(c.filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.serverConf.Key != "" {
		req.Header.Set("X-API-Key", c.serverConf.Key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("unauthorized: check your API key")
		}
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	c.sessionID = sessionResp.SessionID
	c.editURL = sessionResp.EditURL
	return nil
}

func (c *Client) GetEditURL() string {
	return c.editURL
}

func (c *Client) Connect() error {
	u, err := url.Parse(c.serverConf.Addr)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %w", err)
	}
	u.Path = fmt.Sprintf("/api/session/%s", c.sessionID)
	dialOption := &websocket.DialOptions{
		OnPingReceived: func(ctx context.Context, payload []byte) bool {
			return true
		},
	}
	if c.serverConf.Key != "" {
		header := http.Header{}
		header.Set("X-API-Key", c.serverConf.Key)
		dialOption.HTTPHeader = header
	}
	c.conn, _, err = websocket.Dial(c.ctx, u.String(), dialOption)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}
	return nil
}

func (c *Client) HandleMessages() error {
	for {
		var msg map[string]any
		if err := wsjson.Read(c.ctx, c.conn, &msg); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}
			if websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}
		msgType, ok := msg["type"].(string)
		if !ok {
			c.l.Warn("received message without type", "message", msg)
			continue
		}
		switch msgType {
		case "save":
			content := msg["content"].(string)
			err := os.WriteFile(c.filePath, []byte(content), 0644)
			if err != nil {
				c.l.Error("failed to write file", "error", err)
				c.SendResultMessage("save_result", false, "failed to save file")
				continue
			}
			c.l.Infof("file saved with %d bytes", len(content))
			c.SendResultMessage("save_result", true, "file saved successfully")
		}
	}
}

func (c *Client) Close(code websocket.StatusCode, reason string) error {
	if c.conn != nil {
		return c.conn.Close(code, reason)
	}
	return nil
}

func (c *Client) SendResultMessage(msgType string, success bool, reason string) error {
	resultMsg := ResultMessage{
		Type:    msgType,
		Success: success,
		Reason:  reason,
	}
	return wsjson.Write(c.ctx, c.conn, resultMsg)
}
