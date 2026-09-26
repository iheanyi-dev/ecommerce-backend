package presentation

import (
	"go.uber.org/fx"
)

// Module provides the Storage presentation layer.
//
// The HTTP server lifecycle itself is owned by cmd/main.go, following the
// same composition pattern used by the Identity service.
var Module = fx.Module(
	"presentation",

	fx.Provide(
		NewRouter,
	),
)
