// Command topagentnetwork-mcp is a stdio MCP server exposing
// topagentnetwork-go.
//
// Config: ~/.topagentnetwork-mcp/config.json
//
//	{
//	  "email":         "you@example.com",
//	  "password":      "...",
//	  "cookie_header": "paste from browser DevTools → Network → /api/login → Request Headers → cookie"
//	}
//
// Env override: TOPAGENTNETWORK_EMAIL, TOPAGENTNETWORK_PASSWORD,
// TOPAGENTNETWORK_COOKIE_HEADER.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tan "github.com/teslashibe/topagentnetwork-go"
	tanmcp "github.com/teslashibe/topagentnetwork-go/mcp"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type configFile struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	CookieHeader string `json:"cookie_header,omitempty"`
}

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".topagentnetwork-mcp", "config.json")
}

func loadAuth() (tan.Auth, error) {
	var cfg configFile
	data, err := os.ReadFile(defaultConfigPath())
	if err != nil && !os.IsNotExist(err) {
		return tan.Auth{}, fmt.Errorf("read config: %w", err)
	}
	if data != nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return tan.Auth{}, fmt.Errorf("parse config: %w", err)
		}
	}
	if v := os.Getenv("TOPAGENTNETWORK_EMAIL"); v != "" {
		cfg.Email = v
	}
	if v := os.Getenv("TOPAGENTNETWORK_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := os.Getenv("TOPAGENTNETWORK_COOKIE_HEADER"); v != "" {
		cfg.CookieHeader = v
	}
	if cfg.CookieHeader == "" && (cfg.Email == "" || cfg.Password == "") {
		return tan.Auth{}, fmt.Errorf(
			"topagentnetwork credentials not found. Paste a browser cookie into %s "+
				"as 'cookie_header' (the website's /api/login proxy 500s for "+
				"non-browser callers, so cookie-paste is the working auth path).",
			defaultConfigPath())
	}
	return tan.Auth{
		Email:        cfg.Email,
		Password:     cfg.Password,
		CookieHeader: cfg.CookieHeader,
	}, nil
}

func main() {
	log.SetOutput(os.Stderr)
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "topagentnetwork-mcp:", err)
		os.Exit(1)
	}
}

func run() error {
	auth, err := loadAuth()
	if err != nil {
		return err
	}
	client, err := tan.New(auth)
	if err != nil {
		return fmt.Errorf("init client: %w", err)
	}
	s := server.NewMCPServer("topagentnetwork-mcp", "0.1.0", server.WithToolCapabilities(true))
	for _, t := range (tanmcp.Provider{}).Tools() {
		t := t
		rawSchema, err := json.Marshal(t.InputSchema)
		if err != nil {
			return fmt.Errorf("marshal schema for %s: %w", t.Name, err)
		}
		tool := mcp.NewToolWithRawSchema(t.Name, t.Description, rawSchema)
		s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw, err := json.Marshal(req.Params.Arguments)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, invokeErr := t.Invoke(ctx, client, raw)
			if invokeErr != nil {
				return mcp.NewToolResultError(invokeErr.Error()), nil
			}
			out, err := json.Marshal(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(out)), nil
		})
	}
	return server.ServeStdio(s)
}
