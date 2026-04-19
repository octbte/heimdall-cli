package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/octobit/heimdall-cli/internal/api"
)

// Serve starts the MCP stdio server. Blocks until the client disconnects.
func Serve(client *api.Client) error {
	s := server.NewMCPServer(
		"heimdall",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	s.AddTool(
		mcp.NewTool("get_overview",
			mcp.WithDescription("Get a summary of events, errors, and performance metrics for the configured Heimdall project. Returns counts, average durations, top failing events, and latest errors."),
			mcp.WithString("from", mcp.Description("Start time filter in ISO 8601 format (e.g. 2026-04-01T00:00:00Z or 2026-04-01)")),
			mcp.WithString("to", mcp.Description("End time filter in ISO 8601 format")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.GetOverview(
				getString(req, "from"),
				getString(req, "to"),
			)
			if err != nil {
				return nil, fmt.Errorf("get_overview: %w", err)
			}
			return jsonResult(result)
		},
	)

	s.AddTool(
		mcp.NewTool("list_events",
			mcp.WithDescription("List business or technical events captured by the Heimdall SDK for the configured project. Use this to understand what happened in your application."),
			mcp.WithString("level", mcp.Description("Filter by log level: debug, info, warning, error, critical")),
			mcp.WithString("status", mcp.Description("Filter by status: success, error, warning, timeout, canceled")),
			mcp.WithString("event_name", mcp.Description("Filter by event name (e.g. 'pix_transfer_requested')")),
			mcp.WithString("from", mcp.Description("Start time filter (ISO 8601)")),
			mcp.WithString("to", mcp.Description("End time filter (ISO 8601)")),
			mcp.WithNumber("limit", mcp.Description("Number of results to return (default 50, max 200)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.ListEvents(api.ListEventsParams{
				Level:     getString(req, "level"),
				Status:    getString(req, "status"),
				EventName: getString(req, "event_name"),
				From:      getString(req, "from"),
				To:        getString(req, "to"),
				Limit:     getInt(req, "limit", 50),
			})
			if err != nil {
				return nil, fmt.Errorf("list_events: %w", err)
			}
			return jsonResult(result)
		},
	)

	s.AddTool(
		mcp.NewTool("get_event",
			mcp.WithDescription("Get a specific event by ID, including all metadata and tags. Use this to inspect a particular event in detail."),
			mcp.WithString("id", mcp.Required(), mcp.Description("The event UUID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := getString(req, "id")
			if id == "" {
				return nil, fmt.Errorf("id is required")
			}
			result, err := client.GetEvent(id)
			if err != nil {
				return nil, fmt.Errorf("get_event: %w", err)
			}
			return jsonResult(result)
		},
	)

	s.AddTool(
		mcp.NewTool("list_errors",
			mcp.WithDescription("List exceptions and errors captured by the Heimdall SDK. Use this to find recurring errors or investigate failures."),
			mcp.WithString("level", mcp.Description("Filter by log level: warning, error, critical")),
			mcp.WithString("fingerprint", mcp.Description("Filter by error fingerprint (groups similar errors)")),
			mcp.WithString("endpoint", mcp.Description("Filter by API endpoint path (e.g. '/api/transfers')")),
			mcp.WithString("from", mcp.Description("Start time filter (ISO 8601)")),
			mcp.WithString("to", mcp.Description("End time filter (ISO 8601)")),
			mcp.WithNumber("limit", mcp.Description("Number of results (default 50, max 200)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.ListErrors(api.ListErrorsParams{
				Level:       getString(req, "level"),
				Fingerprint: getString(req, "fingerprint"),
				Endpoint:    getString(req, "endpoint"),
				From:        getString(req, "from"),
				To:          getString(req, "to"),
				Limit:       getInt(req, "limit", 50),
			})
			if err != nil {
				return nil, fmt.Errorf("list_errors: %w", err)
			}
			return jsonResult(result)
		},
	)

	s.AddTool(
		mcp.NewTool("get_error",
			mcp.WithDescription("Get a specific error by ID, including the full stacktrace and metadata. Use this to understand the root cause of an exception and propose a fix."),
			mcp.WithString("id", mcp.Required(), mcp.Description("The error UUID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := getString(req, "id")
			if id == "" {
				return nil, fmt.Errorf("id is required")
			}
			result, err := client.GetError(id)
			if err != nil {
				return nil, fmt.Errorf("get_error: %w", err)
			}
			return jsonResult(result)
		},
	)

	s.AddTool(
		mcp.NewTool("list_performance",
			mcp.WithDescription("List performance measurements captured by the Heimdall SDK. Use this to find slow endpoints, jobs, or functions."),
			mcp.WithString("metric_name", mcp.Description("Filter by metric name (e.g. 'request_duration')")),
			mcp.WithString("target_type", mcp.Description("Filter by target type: http, job, function, task")),
			mcp.WithString("target_name", mcp.Description("Filter by target name (e.g. 'POST /api/transfers')")),
			mcp.WithString("from", mcp.Description("Start time filter (ISO 8601)")),
			mcp.WithString("to", mcp.Description("End time filter (ISO 8601)")),
			mcp.WithNumber("limit", mcp.Description("Number of results (default 50, max 200)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.ListPerformance(api.ListPerfParams{
				MetricName: getString(req, "metric_name"),
				TargetType: getString(req, "target_type"),
				TargetName: getString(req, "target_name"),
				From:       getString(req, "from"),
				To:         getString(req, "to"),
				Limit:      getInt(req, "limit", 50),
			})
			if err != nil {
				return nil, fmt.Errorf("list_performance: %w", err)
			}
			return jsonResult(result)
		},
	)

	s.AddTool(
		mcp.NewTool("get_performance",
			mcp.WithDescription("Get a specific performance record by ID, including duration and metadata."),
			mcp.WithString("id", mcp.Required(), mcp.Description("The performance record UUID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := getString(req, "id")
			if id == "" {
				return nil, fmt.Errorf("id is required")
			}
			result, err := client.GetPerformance(id)
			if err != nil {
				return nil, fmt.Errorf("get_performance: %w", err)
			}
			return jsonResult(result)
		},
	)

	return server.ServeStdio(s)
}

func getString(req mcp.CallToolRequest, key string) string {
	v, _ := req.Params.Arguments[key].(string)
	return v
}

func getInt(req mcp.CallToolRequest, key string, defaultVal int) int {
	if v, ok := req.Params.Arguments[key].(float64); ok && v > 0 {
		return int(v)
	}
	return defaultVal
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(data)), nil
}
