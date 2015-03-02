/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package endpoint

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

// Registry is an interface for things that know how to store endpoints.
type Registry interface {
	ListEndpoints() (*api.EndpointsList, error)
	GetEndpoints(name string) (*api.Endpoints, error)
	WatchEndpoints(labels, fields labels.Selector, resourceVersion uint64) (watch.Interface, error)
	UpdateEndpoints(e api.Endpoints) error
}
