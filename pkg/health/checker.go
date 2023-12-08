package health

import (
	"attacknet/cmd/pkg/project"
	"context"
)

type HealthCheckerImpl struct {
}

func BuildHealthChecker(ctx *context.Context, cfg *project.ConfigParsed) *HealthChecker {
	return nil
}

//func (hc *HealthChecker)
