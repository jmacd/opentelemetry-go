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

package loader

import (
	"fmt"
	"os"
	"plugin"
	"reflect"
	"sync"
)

type Flusher interface {
	Flush() error
}

var once sync.Once
var impl interface{}

func Load() interface{} {
	once.Do(func() {
		pluginName := os.Getenv("OPENTELEMETRY_LIB")
		if pluginName == "" {
			return
		}
		so, err := plugin.Open(pluginName)
		if err != nil {
			fmt.Println("Open failed", pluginName, err)
			return
		}
		implPtr, err := so.Lookup("Implementation")
		if err != nil {
			fmt.Println("Not an OTel implementation", pluginName, err)
			return
		}
		implRef := reflect.ValueOf(implPtr)
		if implRef.Type().Kind() != reflect.Ptr {
			fmt.Println("Invalid OTel implementation", pluginName, err)
			return
		}
		impl = implRef.Elem().Interface()
	})
	return impl
}

func Flush() error {
	if impl == nil {
		return nil
	}
	f, ok := impl.(Flusher)
	if !ok || f == nil {
		return nil
	}
	return f.Flush()
}
