package processor

import (
	appctx "config-man/backend/internal/context"
	"config-man/backend/model"
)

func (p *Processor) Notifications(ctx appctx.RequestContext) []model.Notification {
	return p.store.ListNotifications(ctx.Actor.ID)
}
