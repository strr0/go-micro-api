package cmd

import (
	"go-micro.dev/v4/api/handler"
	"go-micro.dev/v4/api/resolver"
	"go-micro.dev/v4/api/router"
	"go-micro.dev/v4/api/server"
)

type Options struct {
	// For the Command Line itself
	Name        string
	Description string
	Version     string

	Server *server.Server

	Routers   map[string]func(...router.Option) router.Router
	Resolvers map[string]func(...resolver.Option) resolver.Resolver
	Handlers  map[string]func(...handler.Option) handler.Handler
}
