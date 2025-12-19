package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// ToolHandler is a function that handles a tool call
type ToolHandler func(arguments map[string]interface{}) (*CallToolResult, error)

// Server represents an MCP server
type Server struct {
	name     string
	version  string
	tools    []Tool
	handlers map[string]ToolHandler
	mu       sync.RWMutex
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
}

// NewServer creates a new MCP server
func NewServer(name, version string) *Server {
	return &Server{
		name:     name,
		version:  version,
		tools:    make([]Tool, 0),
		handlers: make(map[string]ToolHandler),
		stdin:    os.Stdin,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}
}

// RegisterTool registers a tool with its handler
func (s *Server) RegisterTool(tool Tool, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, tool)
	s.handlers[tool.Name] = handler
}

// Run starts the server and processes requests from stdin
func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.stdin)
	// Increase buffer size for large messages
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		response := s.handleMessage([]byte(line))
		if response != nil {
			s.sendResponse(response)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

func (s *Server) handleMessage(data []byte) *JSONRPCResponse {
	var request JSONRPCRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    ParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
	}

	// Handle notifications (no ID)
	if request.ID == nil {
		s.handleNotification(&request)
		return nil
	}

	return s.handleRequest(&request)
}

func (s *Server) handleNotification(request *JSONRPCRequest) {
	switch request.Method {
	case "notifications/initialized":
		// Client initialized notification, no action needed
		fmt.Fprintln(s.stderr, "Client initialized")
	case "notifications/cancelled":
		// Request cancellation, no action needed for now
	}
}

func (s *Server) handleRequest(request *JSONRPCRequest) *JSONRPCResponse {
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	switch request.Method {
	case "initialize":
		response.Result = s.handleInitialize(request.Params)
	case "tools/list":
		response.Result = s.handleListTools()
	case "tools/call":
		result, err := s.handleCallTool(request.Params)
		if err != nil {
			response.Error = &JSONRPCError{
				Code:    InternalError,
				Message: err.Error(),
			}
		} else {
			response.Result = result
		}
	case "ping":
		response.Result = map[string]interface{}{}
	default:
		response.Error = &JSONRPCError{
			Code:    MethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", request.Method),
		}
	}

	return response
}

func (s *Server) handleInitialize(params interface{}) *InitializeResult {
	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
		},
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}
}

func (s *Server) handleListTools() *ListToolsResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &ListToolsResult{
		Tools: s.tools,
	}
}

func (s *Server) handleCallTool(params interface{}) (*CallToolResult, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid params type")
	}

	name, ok := paramsMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tool name")
	}

	arguments, _ := paramsMap["arguments"].(map[string]interface{})

	s.mu.RLock()
	handler, exists := s.handlers[name]
	s.mu.RUnlock()

	if !exists {
		return &CallToolResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Unknown tool: %s", name)}},
			IsError: true,
		}, nil
	}

	return handler(arguments)
}

func (s *Server) sendResponse(response *JSONRPCResponse) {
	data, err := json.Marshal(response)
	if err != nil {
		fmt.Fprintf(s.stderr, "Error marshaling response: %v\n", err)
		return
	}
	fmt.Fprintln(s.stdout, string(data))
}

// Log writes a message to stderr for debugging
func (s *Server) Log(format string, args ...interface{}) {
	fmt.Fprintf(s.stderr, format+"\n", args...)
}
