// Package main - types.go tests type definitions, interfaces, and methods.
package main

import (
	"fmt"
	"io"
	"sync"
)

// Config holds application configuration.
// Tests: struct extraction, field extraction, doc comments.
type Config struct {
	Host     string
	Port     int
	LogLevel string
	Options  map[string]interface{}
}

// Validate validates the configuration - method on Config.
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	return nil
}

// Clone creates a copy of the config - method on Config.
func (c *Config) Clone() *Config {
	return &Config{
		Host:     c.Host,
		Port:     c.Port,
		LogLevel: c.LogLevel,
	}
}

// Server represents the application server.
// Tests: struct with embedded types, multiple methods.
type Server struct {
	config *Config
	mu     sync.RWMutex
	logger *Logger
}

// NewServer creates a new server instance - constructor pattern.
// Tests: function returning struct pointer, composite literal.
func NewServer(cfg *Config) *Server {
	return &Server{
		config: cfg,
		logger: NewLogger("server"),
	}
}

// Start starts the server - public method, should be reachable via main.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting server on %s:%d", s.config.Host, s.config.Port)
	return s.listen()
}

// Stop stops the server - public method.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Stopping server")
	return nil
}

// listen is internal method - called by Start, should be reachable.
func (s *Server) listen() error {
	// Simulated listening
	return nil
}

// handleConnection handles a single connection - DEAD CODE (never called).
func (s *Server) handleConnection(conn io.ReadWriteCloser) {
	defer conn.Close()
	// Handle connection
}

// Logger provides logging functionality.
type Logger struct {
	prefix string
	level  int
}

// NewLogger creates a new logger - called from package init and NewServer.
func NewLogger(prefix string) *Logger {
	return &Logger{
		prefix: prefix,
		level:  1,
	}
}

// Info logs info message - called from Server.Start.
func (l *Logger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] %s: %s\n", l.prefix, fmt.Sprintf(format, args...))
}

// Debug logs debug message - DEAD CODE (never called).
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level >= 2 {
		fmt.Printf("[DEBUG] %s: %s\n", l.prefix, fmt.Sprintf(format, args...))
	}
}

// Error logs error message - DEAD CODE (never called).
func (l *Logger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] %s: %s\n", l.prefix, fmt.Sprintf(format, args...))
}

// Handler is an interface for request handlers.
// Tests: interface extraction.
type Handler interface {
	// Handle processes a request.
	Handle(req *Request) (*Response, error)
	// Name returns the handler name.
	Name() string
}

// Request represents an incoming request.
type Request struct {
	Method string
	Path   string
	Body   []byte
}

// Response represents an outgoing response.
type Response struct {
	Status int
	Body   []byte
}

// EchoHandler implements Handler interface.
// Tests: interface implementation detection.
type EchoHandler struct {
	name string
}

// NewEchoHandler creates a new echo handler - DEAD CODE (never called).
func NewEchoHandler() *EchoHandler {
	return &EchoHandler{name: "echo"}
}

// Handle implements Handler.Handle - part of interface impl.
func (h *EchoHandler) Handle(req *Request) (*Response, error) {
	return &Response{
		Status: 200,
		Body:   req.Body,
	}, nil
}

// Name implements Handler.Name - part of interface impl.
func (h *EchoHandler) Name() string {
	return h.name
}

// UnusedType is a type that is never used - DEAD CODE.
type UnusedType struct {
	Field1 string
	Field2 int
}

// Process is a method on unused type - DEAD CODE.
func (u *UnusedType) Process() {
	fmt.Println("Processing:", u.Field1)
}

// TypeAlias tests type alias extraction.
type TypeAlias = map[string]interface{}

// StringList tests named type extraction.
type StringList []string

// Len returns length - method on named type.
func (s StringList) Len() int {
	return len(s)
}

// genericContainer tests generic type (Go 1.18+).
type genericContainer[T any] struct {
	items []T
}

// Add adds item to container - generic method.
func (c *genericContainer[T]) Add(item T) {
	c.items = append(c.items, item)
}

// Get returns item at index - generic method.
func (c *genericContainer[T]) Get(index int) T {
	return c.items[index]
}
