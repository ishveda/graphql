package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/graphql-go/graphql/gqlerrors"
)

// Plugin is an interface for custom post-processing based on the execution context
// all the pre-processing happens in Resolve*()
// plugins differs from extensions by the time of execution and by how they modify the result
// in resolve func plugin analyzes execution context for each field in order to collect information of
// what to process during execution
// execution happens once query resolve is fully finished
type Plugin interface {
	// Name returns name of the plugin
	Name() string
	// IsCompatible tests whether current field is compatible with the plugin
	IsCompatible(ctx context.Context, i ResolveInfo) bool
	// Execute runs plugin processing on the data accessed by provided json pointer
	Execute(ctx context.Context, pointer string, data interface{}, i ResolveInfo) (interface{}, error)
}

func handlePluginsResolveFieldFinished(eCtx *executionContext, info ResolveInfo) {
	for _, p := range info.Schema.plugins {
		if p.IsCompatible(eCtx.Context, info) {
			eCtx.PluginExecRegistry.Register(PluginExecutable{
				info:   info,
				plugin: p,
			})
		}
	}
}

type PluginExecutionRegistry struct {
	plugins []PluginExecutable
}

func NewPluginExecRegistry() *PluginExecutionRegistry {
	return &PluginExecutionRegistry{
		plugins: make([]PluginExecutable, 0),
	}
}

type PluginExecutable struct {
	info   ResolveInfo
	plugin Plugin
}

func (pr *PluginExecutionRegistry) Register(pe PluginExecutable) {
	pr.plugins = append(pr.plugins, pe)
}

func (pr *PluginExecutionRegistry) Execute(ctx context.Context, data interface{}) (interface{}, []gqlerrors.FormattedError) {
	var plgErrs []gqlerrors.FormattedError
	var err error
	for _, p := range pr.plugins {
		elPath := constructPointer(p.info.Path.AsArray())
		data, err = p.plugin.Execute(ctx, elPath, data, p.info)
		if err != nil {
			plgErrs = append(plgErrs, gqlerrors.FormatError(
				fmt.Errorf("%s.PluginExecution: %v", p.plugin.Name(), err)))
		}
	}

	return data, plgErrs
}

func constructPointer(path []interface{}) string {
	var buf = strings.Builder{}
	buf.WriteString("/")
	for i := 0; i < len(path); i++ {
		buf.WriteString(fmt.Sprintf("%v", path[i]))
		if i != len(path)-1 {
			buf.WriteString("/")
		}
	}
	return buf.String()
}
