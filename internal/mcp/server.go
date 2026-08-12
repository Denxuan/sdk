package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Denxuan/sdk/internal/buildinfo"
	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/service"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server exposes SDK's read-only state through the Model Context Protocol.
type Server struct {
	read *service.ReadService
}

func NewServer(home string) *Server { return &Server{read: service.NewRead(home)} }

type toolInput struct {
	Tool *string `json:"tool,omitempty" jsonschema:"Tool name (java, nodejs, maven, or go)."`
}

type directoryInput struct {
	Directory string `json:"directory,omitempty" jsonschema:"Project directory; defaults to the current working directory."`
}

type installedOutput struct {
	Items []service.Installed `json:"items"`
}

type currentOutput struct {
	Items []service.Current `json:"items"`
}

type availableOutput struct {
	Items []service.Available `json:"items"`
}

type pathOutput struct {
	Tool        model.Tool        `json:"tool"`
	Path        string            `json:"path"`
	Executables map[string]string `json:"executables"`
}

func (s *Server) Run(ctx context.Context) error {
	server := s.protocolServer()
	return server.Run(ctx, &sdkmcp.StdioTransport{})
}

func (s *Server) protocolServer() *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "sdk", Version: buildinfo.Version}, nil)

	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_list_installed", Description: "List SDK-managed installations, optionally filtered by tool."}, s.listInstalled)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_current_versions", Description: "List the currently selected version for every tool."}, s.currentVersions)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_available_versions", Description: "List stable remote versions for one tool."}, s.availableVersions)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_path", Description: "Return the current symlink path for a tool."}, s.path)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_project_versions", Description: "Read the nearest .sdk-version file."}, s.projectVersions)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_doctor", Description: "Run read-only SDK health checks."}, s.doctor)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_state", Description: "Return the SDK state snapshot."}, s.state)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_check_updates", Description: "Check whether installed tools have newer stable versions."}, s.checkUpdates)

	server.AddResource(serverResource("sdk://state", "SDK state", "Current SDK state", "application/json"), s.resourceState)
	server.AddResource(serverResource("sdk://current", "Current versions", "Currently selected SDK versions", "application/json"), s.resourceCurrent)
	server.AddResource(serverResource("sdk://doctor", "Doctor report", "SDK health report", "application/json"), s.resourceDoctor)
	server.AddResource(serverResource("sdk://project/.sdk-version", "Project versions", "Nearest project version file", "application/json"), s.resourceProject)

	return server
}

func serverResource(uri, name, description, mime string) *sdkmcp.Resource {
	return &sdkmcp.Resource{URI: uri, Name: name, Description: description, MIMEType: mime}
}

func parseTool(value *string) (*model.Tool, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	tool := model.Tool(*value)
	if !tool.Valid() {
		return nil, fmt.Errorf("unsupported tool %q; supported: java, nodejs, maven, go", *value)
	}
	return &tool, nil
}

func (s *Server) listInstalled(ctx context.Context, _ *sdkmcp.CallToolRequest, in toolInput) (*sdkmcp.CallToolResult, installedOutput, error) {
	tool, err := parseTool(in.Tool)
	if err != nil {
		return nil, installedOutput{}, err
	}
	items, err := s.read.ListInstalled(ctx, tool)
	return nil, installedOutput{Items: items}, err
}

func (s *Server) currentVersions(ctx context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, currentOutput, error) {
	items, err := s.read.CurrentVersions(ctx)
	return nil, currentOutput{Items: items}, err
}

func (s *Server) availableVersions(ctx context.Context, _ *sdkmcp.CallToolRequest, in toolInput) (*sdkmcp.CallToolResult, availableOutput, error) {
	tool, err := parseTool(in.Tool)
	if err != nil || tool == nil {
		if err == nil {
			err = fmt.Errorf("tool is required")
		}
		return nil, availableOutput{}, err
	}
	items, err := s.read.AvailableVersions(ctx, *tool)
	return nil, availableOutput{Items: items}, err
}

func (s *Server) path(ctx context.Context, _ *sdkmcp.CallToolRequest, in toolInput) (*sdkmcp.CallToolResult, pathOutput, error) {
	tool, err := parseTool(in.Tool)
	if err != nil || tool == nil {
		if err == nil {
			err = fmt.Errorf("tool is required")
		}
		return nil, pathOutput{}, err
	}
	path, err := s.read.ToolPath(ctx, *tool)
	if err != nil {
		return nil, pathOutput{}, err
	}
	executables, err := s.read.ToolExecutables(ctx, *tool)
	return nil, pathOutput{Tool: *tool, Path: path, Executables: executables}, err
}

func (s *Server) projectVersions(ctx context.Context, _ *sdkmcp.CallToolRequest, in directoryInput) (*sdkmcp.CallToolResult, service.Project, error) {
	project, err := s.read.ProjectVersions(ctx, in.Directory)
	return nil, project, err
}

func (s *Server) doctor(ctx context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, service.Doctor, error) {
	report, err := s.read.DoctorReport(ctx)
	return nil, report, err
}

func (s *Server) state(ctx context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, map[string]any, error) {
	data, err := s.read.StateJSON(ctx)
	if err != nil {
		return nil, nil, err
	}
	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, nil, err
	}
	return nil, state, nil
}

func (s *Server) checkUpdates(ctx context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, map[string]any, error) {
	items, err := s.read.CheckUpdates(ctx)
	return nil, map[string]any{"items": items}, err
}

func (s *Server) resourceState(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	data, err := s.read.StateJSON(ctx)
	return resourceResult(req.Params.URI, data, err)
}

func (s *Server) resourceCurrent(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	data, err := s.read.CurrentVersions(ctx)
	return marshalResource(req.Params.URI, data, err)
}

func (s *Server) resourceDoctor(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	data, err := s.read.DoctorReport(ctx)
	return marshalResource(req.Params.URI, data, err)
}

func (s *Server) resourceProject(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	data, err := s.read.ProjectVersions(ctx, "")
	return marshalResource(req.Params.URI, data, err)
}

func marshalResource(uri string, value any, err error) (*sdkmcp.ReadResourceResult, error) {
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(value)
	return resourceResult(uri, data, err)
}

func resourceResult(uri string, data []byte, err error) (*sdkmcp.ReadResourceResult, error) {
	if err != nil {
		return nil, err
	}
	return &sdkmcp.ReadResourceResult{Contents: []*sdkmcp.ResourceContents{{URI: uri, MIMEType: "application/json", Text: string(data)}}}, nil
}
