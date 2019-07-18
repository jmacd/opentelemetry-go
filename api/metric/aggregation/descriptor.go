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

package aggregation

import "go.opentelemetry.io/api/core"

type Operator int

const (
	None Operator = iota
	SUM
	COUNT
	MIN
	MAX
	LAST_VALUE
	DISTRIBUTION
)

type Descriptor struct {
	Operator Operator
	Keys     []core.Key
}

func Sum(keys ...core.Key) Descriptor {
	return Descriptor{
		Operator: SUM,
		Keys:     keys,
	}
}

func Count(keys ...core.Key) Descriptor {
	return Descriptor{
		Operator: COUNT,
		Keys:     keys,
	}
}

func Min(keys ...core.Key) Descriptor {
	return Descriptor{
		Operator: MIN,
		Keys:     keys,
	}
}

func Max(keys ...core.Key) Descriptor {
	return Descriptor{
		Operator: MAX,
		Keys:     keys,
	}
}

func LastValue(keys ...core.Key) Descriptor {
	return Descriptor{
		Operator: LAST_VALUE,
		Keys:     keys,
	}
}

func Distribution(keys ...core.Key) Descriptor {
	return Descriptor{
		Operator: SUM,
		Keys:     keys,
	}
}
