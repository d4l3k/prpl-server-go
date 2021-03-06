package prpl

import (
	"net/http"

	"github.com/ua-parser/uap-go/uaparser"
)

type (
	// prpl is an instance of the prpl-server service
	PRPL struct {
		http.Handler
		parser         *uaparser.Parser
		config         *ProjectConfig
		builds         builds
		root           http.Dir
		routes         Routes
		version        string
		staticHandlers map[string]http.Handler
		createTemplate createTemplateFn
		shouldPush     func(r *http.Request) bool
	}

	// optionFn provides functional option configuration
	optionFn func(*PRPL) error
)

// New creates a new prpl instance
func New(options ...optionFn) (*PRPL, error) {
	p := PRPL{
		parser:         uaparser.NewFromSaved(),
		root:           http.Dir("."),
		version:        "/static/",
		staticHandlers: make(map[string]http.Handler),
		createTemplate: createDefaultTemplate,
	}

	for _, option := range options {
		if err := option(&p); err != nil {
			return nil, err
		}
	}

	// use polymer.json for build file by default
	if p.config == nil {
		if err := WithConfigFile("polymer.json")(&p); err != nil {
			return nil, err
		}
	}

	// TODO: pass p in rather than all the properties
	p.builds = loadBuilds(p.config, p.root, p.routes, p.version, p.createTemplate)

	p.Handler = p.createHandler()

	return &p, nil
}

// TODO: provide options to auto-create the version
// based on last modified timestamp or content hash

// WithVersion sets the version prefix
func WithVersion(version string) optionFn {
	return func(p *PRPL) error {
		p.version = "/" + version + "/"
		return nil
	}
}

// WithRoutes sets the route -> fragment mapping
func WithRoutes(routes Routes) optionFn {
	return func(p *PRPL) error {
		p.routes = routes
		return nil
	}
}

// WithRoot sets the root directory
func WithRoot(root http.Dir) optionFn {
	return func(p *PRPL) error {
		p.root = root
		return nil
	}
}

// WithConfig sets the project configuration
func WithConfig(config *ProjectConfig) optionFn {
	return func(p *PRPL) error {
		p.config = config
		return nil
	}
}

// WithConfigFile loads the project configuration
func WithConfigFile(filename string) optionFn {
	return func(p *PRPL) error {
		config, err := ConfigFromFile(filename)
		if err != nil {
			return err
		}
		p.config = config
		return nil
	}
}

// WithUAParserFile allows the uaparser configuration
// to be overriden from the inbuilt settings
func WithUAParserFile(regexFile string) optionFn {
	return func(p *PRPL) error {
		parser, err := uaparser.New(regexFile)
		if err != nil {
			return err
		}
		p.parser = parser
		return nil
	}
}

// WithUAParserBytes allows the uaparser configuration
// to be overriden from the inbuilt settings
func WithUAParserBytes(data []byte) optionFn {
	return func(p *PRPL) error {
		parser, err := uaparser.NewFromBytes(data)
		if err != nil {
			return err
		}
		p.parser = parser
		return nil
	}
}

// WithStaticHandler allows the handler for certain static
// files to be overridden. This could be used to customize
// the manifest.json file per tenant or to serve specific
// images based on host headers etc ...
func WithStaticHandler(path string, handler http.Handler) optionFn {
	return func(p *PRPL) error {
		p.staticHandlers[path] = handler
		return nil
	}
}

// WithRouteTemplate allows the entrypoint to be converted
// into a template so that the output can be transformed if
// required
func WithRouteTemplate(factory createTemplateFn) optionFn {
	return func(p *PRPL) error {
		p.createTemplate = factory
		return nil
	}
}

// WithShouldPush specifies when the server should do a direct HTTP server push
// instead of just setting the server push Link header.
func WithShouldPush(shouldPush func(*http.Request) bool) optionFn {
	return func(p *PRPL) error {
		p.shouldPush = shouldPush
		return nil
	}
}
