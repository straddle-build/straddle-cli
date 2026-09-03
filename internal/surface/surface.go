// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

// Package surface is the contract-derived shape of one CLI command: what the
// operation accepts and where each input lands on the wire. The generator
// emits it, drift compares it, and the runtime binds flags from it.
package surface

type Kind string

const (
	KindString  Kind = "string"
	KindInteger Kind = "integer"
	KindNumber  Kind = "number"
	KindBoolean Kind = "boolean"
	KindJSON    Kind = "json"
)

type In string

const (
	InQuery  In = "query"
	InHeader In = "header"
	InBody   In = "body"
)

type Style string

const (
	StyleForm   Style = "form"
	StyleSimple Style = "simple"
)

type Flag struct {
	Name        string   `json:"name"`
	In          In       `json:"in"`
	Key         string   `json:"key"`
	Kind        Kind     `json:"kind"`
	Array       bool     `json:"array,omitempty"`
	Style       Style    `json:"style,omitempty"`
	Explode     bool     `json:"explode,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Description string   `json:"description,omitempty"`
	Format      string   `json:"format,omitempty"`
	Default     string   `json:"default,omitempty"`
}

type Surface struct {
	Endpoint             string   `json:"endpoint"`
	OperationID          string   `json:"operation_id"`
	Method               string   `json:"method"`
	Path                 string   `json:"path"`
	PathParams           []string `json:"path_params,omitempty"`
	Flags                []Flag   `json:"flags,omitempty"`
	HasBody              bool     `json:"has_body,omitempty"`
	BodyRequired         bool     `json:"body_required,omitempty"`
	AcceptsAccountHeader bool     `json:"accepts_account_header,omitempty"`
	ReadOnly             bool     `json:"read_only,omitempty"`
}
