// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

func (m *mapper) customFieldsTo(target *types.Dynamic) {
	if cf, ok := m.v["customFields"].(map[string]any); ok {
		if len(cf) == 0 {
			return
		}
		translated := make(map[string]attr.Value)
		for k, v := range cf {
			switch v := v.(type) {
			case string:
				translated[k] = types.DynamicValue(types.StringValue(v))
			case float64:
				translated[k] = types.DynamicValue(types.Float64Value(v))
			default:
				m.diagnostics.AddError("Invalid custom field value", fmt.Sprintf("Custom field %s has an invalid value type: %T", k, v))
			}
		}
		switch target.UnderlyingValue().(type) {
		case types.Map:
			mv, diag := types.MapValue(types.DynamicType, translated)
			m.diagnostics.Append(diag...)
			*target = types.DynamicValue(mv)
		case types.Object:
			typ := make(map[string]attr.Type)
			for k := range translated {
				typ[k] = types.DynamicType
			}
			ov, diag := types.ObjectValue(typ, translated)
			m.diagnostics.Append(diag...)
			*target = types.DynamicValue(ov)
		default:
			mv, diag := types.MapValue(types.DynamicType, translated)
			m.diagnostics.Append(diag...)
			*target = types.DynamicValue(mv)
		}
	}
}

func (m *mapper) from(source unknowable, target string) {
	if source.IsUnknown() || source.IsNull() {
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

func (m *mapper) customFieldsFrom(source types.Dynamic) {
	if !source.IsUnknown() {
		customFields := make(map[string]any)
		if !source.IsNull() && !source.IsUnderlyingValueNull() {

			var elements map[string]attr.Value
			switch source := source.UnderlyingValue().(type) {
			case types.Map:
				elements = source.Elements()
			case types.Object:
				elements = source.Attributes()
			default:
				m.diagnostics.AddError("Invalid custom fields", fmt.Sprintf("Custom fields must be a map, not %T", source))
			}

			var convertMapValue func(v attr.Value) any
			convertMapValue = func(v attr.Value) any {
				switch v := v.(type) {
				case types.String:
					return v.ValueString()
				case types.Float32:
					return v.ValueFloat32()
				case types.Float64:
					return v.ValueFloat64()
				case types.Int32:
					return v.ValueInt32()
				case types.Int64:
					return v.ValueInt64()
				case types.Dynamic:
					return convertMapValue(v.UnderlyingValue())
				default:
					m.diagnostics.AddError("Invalid custom field value", fmt.Sprintf("Custom field has an invalid value type: %T, must be a string or number", v))
					return nil
				}
			}

			for k, v := range elements {
				customFields[k] = convertMapValue(v)
			}
		}
		m.v["customFields"] = customFields
	}
}

type idable[T any] interface {
	*T

	GetId() types.String
}

func genericCreate[T any](ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, client *m3terClient, path, name string, read func(context.Context, *T, map[string]any, *diag.Diagnostics), write func(context.Context, *T, map[string]any, *diag.Diagnostics)) {
	var data T

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	restData := make(map[string]any)
	write(ctx, &data, restData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var updatedRestData map[string]any
	err := client.execute(ctx, "POST", path, nil, restData, &updatedRestData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create %s, got error: %s", name, err))
	}

	read(ctx, &data, updatedRestData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func genericRead[T any, PT idable[T]](ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse, client *m3terClient, path, name string, read func(context.Context, PT, map[string]any, *diag.Diagnostics)) {
	var data T

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var restData map[string]any
	err := client.execute(ctx, "GET", path+"/"+url.PathEscape(PT(&data).GetId().ValueString()), nil, nil, &restData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read %s, got error: %s", name, err))
		return
	}

	read(ctx, &data, restData, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func genericUpdate[T any, PT idable[T]](ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, client *m3terClient, path, name string, read func(context.Context, PT, map[string]any, *diag.Diagnostics), write func(context.Context, PT, map[string]any, *diag.Diagnostics)) {
	var data T

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var restData map[string]any
	err := client.execute(ctx, "GET", path+"/"+url.PathEscape(PT(&data).GetId().ValueString()), nil, nil, &restData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read %s, got error: %s", name, err))
		return
	}

	write(ctx, &data, restData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var newRestData map[string]any
	err = client.execute(ctx, "PUT", path+"/"+url.PathEscape(PT(&data).GetId().ValueString()), nil, restData, &newRestData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update %s, got error: %s", name, err))
	}

	read(ctx, &data, newRestData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func genericDelete[T any, PT idable[T]](ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse, client *m3terClient, path, name string) {
	var data T
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.execute(ctx, "DELETE", path+"/"+url.PathEscape(PT(&data).GetId().ValueString()), nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete %s, got error: %s", name, err))
	}
}
