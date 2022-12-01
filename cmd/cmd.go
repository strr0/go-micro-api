package cmd

import (
	"github.com/urfave/cli/v2"
	"go-micro.dev/v4/api/handler"
	"go-micro.dev/v4/api/handler/api"
	"go-micro.dev/v4/api/handler/event"
	"go-micro.dev/v4/api/handler/http"
	"go-micro.dev/v4/api/handler/rpc"
	"go-micro.dev/v4/api/handler/web"
	"go-micro.dev/v4/api/resolver"
	"go-micro.dev/v4/api/resolver/grpc"
	"go-micro.dev/v4/api/resolver/host"
	"go-micro.dev/v4/api/resolver/path"
	"go-micro.dev/v4/api/resolver/vpath"
	"go-micro.dev/v4/api/router"
	"go-micro.dev/v4/api/router/registry"
	"go-micro.dev/v4/api/router/static"
	httpServer "go-micro.dev/v4/api/server/http"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type Cmd interface {
	// The cli app within this cmd
	App() *cli.App
	// Adds options, parses flags and initialize
	// exits on error
	Init(opts ...Option) error
	// Options set within this command
	Options() Options
}

type cmd struct {
	opts Options
	app  *cli.App
}

type Option func(o *Options)

var (
	DefaultCmd   = newCmd()
	DefaultFlags = []cli.Flag{
		&cli.StringFlag{
			Name:  "server_address",
			Value: ":8080",
			Usage: "--server_address=[server_address]",
		},
		&cli.StringFlag{
			Name:  "namespace",
			Value: "go.micro",
			Usage: "--namespace=[namespace]",
		},
		&cli.StringFlag{
			Name:  "router",
			Value: "registry",
			Usage: "--router=[router]",
		},
		&cli.StringFlag{
			Name:  "resolver",
			Value: "vpath",
			Usage: "--resolver=[resolver]",
		},
		&cli.StringFlag{
			Name:  "handler",
			Value: "rpc",
			Usage: "--handler",
		},
	}
	DefaultRouters = map[string]func(...router.Option) router.Router{
		"registry": func(option ...router.Option) router.Router {
			return registry.NewRouter(option...)
		},
		"static": func(option ...router.Option) router.Router {
			return static.NewRouter(option...)
		},
	}
	DefaultResolvers = map[string]func(...resolver.Option) resolver.Resolver{
		"grpc": func(option ...resolver.Option) resolver.Resolver {
			return grpc.NewResolver(option...)
		},
		"host": func(option ...resolver.Option) resolver.Resolver {
			return host.NewResolver(option...)
		},
		"path": func(option ...resolver.Option) resolver.Resolver {
			return path.NewResolver(option...)
		},
		"vpath": func(option ...resolver.Option) resolver.Resolver {
			return vpath.NewResolver(option...)
		},
	}
	DefaultHandlers = map[string]func(...handler.Option) handler.Handler{
		"api": func(option ...handler.Option) handler.Handler {
			return api.NewHandler(option...)
		},
		"event": func(option ...handler.Option) handler.Handler {
			return event.NewHandler(option...)
		},
		"http": func(option ...handler.Option) handler.Handler {
			return http.NewHandler(option...)
		},
		"rpc": func(option ...handler.Option) handler.Handler {
			return rpc.NewHandler(option...)
		},
		"web": func(option ...handler.Option) handler.Handler {
			return web.NewHandler(option...)
		},
	}
)

func newCmd(opts ...Option) Cmd {
	options := Options{
		Routers:   DefaultRouters,
		Resolvers: DefaultResolvers,
		Handlers:  DefaultHandlers,
	}
	for _, o := range opts {
		o(&options)
	}
	if len(options.Description) == 0 {
		options.Description = "a go-micro-api service"
	}
	cmd := new(cmd)
	cmd.opts = options
	cmd.app = cli.NewApp()
	cmd.app.Name = cmd.opts.Name
	cmd.app.Version = cmd.opts.Version
	cmd.app.Usage = cmd.opts.Description
	cmd.app.Before = cmd.Before
	cmd.app.Flags = DefaultFlags
	cmd.app.Action = cmd.Action
	return cmd
}

func (c *cmd) App() *cli.App {
	return c.app
}

func (c *cmd) Options() Options {
	return c.opts
}

func (c *cmd) Before(ctx *cli.Context) error {
	var routerOpts []router.Option
	var resolverOpts []resolver.Option
	var handlerOpts []handler.Option

	var newRouter = registry.NewRouter
	var newResolver = vpath.NewResolver
	var newHandler = rpc.NewHandler

	var address = ":8080"
	var newServer = httpServer.NewServer

	if arg := ctx.String("server_address"); len(arg) > 0 {
		address = arg
	}

	if arg := ctx.String("namespace"); len(arg) > 0 {
		resolverOpts = append(resolverOpts, resolver.WithNamespace(resolver.StaticNamespace(arg)))
	}

	if arg := ctx.String("router"); len(arg) > 0 {
		if r, ok := c.opts.Routers[arg]; ok {
			newRouter = r
		} else {
			log.Fatalf("Router %v is not found", arg)
		}
	}

	if arg := ctx.String("handler"); len(arg) > 0 {
		resolverOpts = append(resolverOpts, resolver.WithHandler(arg))
		if h, ok := c.opts.Handlers[arg]; ok {
			newHandler = h
		} else {
			log.Fatalf("Handler %v is not found", arg)
		}
	}

	if arg := ctx.String("resolver"); len(arg) > 0 {
		if r, ok := c.opts.Resolvers[arg]; ok {
			newResolver = r
		} else {
			log.Fatalf("Resolver %v is not found", arg)
		}
	}

	routerOpts = append(routerOpts, router.WithResolver(newResolver(resolverOpts...)))
	handlerOpts = append(handlerOpts, handler.WithRouter(newRouter(routerOpts...)))
	hdlr := newHandler(handlerOpts...)
	srv := newServer(address)
	srv.Handle("/", CorsMiddleware(hdlr))
	c.opts.Server = &srv

	return nil
}

func (c *cmd) Action(ctx *cli.Context) error {
	if err := (*c.opts.Server).Start(); err != nil {
		return err
	}

	// wait to finish
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := (*c.opts.Server).Stop(); err != nil {
		return err
	}

	return nil
}

func (c *cmd) Init(opts ...Option) error {
	for _, o := range opts {
		o(&c.opts)
	}
	if len(c.opts.Name) > 0 {
		c.app.Name = c.opts.Name
	}
	if len(c.opts.Version) > 0 {
		c.app.Version = c.opts.Version
	}
	c.app.HideVersion = len(c.opts.Version) == 0
	c.app.Usage = c.opts.Description
	c.app.RunAndExitOnError()
	return nil
}

func Run() {
	if err := DefaultCmd.App().Run(os.Args); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}
