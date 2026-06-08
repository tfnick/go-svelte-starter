package usecase

import "context"

type Surface string

const (
	SurfaceInternalAPI Surface = "api"
	SurfaceOpenAPI     Surface = "open-api"
	SurfaceSystem      Surface = "system"
)

type Context struct {
	Context   context.Context
	Surface   Surface
	RequestID string
	Actor     ActorContext
	Consumer  ConsumerContext
}

type ActorContext struct {
	Authenticated bool
	UserID        string
	Name          string
	Email         string
	IsAdmin       bool
}

type ConsumerContext struct {
	Authenticated bool
	KeyID         string
	PartnerID     string
	AccountID     string
	Environment   string
	Scopes        []string
}

func NewContext(ctx context.Context, surface Surface) Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return Context{
		Context: ctx,
		Surface: surface,
	}
}

func (c Context) Std() context.Context {
	if c.Context == nil {
		return context.Background()
	}
	return c.Context
}

func (c Context) WithStd(ctx context.Context) Context {
	if ctx == nil {
		ctx = context.Background()
	}
	c.Context = ctx
	return c
}

func (c Context) HasConsumerScope(scope string) bool {
	for _, candidate := range c.Consumer.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}
