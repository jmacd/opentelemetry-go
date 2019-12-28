// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This test is too large for the race detector.  This SDK uses no locks
// that the race detector would help with, anyway.

package scope

import (
	"go.opentelemetry.io/otel/api/context/baggage"
	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

type (
	Scope struct {
		name     string
		provider Provider
	}

	Provider interface {
		Tracer() trace.Tracer
		Meter() metric.Meter
		Propagators() propagation.Propagators
	}

	provider struct {
		tracer      trace.Tracer
		meter       metric.Meter
		propagators propagation.Propagators
		resources   baggage.Map
	}
)

func NewProvider(t trace.Tracer, m metric.Meter, p propagation.Propagators, r baggage.Map) Provider {
	return &provider{
		tracer:      t,
		meter:       m,
		propagators: p,
		resources:   r,
	}
}

func New(name string, provider Provider) Scope {
	return Scope{
		name:     name,
		provider: provider,
	}
}

func (p *provider) Tracer() trace.Tracer {
	return p.tracer
}

func (p *provider) Meter() metric.Meter {
	return p.meter
}

func (p *provider) Propagators() propagation.Propagators {
	return p.propagators
}

func (s *Scope) Tracer() trace.Tracer {
	return s.provider.Tracer()
}

func (s *Scope) Meter() metric.Meter {
	return s.provider.Meter()
}

func (s *Scope) Propagators() propagation.Propagators {
	return s.provider.Propagators()
}
