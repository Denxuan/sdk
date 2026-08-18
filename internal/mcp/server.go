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
	Tool *string `json:"tool,omitempty" jsonschema:"工具名称 / Tool name: java, nodejs, maven, mvnd, or go."`
}

type directoryInput struct {
	Directory string `json:"directory,omitempty" jsonschema:"项目目录 / Project directory; 留空表示当前工作目录 / empty means the current working directory."`
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

	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_list_installed", Description: "列出 SDK 管理的已安装版本，可按工具筛选。/ List SDK-managed installed versions, optionally filtered by tool."}, s.listInstalled)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_current_versions", Description: "列出所有工具当前正在使用的版本。/ List the currently active version for every tool."}, s.currentVersions)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_available_versions", Description: "列出指定工具的远程正式稳定版本。/ List stable official remote releases for one tool."}, s.availableVersions)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_path", Description: "返回工具当前版本的目录、current 软链接和可执行文件路径。/ Return the current tool directory, current symlink, and executable paths."}, s.path)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_project_versions", Description: "读取当前目录或父目录最近的 .sdk-version 项目版本文件。/ Read the nearest project .sdk-version file from the given directory or its parents."}, s.projectVersions)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_doctor", Description: "执行只读的 SDK 安装、链接和环境健康检查。/ Run read-only SDK installation, link, and environment health checks."}, s.doctor)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_state", Description: "返回 SDK 的本地状态快照，包括默认版本和已安装版本。/ Return the local SDK state snapshot, including defaults and installed versions."}, s.state)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "sdk_check_updates", Description: "检查已安装工具是否有更新的正式稳定版本。/ Check whether installed tools have newer stable official releases."}, s.checkUpdates)

	server.AddResource(serverResource("sdk://state", "SDK 状态 / SDK state", "当前 SDK 状态 / Current SDK state", "application/json"), s.resourceState)
	server.AddResource(serverResource("sdk://current", "当前版本 / Current versions", "当前正在使用的工具版本 / Currently active SDK versions", "application/json"), s.resourceCurrent)
	server.AddResource(serverResource("sdk://doctor", "健康检查 / Doctor report", "SDK 安装和环境健康报告 / SDK installation and environment health report", "application/json"), s.resourceDoctor)
	server.AddResource(serverResource("sdk://project/.sdk-version", "项目版本 / Project versions", "最近的 .sdk-version 项目版本配置 / Nearest project .sdk-version configuration", "application/json"), s.resourceProject)

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
		return nil, fmt.Errorf("不支持的工具 %q / unsupported tool; 支持的工具 / supported tools: java, nodejs, maven, mvnd, go", *value)
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
			err = fmt.Errorf("必须提供工具名称 / tool is required")
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
			err = fmt.Errorf("必须提供工具名称 / tool is required")
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
