package app

import (
	"context"
	"log/slog"
)

func audit(
	ctx context.Context,
	principal string,
	action Action,
	target string,
	decision, outcome string,
) {
	targetField := "key"
	if action == ActionList {
		targetField = "namespace"
	}

	slog.InfoContext(ctx, "audit",
		"principal", principal,
		"action", string(action),
		targetField, target,
		"decision", decision,
		"outcome", outcome,
	)
}
