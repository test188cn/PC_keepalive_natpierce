package wsclient

import (
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EventType categorizes WebSocket events.
type EventType int

const (
	EventConnected    EventType = iota
	EventDisconnected
	EventMessage
)

// Event represents a WebSocket event.
type Event struct {
	Type    EventType
	Message string // raw text, only valid for EventMessage
	Reason  string // disconnect reason, only valid for EventDisconnected
}

// Listener is called when a WebSocket event occurs.
// IMPORTANT: This is called from the recv goroutine.
// For UI updates, use walk.CallFromGoroutine().
type Listener func(Event)

// Client wraps gorilla/websocket with a simple send/recv API.
type Client struct {
	conn      *websocket.Conn
	listener  Listener
	sendMu    sync.Mutex
	connected bool
	done      chan struct{}
}

// New creates a new WebSocket client with the given event listener.
func New(listener Listener) *Client {
	return &Client{
		listener: listener,
		done:     make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the given ws:// URL.
// URL format: "ws://host:port/path" or "host:port/path"
func (c *Client) Connect(rawURL string) error {
	// Ensure proper URL scheme
	if len(rawURL) > 0 && rawURL[:2] != "ws" {
		rawURL = "ws://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	dialer := websocket.Dialer{
		ReadBufferSize:  65536,
		WriteBufferSize: 65536,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	c.conn = conn
	c.connected = true
	c.done = make(chan struct{})

	// Set initial read deadline so dead connections are detected
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Start recv goroutine
	go c.recvLoop()

	// Notify connected
	c.listener(Event{Type: EventConnected})

	return nil
}

// Send sends a text message. Returns false if not connected or send fails.
func (c *Client) Send(text string) bool {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	if c.conn == nil || !c.connected {
		return false
	}

	err := c.conn.WriteMessage(websocket.TextMessage, []byte(text))
	if err != nil {
		return false
	}
	return true
}

// Close gracefully shuts down the connection.
func (c *Client) Close() {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	if c.conn == nil {
		return
	}

	// Send close frame
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	c.conn.WriteMessage(websocket.CloseMessage, msg)
	c.conn.Close()
	c.conn = nil
	c.connected = false
}

// recvLoop reads messages from the WebSocket in a loop.
func (c *Client) recvLoop() {
	defer func() {
		c.sendMu.Lock()
		wasConnected := c.connected
		c.connected = false
		c.conn = nil
		c.sendMu.Unlock()

		if wasConnected {
			c.listener(Event{Type: EventDisconnected, Reason: "连接关闭"})
		}
		close(c.done)
	}()

	// Set up ping/pong handling
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Respond to server pings with pong
	c.conn.SetPingHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return c.conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	// Start ping ticker
	go c.pingLoop()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		// Reset read deadline on every received message (not just pong/ping)
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		c.listener(Event{Type: EventMessage, Message: string(message)})
	}
}

// pingLoop sends periodic ping messages.
func (c *Client) pingLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.sendMu.Lock()
			if c.conn != nil && c.connected {
				c.conn.WriteMessage(websocket.PingMessage, nil)
			}
			c.sendMu.Unlock()
		}
	}
}
