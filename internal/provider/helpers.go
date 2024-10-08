// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type mapper struct {
	ctx         context.Context
	diagnostics *diag.Diagnostics
	v           map[string]any
}

type attrTyped interface {
	Type(_ context.Context) attr.Type
}

type unknowable interface {
	IsUnknown() bool
	IsNull() bool
}

type stringValuer interface {
	ValueString() string
}

type int32Valuer interface {
	ValueInt32() int32
}

type int64Valuer interface {
	ValueInt64() int64
}

type float64Valuer interface {
	ValueFloat64() float64
}

type boolValuer interface {
	ValueBool() bool
}

func (m *mapper) to(key string, target attrTyped) {
	if v, ok := m.v[key]; ok {
		m.diagnostics.Append(tfsdk.ValueFrom(m.ctx, v, target.Type(m.ctx), target)...)
	}
}

func (m *mapper) listTo(key string, target *types.List, elemType attr.Type, fn func(any) (attr.Value, diag.Diagnostics)) {
	if v, ok := m.v[key]; ok {
		if v, ok := v.([]any); ok {
			var elements []attr.Value
			for _, e := range v {
				elem, diag := fn(e)
				m.diagnostics.Append(diag...)
				elements = append(elements, elem)
			}
			lv, diag := types.ListValue(elemType, elements)
			m.diagnostics.Append(diag...)
			*target = lv
		}
	}
}

func (m *mapper) from(source unknowable, target string) {
	if source.IsUnknown() {
		return
	}

	switch source := source.(type) {
	case stringValuer:
		m.v[target] = source.ValueString()
	case int32Valuer:
		m.v[target] = source.ValueInt32()
	case int64Valuer:
		m.v[target] = source.ValueInt64()
	case float64Valuer:
		m.v[target] = source.ValueFloat64()
	case boolValuer:
		m.v[target] = source.ValueBool()
	default:
		m.diagnostics.AddError("Cannot map field "+target, "unknown type")
	}
}

func (m *mapper) listFrom(source types.List, target string, fn func(v attr.Value) (any, diag.Diagnostics)) {
	if source.IsUnknown() {
		return
	}

	var v []any
	for _, e := range source.Elements() {
		elem, diag := fn(e)
		m.diagnostics.Append(diag...)
		v = append(v, elem)
	}
	m.v[target] = v
}
