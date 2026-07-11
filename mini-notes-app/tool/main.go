package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

type JsonRpcRequest struct {
	JsonRpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

type JsonRpcResponse struct {
	JsonRpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JsonRpcError   `json:"error,omitempty"`
	ID      any             `json:"id"`
}

type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

var (
	stdoutMutex sync.Mutex
	pendingMtx  sync.Mutex
	pendingMap  = make(map[string]chan *JsonRpcResponse)
)

func logStderr(msg string) {
	fmt.Fprintf(os.Stderr, "[Executa Tool LOG] %s\n", msg)
}

func idToString(id any) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf("%v", id)
}

func registerCallback(id string, ch chan *JsonRpcResponse) {
	pendingMtx.Lock()
	defer pendingMtx.Unlock()
	pendingMap[id] = ch
}

func deregisterCallback(id string) {
	pendingMtx.Lock()
	defer pendingMtx.Unlock()
	delete(pendingMap, id)
}

func getCallback(id string) chan *JsonRpcResponse {
	pendingMtx.Lock()
	defer pendingMtx.Unlock()
	return pendingMap[id]
}

func writeStdout(data []byte) {
	stdoutMutex.Lock()
	defer stdoutMutex.Unlock()
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Sync()
}

func sendSuccess(id any, result any) {
	rawResult, _ := json.Marshal(result)
	resp := JsonRpcResponse{
		JsonRpc: "2.0",
		Result:  rawResult,
		ID:      id,
	}
	rawResp, _ := json.Marshal(resp)
	writeStdout(rawResp)
}

func sendError(id any, code int, msg string) {
	resp := JsonRpcResponse{
		JsonRpc: "2.0",
		Error: &JsonRpcError{
			Code:    code,
			Message: msg,
		},
		ID: id,
	}
	rawResp, _ := json.Marshal(resp)
	writeStdout(rawResp)
}

func main() {
	logStderr("Note Summarizer Executa Tool process initialized.")
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				logStderr("Received EOF signal. Terminating process.")
				break
			}
			logStderr(fmt.Sprintf("Failed to read stdin: %v", err))
			break
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			logStderr(fmt.Sprintf("Malformed JSON received: %v", err))
			continue
		}

		_, isReq := raw["method"]
		if isReq {
			go handleRequest(line)
		} else {
			go handleResponse(line)
		}
	}
}

func handleRequest(data []byte) {
	var req JsonRpcRequest
	if err := json.Unmarshal(data, &req); err != nil {
		sendError(nil, -32700, "Parse error")
		return
	}

	switch req.Method {
	case "initialize":
		handleInitialize(req)
	case "describe":
		handleDescribe(req)
	case "call", "invoke":
		handleCall(req)
	default:
		sendError(req.ID, -32601, fmt.Sprintf("Method '%s' not recognized.", req.Method))
	}
}

func handleResponse(data []byte) {
	var resp JsonRpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		logStderr(fmt.Sprintf("Error parsing response: %v", err))
		return
	}

	idStr := idToString(resp.ID)
	ch := getCallback(idStr)
	if ch != nil {
		ch <- &resp
	} else {
		logStderr(fmt.Sprintf("No registered channel matched for ID '%s'", idStr))
	}
}

func handleInitialize(req JsonRpcRequest) {
	logStderr("Negotiating capability protocol version v2.0...")
	res := map[string]any{
		"protocol_version": "2.0",
		"capabilities": map[string]any{
			"sampling": map[string]any{},
		},
		"server_info": map[string]any{
			"name":    "note-summarizer",
			"version": "0.1.0",
		},
	}
	sendSuccess(req.ID, res)
}

func handleDescribe(req JsonRpcRequest) {
	res := map[string]any{
		"name":              "note-summarizer",
		"display_name":      "Note Summarizer",
		"version":           "0.1.0",
		"description":       "Summarizes notes via reverse LLM sampling capabilities",
		"host_capabilities": []string{"llm.sample"},
		"tools": []map[string]any{
			{
				"name":        "summarize",
				"description": "Transforms a collection of textual notes into an actionable summarized review.",
				"parameters": []map[string]any{
					{
						"name":        "notes",
						"type":        "array",
						"description": "A collection of notes textual items.",
						"required":    true,
						"items": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
		"runtime": "go",
	}
	sendSuccess(req.ID, res)
}

type CallParams struct {
	Name      string          `json:"name"`
	Tool      string          `json:"tool"`
	ToolID    string          `json:"tool_id"`
	Args      json.RawMessage `json:"args"`
	Params    json.RawMessage `json:"params"`
	Arguments json.RawMessage `json:"arguments"`
}

type SummarizeArgs struct {
	Notes []string `json:"notes"`
}

func handleCall(req JsonRpcRequest) {
	var params CallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(req.ID, -32602, "Invalid parameters schema.")
		return
	}

	toolName := params.Name
	if toolName == "" {
		toolName = params.Tool
	}
	if toolName == "" {
		toolName = params.ToolID
	}

	if toolName != "summarize" {
		sendError(req.ID, -32601, fmt.Sprintf("Tool %s not found", toolName))
		return
	}

	var rawArgs json.RawMessage
	if len(params.Args) > 0 {
		rawArgs = params.Args
	} else if len(params.Params) > 0 {
		rawArgs = params.Params
	} else if len(params.Arguments) > 0 {
		rawArgs = params.Arguments
	}

	var args SummarizeArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		sendError(req.ID, -32602, "Arguments parsing failed.")
		return
	}

	if len(args.Notes) == 0 {
		sendSuccess(req.ID, map[string]any{"output": "Your notes list is empty."})
		return
	}

	var notesContent string
	for i, note := range args.Notes {
		notesContent += fmt.Sprintf("- [Note %d]: %s\n", i+1, note)
	}

	prompt := fmt.Sprintf(
		"Please read the following list of notes and provide a structured, cohesive bullet-point summary:\n\n%s",
		notesContent,
	)

	samplingID := fmt.Sprintf("samp-%v", req.ID)
	samplingReq := map[string]any{
		"jsonrpc": "2.0",
		"method":  "sampling/createMessage",
		"params": map[string]any{
			"messages": []map[string]any{
				{
					"role":    "user",
					"content": prompt,
				},
			},
			"max_tokens": 1024,
			"metadata": map[string]any{
				"invoke_id": fmt.Sprintf("%v", req.ID),
			},
		},
		"id": samplingID,
	}

	respCh := make(chan *JsonRpcResponse, 1)
	registerCallback(samplingID, respCh)
	defer deregisterCallback(samplingID)

	rawReq, _ := json.Marshal(samplingReq)
	logStderr(fmt.Sprintf("Dispatching reverse sampling request 'sampling/createMessage' with ID: %s", samplingID))
	writeStdout(rawReq)

	resp := <-respCh

	if resp.Error != nil {
		logStderr(fmt.Sprintf("Host sampling error returned: (%d) %s", resp.Error.Code, resp.Error.Message))
		sendError(req.ID, resp.Error.Code, fmt.Sprintf("Host LLM completion failed: %s", resp.Error.Message))
		return
	}

	logStderr(fmt.Sprintf("Raw Sampling Result matched from host: %s", string(resp.Result)))

	var rawMap map[string]any
	_ = json.Unmarshal(resp.Result, &rawMap)

	var summaryText string

	if content, exists := rawMap["content"]; exists {
		if contentArr, ok := content.([]any); ok && len(contentArr) > 0 {
			if firstBlock, ok := contentArr[0].(map[string]any); ok {
				if textVal, ok := firstBlock["text"].(string); ok {
					summaryText = textVal
				}
			}
		} else if contentObj, ok := content.(map[string]any); ok {
			if textVal, ok := contentObj["text"].(string); ok {
				summaryText = textVal
			}
		} else if contentStr, ok := content.(string); ok {
			summaryText = contentStr
		}
	}

	if summaryText == "" {
		if message, exists := rawMap["message"]; exists {
			if msgMap, ok := message.(map[string]any); ok {
				if contentStr, ok := msgMap["content"].(string); ok {
					summaryText = contentStr
				}
			}
		}
	}

	if summaryText == "" {
		if textVal, ok := rawMap["text"].(string); ok {
			summaryText = textVal
		} else {
			summaryText = string(resp.Result)
		}
	}

	logStderr("Reverse sampling transaction complete.")
	sendSuccess(req.ID, map[string]any{
		"output": summaryText,
	})
}