package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
)

// JSON-RPC 2.0 types

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP protocol types

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// ToolHandler is the function signature for tool implementations.
type ToolHandler func(args map[string]any) (any, error)

// Server is the MCP JSON-RPC server over stdio.
type Server struct {
	info    ServerInfo
	tools   []Tool
	handler map[string]ToolHandler
	mu      sync.RWMutex
}

func New(name, version string) *Server {
	return &Server{
		info:    ServerInfo{Name: name, Version: version},
		handler: make(map[string]ToolHandler),
	}
}

func (s *Server) RegisterTool(tool Tool, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, tool)
	s.handler[tool.Name] = handler
}

func (s *Server) Run(in io.Reader, out io.Writer, errOut io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)
	enc := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(Response{
				JSONRPC: "2.0",
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
			})
			continue
		}

		if req.ID == nil {
			s.handleNotification(req, errOut)
			continue
		}

		if err := enc.Encode(s.handleRequest(req)); err != nil {
			fmt.Fprintf(errOut, "hudl-mcp: encode error: %v\n", err)
		}
	}

	return scanner.Err()
}

func (s *Server) handleNotification(req Request, errOut io.Writer) {
	switch req.Method {
	case "notifications/initialized", "notifications/cancelled":
	default:
		fmt.Fprintf(errOut, "hudl-mcp: unknown notification %q\n", req.Method)
	}
}

func (s *Server) handleRequest(req Request) Response {
	switch req.Method {
	case "initialize":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities:    ServerCapabilities{Tools: &ToolsCapability{}},
			ServerInfo:      s.info,
		}}
	case "tools/list":
		s.mu.RLock()
		defer s.mu.RUnlock()
		return Response{JSONRPC: "2.0", ID: req.ID, Result: ToolsListResult{Tools: s.tools}}
	case "tools/call":
		return s.handleToolsCall(req)
	case "ping":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "resources/list":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"resources": []any{}}}
	case "prompts/list":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"prompts": []any{}}}
	default:
		return Response{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32601, Message: "Method not found: " + req.Method}}
	}
}

func (s *Server) handleToolsCall(req Request) Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32602, Message: "Invalid params: " + err.Error()}}
	}

	s.mu.RLock()
	handler, ok := s.handler[params.Name]
	s.mu.RUnlock()

	if !ok {
		return Response{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32602, Message: "Unknown tool: " + params.Name}}
	}

	result, err := handler(params.Arguments)
	if err != nil {
		return Response{JSONRPC: "2.0", ID: req.ID, Result: CallToolResult{
			Content: []Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}}
	}

	text, err := marshalResult(result)
	if err != nil {
		return Response{JSONRPC: "2.0", ID: req.ID, Result: CallToolResult{
			Content: []Content{{Type: "text", Text: "serialize error: " + err.Error()}},
			IsError: true,
		}}
	}

	return Response{JSONRPC: "2.0", ID: req.ID, Result: CallToolResult{
		Content: []Content{{Type: "text", Text: text}},
	}}
}

func marshalResult(value any) (string, error) {
	if s, ok := value.(string); ok {
		return s, nil
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Schema helpers

func ObjectSchema(description string, properties map[string]any, required []string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func StringProp(desc string) map[string]any       { return map[string]any{"type": "string", "description": desc} }
func IntProp(desc string) map[string]any          { return map[string]any{"type": "integer", "description": desc} }
func BoolProp(desc string) map[string]any         { return map[string]any{"type": "boolean", "description": desc} }
func StringArrayProp(desc string) map[string]any  { return map[string]any{"type": "array", "description": desc, "items": map[string]any{"type": "string"}} }
func EnumProp(desc string, vals []string) map[string]any {
	anys := make([]any, len(vals))
	for i, v := range vals { anys[i] = v }
	return map[string]any{"type": "string", "description": desc, "enum": anys}
}

// Arg extraction helpers

func ArgString(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		switch t := v.(type) {
		case string:
			return t
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64)
		default:
			return fmt.Sprintf("%v", t)
		}
	}
	return ""
}

func ArgInt(args map[string]any, key string) int {
	if v, ok := args[key]; ok {
		switch t := v.(type) {
		case float64:
			return int(t)
		case int:
			return t
		case string:
			n, _ := strconv.Atoi(t)
			return n
		}
	}
	return 0
}

func ArgBool(args map[string]any, key string, def bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func ArgStringArray(args map[string]any, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}
