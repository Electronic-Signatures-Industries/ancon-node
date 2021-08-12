package main

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/osamingo/jsonrpc/v2"
)

type (
	EchoHandler struct{}
	EchoParams  struct {
		Name string `json:"name"`
	}
	EchoResult struct {
		Message string `json:"message"`
	}

	PositionalHandler struct{}
	PositionalParams  []int
	PositionalResult  struct {
		Message []int `json:"message"`
	}
)

func (h EchoHandler) ServeJSONRPC(c context.Context, params *json.RawMessage) (interface{}, *jsonrpc.Error) {

	var p EchoParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	return EchoResult{
		Message: "Hello, " + p.Name,
	}, nil
}

func (h PositionalHandler) ServeJSONRPC(c context.Context, params *json.RawMessage) (interface{}, **jsonrpc.Error) {

	var p PositionalParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, nil
	}

	return PositionalResult{
		Message: p,
	}, nil
}
