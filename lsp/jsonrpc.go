package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Message is a JSON-RPC 2.0 envelope.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (m *Message) IsRequest() bool      { return m.Method != "" && m.ID != nil }
func (m *Message) IsNotification() bool { return m.Method != "" && m.ID == nil }
func (m *Message) IsResponse() bool     { return m.Method == "" }

// ReadMessage reads one LSP message with Content-Length framing from r.
func ReadMessage(r *bufio.Reader) (*Message, error) {
	var contentLength int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			n, err := strconv.Atoi(strings.TrimPrefix(line, "Content-Length: "))
			if err != nil {
				return nil, fmt.Errorf("parse Content-Length: %w", err)
			}
			contentLength = n
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("missing or zero Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}
	return &msg, nil
}

// Writer serializes LSP messages with Content-Length framing. Thread-safe.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
}

func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

func (wr *Writer) Write(msg *Message) error {
	if msg.JSONRPC == "" {
		msg.JSONRPC = "2.0"
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	wr.mu.Lock()
	defer wr.mu.Unlock()
	if _, err := fmt.Fprintf(wr.w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = wr.w.Write(body)
	return err
}
