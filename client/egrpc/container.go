package egrpc

import (
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/core/elog"
	"google.golang.org/grpc"
)

type Option func(c *Container)

type Container struct {
	config *Config
	name   string
	logger *elog.Component
}

func DefaultContainer() *Container {
	return &Container{
		config: DefaultConfig(),
		logger: elog.EgoLogger.With(elog.FieldComponent(PackageName)),
	}
}

func Load(key string) *Container {
	c := DefaultContainer()
	if err := econf.UnmarshalKey(key, &c.config); err != nil {
		c.logger.Panic("parse config error", elog.FieldErr(err), elog.FieldKey(key))
		return c
	}
	c.logger = c.logger.With(elog.FieldComponentName(key))
	c.name = key
	return c
}

// WithDialOption ...
func WithDialOption(opts ...grpc.DialOption) Option {
	return func(c *Container) {
		if c.config.dialOptions == nil {
			c.config.dialOptions = make([]grpc.DialOption, 0)
		}
		c.config.dialOptions = append(c.config.dialOptions, opts...)
	}
}

// Build ...
func (c *Container) Build(options ...Option) *Component {
	if options == nil {
		options = make([]Option, 0)
	}

	if c.config.Debug {
		options = append(options, WithDialOption(grpc.WithChainUnaryInterceptor(debugUnaryClientInterceptor(c.config.Address))))
	}

	if !c.config.DisableAidInterceptor {
		options = append(options, WithDialOption(grpc.WithChainUnaryInterceptor(aidUnaryClientInterceptor())))
	}

	if !c.config.DisableTimeoutInterceptor {
		options = append(options, WithDialOption(grpc.WithChainUnaryInterceptor(timeoutUnaryClientInterceptor(c.logger, c.config.ReadTimeout, c.config.SlowThreshold))))
	}

	if !c.config.DisableTraceInterceptor {
		options = append(options,
			WithDialOption(grpc.WithChainUnaryInterceptor(traceUnaryClientInterceptor())),
		)
	}

	if !c.config.DisableAccessInterceptor {
		options = append(options,
			WithDialOption(grpc.WithChainUnaryInterceptor(loggerUnaryClientInterceptor(c.logger, c.config.AccessInterceptorLevel))),
		)
	}

	if !c.config.DisableMetricInterceptor {
		options = append(options,
			WithDialOption(grpc.WithChainUnaryInterceptor(metricUnaryClientInterceptor(c.config.Name))),
		)
	}

	for _, option := range options {
		option(c)
	}

	c.logger.With(elog.FieldAddr(c.config.Address))
	return newComponent(c.name, c.config, c.logger)
}
