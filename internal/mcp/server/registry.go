package server

import (
	"fmt"
	"sync"
)

// Registry manages tools and resources for an MCP server
type Registry struct {
	tools     map[string]registeredTool
	resources map[string]registeredResource
	templates map[string]registeredTemplate
	mu        sync.RWMutex
}

type registeredTool struct {
	definition Tool
	handler    ToolHandler
}

type registeredResource struct {
	definition Resource
	handler    ResourceHandler
}

type registeredTemplate struct {
	definition ResourceTemplate
	handler    ResourceHandler
}

// NewRegistry creates a new tool/resource registry
func NewRegistry() *Registry {
	return &Registry{
		tools:     make(map[string]registeredTool),
		resources: make(map[string]registeredResource),
		templates: make(map[string]registeredTemplate),
	}
}

// RegisterTool adds a tool to the registry
func (r *Registry) RegisterTool(tool Tool, handler ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if handler == nil {
		return fmt.Errorf("tool handler is required")
	}
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}

	r.tools[tool.Name] = registeredTool{
		definition: tool,
		handler:    handler,
	}
	return nil
}

// UnregisterTool removes a tool from the registry
func (r *Registry) UnregisterTool(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// GetTool returns a tool by name
func (r *Registry) GetTool(name string) (Tool, ToolHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if rt, ok := r.tools[name]; ok {
		return rt.definition, rt.handler, true
	}
	return Tool{}, nil, false
}

// ListTools returns all registered tools
func (r *Registry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, rt := range r.tools {
		tools = append(tools, rt.definition)
	}
	return tools
}

// RegisterResource adds a static resource to the registry
func (r *Registry) RegisterResource(resource Resource, handler ResourceHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if resource.URI == "" {
		return fmt.Errorf("resource URI is required")
	}
	if handler == nil {
		return fmt.Errorf("resource handler is required")
	}
	if _, exists := r.resources[resource.URI]; exists {
		return fmt.Errorf("resource %s already registered", resource.URI)
	}

	r.resources[resource.URI] = registeredResource{
		definition: resource,
		handler:    handler,
	}
	return nil
}

// UnregisterResource removes a resource from the registry
func (r *Registry) UnregisterResource(uri string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.resources, uri)
}

// GetResource returns a resource by URI
func (r *Registry) GetResource(uri string) (Resource, ResourceHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if rr, ok := r.resources[uri]; ok {
		return rr.definition, rr.handler, true
	}
	return Resource{}, nil, false
}

// ListResources returns all registered resources
func (r *Registry) ListResources() []Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resources := make([]Resource, 0, len(r.resources))
	for _, rr := range r.resources {
		resources = append(resources, rr.definition)
	}
	return resources
}

// RegisterResourceTemplate adds a parameterized resource template
func (r *Registry) RegisterResourceTemplate(template ResourceTemplate, handler ResourceHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if template.URITemplate == "" {
		return fmt.Errorf("resource URI template is required")
	}
	if handler == nil {
		return fmt.Errorf("resource handler is required")
	}
	if _, exists := r.templates[template.URITemplate]; exists {
		return fmt.Errorf("template %s already registered", template.URITemplate)
	}

	r.templates[template.URITemplate] = registeredTemplate{
		definition: template,
		handler:    handler,
	}
	return nil
}

// ListResourceTemplates returns all registered resource templates
func (r *Registry) ListResourceTemplates() []ResourceTemplate {
	r.mu.RLock()
	defer r.mu.RUnlock()

	templates := make([]ResourceTemplate, 0, len(r.templates))
	for _, rt := range r.templates {
		templates = append(templates, rt.definition)
	}
	return templates
}

// ToolCount returns number of registered tools
func (r *Registry) ToolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// ResourceCount returns number of registered resources
func (r *Registry) ResourceCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.resources) + len(r.templates)
}
