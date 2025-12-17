package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

/**
 * Knot MCP Server - Read-only API documentation query interface
 * Provides tools to explore API groups and their endpoints
 *
 * Design Principles:
 * - Read-only operations only (no create/update/delete)
 * - Fuzzy search support for group names
 * - Hierarchical data structure (groups → APIs → parameters)
 */

// Configuration
var (
	KNOT_BASE_URL = getEnv("KNOT_BASE_URL", "http://localhost:3000")
	API_ENDPOINT  = KNOT_BASE_URL + "/api/mcp-tools"
)

// APIRequest represents the request to backend API
type APIRequest struct {
	Tool string                 `json:"tool"`
	Args map[string]interface{} `json:"args"`
}

// APIResponse represents the response from backend API
type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// getEnv gets environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// callAPI makes a call to the backend API
func callAPI(tool string, args map[string]interface{}) (interface{}, error) {
	reqBody := APIRequest{
		Tool: tool,
		Args: args,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(API_ENDPOINT, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp APIResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("API call failed: %s", resp.Status)
	}

	var result APIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}

	return result.Data, nil
}

func main() {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"knot-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
		server.WithPromptCapabilities(true),
	)

	// Register list_groups tool
	mcpServer.AddTool(mcp.Tool{
		Name:        "list_groups",
		Description: "List all API groups in the Knot database. Returns an array of all available API groups with their IDs and names. Use this as the starting point to explore the API catalog.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
			Required:   []string{},
		},
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		data, err := callAPI("list_groups", args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// Register get_group tool
	mcpServer.AddTool(mcp.Tool{
		Name:        "get_group",
		Description: "Get detailed information about a specific API group. Supports fuzzy matching - you can provide a partial group name (e.g., 'user' will match 'USER-SERVICE'). Returns group details including the total count of APIs in that group. Use this to verify the exact group name before listing its APIs.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"groupName": map[string]interface{}{
					"type":        "string",
					"description": "Full or partial name of the API group. Case-insensitive fuzzy matching is applied. Examples: 'user', 'auth', 'payment'",
				},
			},
			Required: []string{"groupName"},
		},
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		data, err := callAPI("get_group", args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// Register list_apis_by_group tool
	mcpServer.AddTool(mcp.Tool{
		Name:        "list_apis_by_group",
		Description: "List all APIs within a specific group. Supports fuzzy matching on group name - you can provide a partial name. Returns the group information and an array of all APIs in that group, including API ID, name, endpoint, method (GET/POST/etc), and type (HTTP/RPC). This is the primary tool to discover APIs within a known or partially-known group.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"groupName": map[string]interface{}{
					"type":        "string",
					"description": "Full or partial name of the API group. Examples: 'user-service', 'auth', 'payment-gateway'. Case-insensitive fuzzy matching is applied.",
				},
			},
			Required: []string{"groupName"},
		},
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		data, err := callAPI("list_apis_by_group", args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// Register get_api tool
	mcpServer.AddTool(mcp.Tool{
		Name:        "get_api",
		Description: "Get comprehensive details about a specific API. Returns full API documentation including: endpoint, HTTP method, type (HTTP/RPC), group name, and hierarchical request/response parameters with types, descriptions, and required flags. Use this after identifying the API ID from list_apis_by_group or search_apis.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"apiId": map[string]interface{}{
					"type":        "number",
					"description": "The unique ID of the API (obtained from list_apis_by_group or search_apis results)",
				},
			},
			Required: []string{"apiId"},
		},
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		data, err := callAPI("get_api", args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// Register search_apis tool
	mcpServer.AddTool(mcp.Tool{
		Name:        "search_apis",
		Description: "Search for APIs across all groups by name or endpoint path. Performs fuzzy matching on both API name and endpoint URL. Returns up to 50 matching APIs with their group names. Use this when you know part of an API name or endpoint but don't know which group it belongs to.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search term to match against API names or endpoint paths. Examples: 'login', '/api/user', 'transaction', '流程'. Case-insensitive partial matching is applied.",
				},
			},
			Required: []string{"query"},
		},
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		data, err := callAPI("search_apis", args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// Register get_api_json_example tool
	mcpServer.AddTool(mcp.Tool{
		Name:        "get_api_json_example",
		Description: "Generate example JSON for a specific API's request and response payloads. Returns the API name, endpoint, HTTP method, and auto-generated example JSON structures based on the parameter definitions. Use this to understand the expected data format for API calls.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"apiId": map[string]interface{}{
					"type":        "number",
					"description": "The unique ID of the API (obtained from list_apis_by_group or search_apis results)",
				},
			},
			Required: []string{"apiId"},
		},
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		data, err := callAPI("get_api_json_example", args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// Register resource
	mcpServer.AddResource(
		mcp.Resource{
			URI:         "knot://groups",
			Name:        "All API Groups",
			Description: "List of all available API groups in the Knot database",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			data, err := callAPI("list_groups", map[string]interface{}{})
			if err != nil {
				return nil, fmt.Errorf("failed to fetch groups: %w", err)
			}

			jsonData, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format result: %w", err)
			}

			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "knot://groups",
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			}, nil
		},
	)

	// Register prompts
	mcpServer.AddPrompt(
		mcp.Prompt{
			Name:        "explore-api-group",
			Description: "Explore APIs in a specific group",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "groupName",
					Description: "Name of the API group to explore",
					Required:    true,
				},
			},
		},
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			groupName, ok := request.Params.Arguments["groupName"]
			if !ok || groupName == "" {
				return nil, fmt.Errorf("groupName argument is required")
			}

			return &mcp.GetPromptResult{
				Messages: []mcp.PromptMessage{
					{
						Role: "user",
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Please show me all APIs in the \"%s\" group. Include their names, endpoints, and methods.", groupName),
						},
					},
				},
			}, nil
		},
	)

	mcpServer.AddPrompt(
		mcp.Prompt{
			Name:        "find-api",
			Description: "Find an API by name or endpoint",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "query",
					Description: "Search term for API name or endpoint",
					Required:    true,
				},
			},
		},
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			query, ok := request.Params.Arguments["query"]
			if !ok || query == "" {
				return nil, fmt.Errorf("query argument is required")
			}

			return &mcp.GetPromptResult{
				Messages: []mcp.PromptMessage{
					{
						Role: "user",
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Search for APIs matching \"%s\" and show me the results with their details.", query),
						},
					},
				},
			}, nil
		},
	)

	// Start server on stdio
	log.Printf("Knot MCP Server running on stdio")
	log.Printf("Connecting to: %s", KNOT_BASE_URL)

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
