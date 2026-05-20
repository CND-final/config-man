package context

import "config-man/backend/model"

type RequestContext struct {
	Actor model.User
}

func (ctx RequestContext) ActorName() string {
	return ctx.Actor.Name
}
