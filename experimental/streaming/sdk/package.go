package sdk

import (
	"go.opentelemetry.io/api/trace/global"
)

func init() {
	global.SetTracer(New())
}
