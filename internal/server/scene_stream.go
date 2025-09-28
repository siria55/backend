package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"eeo/backend/internal/service/game"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type sceneStream struct {
	service     *game.Service
	interval    time.Duration
	seconds     float64
	drainFactor float64

	ctx    context.Context
	cancel context.CancelFunc

	mu          sync.RWMutex
	clients     map[*sceneStreamClient]struct{}
	lastPayload []byte
}

type sceneStreamClient struct {
	stream *sceneStream
	conn   *websocket.Conn
	send   chan string
	once   sync.Once
}

func newSceneStream(svc *game.Service, interval time.Duration, seconds, drainFactor float64) *sceneStream {
	if interval <= 0 {
		interval = time.Second
	}
	if seconds <= 0 {
		seconds = interval.Seconds()
	}
	if drainFactor <= 0 {
		drainFactor = game.DefaultDrainFactor
	}

	ctx, cancel := context.WithCancel(context.Background())

	stream := &sceneStream{
		service:     svc,
		interval:    interval,
		seconds:     seconds,
		drainFactor: drainFactor,
		ctx:         ctx,
		cancel:      cancel,
		clients:     make(map[*sceneStreamClient]struct{}),
	}

	go stream.loop()

	return stream
}

func (s *sceneStream) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *sceneStream) tick() {
	scene, err := s.service.AdvanceEnergyState(s.ctx, s.seconds, s.drainFactor)
	if err != nil {
		log.Printf("sceneStream: advance energy failed: %v", err)
		return
	}

	payload, err := json.Marshal(scene)
	if err != nil {
		log.Printf("sceneStream: marshal scene failed: %v", err)
		return
	}

	s.mu.Lock()
	s.lastPayload = payload
	clients := make([]*sceneStreamClient, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.mu.Unlock()

	for _, client := range clients {
		client.enqueue(payload)
	}
}

func (s *sceneStream) handle(c *gin.Context) {
	wsServer := websocket.Server{
		Handshake: func(cfg *websocket.Config, req *http.Request) error {
			return nil
		},
		Handler: func(conn *websocket.Conn) {
			s.addClient(conn)
		},
	}

	wsServer.ServeHTTP(c.Writer, c.Request)
}

func (s *sceneStream) addClient(conn *websocket.Conn) {
	client := &sceneStreamClient{
		stream: s,
		conn:   conn,
		send:   make(chan string, 8),
	}

	s.mu.Lock()
	s.clients[client] = struct{}{}
	s.mu.Unlock()

	if payload := s.initialPayload(); len(payload) > 0 {
		client.enqueue(payload)
	}

	go client.writeLoop()
	go client.readLoop()
}

func (s *sceneStream) initialPayload() []byte {
	s.mu.RLock()
	if s.lastPayload != nil {
		payload := make([]byte, len(s.lastPayload))
		copy(payload, s.lastPayload)
		s.mu.RUnlock()
		return payload
	}
	s.mu.RUnlock()

	payload, err := json.Marshal(s.service.Scene())
	if err != nil {
		log.Printf("sceneStream: marshal initial scene failed: %v", err)
		return nil
	}
	return payload
}

func (s *sceneStream) stop() {
	s.cancel()

	s.mu.Lock()
	clients := make([]*sceneStreamClient, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.mu.Unlock()

	for _, client := range clients {
		client.close()
	}
}

func (c *sceneStreamClient) enqueue(payload []byte) {
	if len(payload) == 0 {
		return
	}
	message := string(payload)
	select {
	case c.send <- message:
	default:
		c.close()
	}
}

func (c *sceneStreamClient) close() {
	c.once.Do(func() {
		c.stream.removeClient(c)
	})
}

func (s *sceneStream) removeClient(client *sceneStreamClient) {
	s.mu.Lock()
	if _, ok := s.clients[client]; ok {
		delete(s.clients, client)
	}
	s.mu.Unlock()

	close(client.send)
	_ = client.conn.Close()
}

func (c *sceneStreamClient) writeLoop() {
	defer c.close()

	for payload := range c.send {
		if err := websocket.Message.Send(c.conn, payload); err != nil {
			return
		}
	}
}

func (c *sceneStreamClient) readLoop() {
	defer c.close()

	for {
		var message string
		if err := websocket.Message.Receive(c.conn, &message); err != nil {
			return
		}
	}
}
