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

package client

import (
	"encoding/json"
	"fmt"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/version"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

// Interface holds the methods for clients of Kubernetes,
// an interface to allow mock testing.
// TODO: these should return/take pointers.
type Interface interface {
	PodInterface
	ReplicationControllerInterface
	ServiceInterface
	VersionInterface
	MinionInterface
}

// PodInterface has methods to work with Pod resources.
type PodInterface interface {
	ListPods(ctx api.Context, selector labels.Selector) (*api.PodList, error)
	GetPod(ctx api.Context, id string) (*api.Pod, error)
	DeletePod(ctx api.Context, id string) error
	CreatePod(ctx api.Context, pod *api.Pod) (*api.Pod, error)
	UpdatePod(ctx api.Context, pod *api.Pod) (*api.Pod, error)
}

// ReplicationControllerInterface has methods to work with ReplicationController resources.
type ReplicationControllerInterface interface {
	ListReplicationControllers(ctx api.Context, selector labels.Selector) (*api.ReplicationControllerList, error)
	GetReplicationController(ctx api.Context, id string) (*api.ReplicationController, error)
	CreateReplicationController(ctx api.Context, ctrl *api.ReplicationController) (*api.ReplicationController, error)
	UpdateReplicationController(ctx api.Context, ctrl *api.ReplicationController) (*api.ReplicationController, error)
	DeleteReplicationController(ctx api.Context, id string) error
	WatchReplicationControllers(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error)
}

// ServiceInterface has methods to work with Service resources.
type ServiceInterface interface {
	ListServices(ctx api.Context, selector labels.Selector) (*api.ServiceList, error)
	GetService(ctx api.Context, id string) (*api.Service, error)
	CreateService(ctx api.Context, srv *api.Service) (*api.Service, error)
	UpdateService(ctx api.Context, srv *api.Service) (*api.Service, error)
	DeleteService(ctx api.Context, id string) error
	WatchServices(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error)
}

// EndpointsInterface has methods to work with Endpoints resources
type EndpointsInterface interface {
	ListEndpoints(ctx api.Context, selector labels.Selector) (*api.EndpointsList, error)
	GetEndpoints(ctx api.Context, id string) (*api.Endpoints, error)
	WatchEndpoints(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error)
}

// EventInterface has methods to work with Event resources
type EventInterface interface {
	CreateEvent(event *api.Event) (*api.Event, error)
	ListEvents(selector labels.Selector) (*api.EventList, error)
	GetEvent(id string) (*api.Event, error)
	WatchEvents(label, field labels.Selector, resourceVersion string) (watch.Interface, error)
}

// VersionInterface has a method to retrieve the server version.
type VersionInterface interface {
	ServerVersion() (*version.Info, error)
}

type MinionInterface interface {
	ListMinions() (*api.MinionList, error)
}

// APIStatus is exposed by errors that can be converted to an api.Status object
// for finer grained details.
type APIStatus interface {
	Status() api.Status
}

// Client is the implementation of a Kubernetes client.
type Client struct {
	*RESTClient
}

// ListPods takes a selector, and returns the list of pods that match that selector.
func (c *Client) ListPods(ctx api.Context, selector labels.Selector) (result *api.PodList, err error) {
	result = &api.PodList{}
	err = c.Get().Namespace(api.Namespace(ctx)).Path("pods").SelectorParam("labels", selector).Do().Into(result)
	return
}

// GetPod takes the id of the pod, and returns the corresponding Pod object, and an error if it occurs
func (c *Client) GetPod(ctx api.Context, id string) (result *api.Pod, err error) {
	result = &api.Pod{}
	err = c.Get().Namespace(api.Namespace(ctx)).Path("pods").Path(id).Do().Into(result)
	return
}

// DeletePod takes the id of the pod, and returns an error if one occurs
func (c *Client) DeletePod(ctx api.Context, id string) error {
	return c.Delete().Namespace(api.Namespace(ctx)).Path("pods").Path(id).Do().Error()
}

// CreatePod takes the representation of a pod.  Returns the server's representation of the pod, and an error, if it occurs.
func (c *Client) CreatePod(ctx api.Context, pod *api.Pod) (result *api.Pod, err error) {
	result = &api.Pod{}
	err = c.Post().Namespace(api.Namespace(ctx)).Path("pods").Body(pod).Do().Into(result)
	return
}

// UpdatePod takes the representation of a pod to update.  Returns the server's representation of the pod, and an error, if it occurs.
func (c *Client) UpdatePod(ctx api.Context, pod *api.Pod) (result *api.Pod, err error) {
	result = &api.Pod{}
	if len(pod.ResourceVersion) == 0 {
		err = fmt.Errorf("invalid update object, missing resource version: %v", pod)
		return
	}
	err = c.Put().Namespace(api.Namespace(ctx)).Path("pods").Path(pod.ID).Body(pod).Do().Into(result)
	return
}

// ListReplicationControllers takes a selector, and returns the list of replication controllers that match that selector.
func (c *Client) ListReplicationControllers(ctx api.Context, selector labels.Selector) (result *api.ReplicationControllerList, err error) {
	result = &api.ReplicationControllerList{}
	err = c.Get().Namespace(api.Namespace(ctx)).Path("replicationControllers").SelectorParam("labels", selector).Do().Into(result)
	return
}

// GetReplicationController returns information about a particular replication controller.
func (c *Client) GetReplicationController(ctx api.Context, id string) (result *api.ReplicationController, err error) {
	result = &api.ReplicationController{}
	err = c.Get().Namespace(api.Namespace(ctx)).Path("replicationControllers").Path(id).Do().Into(result)
	return
}

// CreateReplicationController creates a new replication controller.
func (c *Client) CreateReplicationController(ctx api.Context, controller *api.ReplicationController) (result *api.ReplicationController, err error) {
	result = &api.ReplicationController{}
	err = c.Post().Namespace(api.Namespace(ctx)).Path("replicationControllers").Body(controller).Do().Into(result)
	return
}

// UpdateReplicationController updates an existing replication controller.
func (c *Client) UpdateReplicationController(ctx api.Context, controller *api.ReplicationController) (result *api.ReplicationController, err error) {
	result = &api.ReplicationController{}
	if len(controller.ResourceVersion) == 0 {
		err = fmt.Errorf("invalid update object, missing resource version: %v", controller)
		return
	}
	err = c.Put().Namespace(api.Namespace(ctx)).Path("replicationControllers").Path(controller.ID).Body(controller).Do().Into(result)
	return
}

// DeleteReplicationController deletes an existing replication controller.
func (c *Client) DeleteReplicationController(ctx api.Context, id string) error {
	return c.Delete().Namespace(api.Namespace(ctx)).Path("replicationControllers").Path(id).Do().Error()
}

// WatchReplicationControllers returns a watch.Interface that watches the requested controllers.
func (c *Client) WatchReplicationControllers(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	return c.Get().
		Namespace(api.Namespace(ctx)).
		Path("watch").
		Path("replicationControllers").
		Param("resourceVersion", resourceVersion).
		SelectorParam("labels", label).
		SelectorParam("fields", field).
		Watch()
}

// ListServices takes a selector, and returns the list of services that match that selector
func (c *Client) ListServices(ctx api.Context, selector labels.Selector) (result *api.ServiceList, err error) {
	result = &api.ServiceList{}
	err = c.Get().Namespace(api.Namespace(ctx)).Path("services").SelectorParam("labels", selector).Do().Into(result)
	return
}

// GetService returns information about a particular service.
func (c *Client) GetService(ctx api.Context, id string) (result *api.Service, err error) {
	result = &api.Service{}
	err = c.Get().Namespace(api.Namespace(ctx)).Path("services").Path(id).Do().Into(result)
	return
}

// CreateService creates a new service.
func (c *Client) CreateService(ctx api.Context, svc *api.Service) (result *api.Service, err error) {
	result = &api.Service{}
	err = c.Post().Namespace(api.Namespace(ctx)).Path("services").Body(svc).Do().Into(result)
	return
}

// UpdateService updates an existing service.
func (c *Client) UpdateService(ctx api.Context, svc *api.Service) (result *api.Service, err error) {
	result = &api.Service{}
	if len(svc.ResourceVersion) == 0 {
		err = fmt.Errorf("invalid update object, missing resource version: %v", svc)
		return
	}
	err = c.Put().Namespace(api.Namespace(ctx)).Path("services").Path(svc.ID).Body(svc).Do().Into(result)
	return
}

// DeleteService deletes an existing service.
func (c *Client) DeleteService(ctx api.Context, id string) error {
	return c.Delete().Namespace(api.Namespace(ctx)).Path("services").Path(id).Do().Error()
}

// WatchServices returns a watch.Interface that watches the requested services.
func (c *Client) WatchServices(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	return c.Get().
		Namespace(api.Namespace(ctx)).
		Path("watch").
		Path("services").
		Param("resourceVersion", resourceVersion).
		SelectorParam("labels", label).
		SelectorParam("fields", field).
		Watch()
}

// ListEndpoints takes a selector, and returns the list of endpoints that match that selector
func (c *Client) ListEndpoints(ctx api.Context, selector labels.Selector) (result *api.EndpointsList, err error) {
	result = &api.EndpointsList{}
	err = c.Get().Namespace(api.Namespace(ctx)).Path("endpoints").SelectorParam("labels", selector).Do().Into(result)
	return
}

// GetEndpoints returns information about the endpoints for a particular service.
func (c *Client) GetEndpoints(ctx api.Context, id string) (result *api.Endpoints, err error) {
	result = &api.Endpoints{}
	err = c.Get().Namespace(api.Namespace(ctx)).Path("endpoints").Path(id).Do().Into(result)
	return
}

// WatchEndpoints returns a watch.Interface that watches the requested endpoints for a service.
func (c *Client) WatchEndpoints(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	return c.Get().
		Namespace(api.Namespace(ctx)).
		Path("watch").
		Path("endpoints").
		Param("resourceVersion", resourceVersion).
		SelectorParam("labels", label).
		SelectorParam("fields", field).
		Watch()
}

func (c *Client) CreateEndpoints(ctx api.Context, endpoints *api.Endpoints) (*api.Endpoints, error) {
	result := &api.Endpoints{}
	err := c.Post().Namespace(api.Namespace(ctx)).Path("endpoints").Body(endpoints).Do().Into(result)
	return result, err
}

func (c *Client) UpdateEndpoints(ctx api.Context, endpoints *api.Endpoints) (*api.Endpoints, error) {
	result := &api.Endpoints{}
	if len(endpoints.ResourceVersion) == 0 {
		return nil, fmt.Errorf("invalid update object, missing resource version: %v", endpoints)
	}
	err := c.Put().
		Namespace(api.Namespace(ctx)).
		Path("endpoints").
		Path(endpoints.ID).
		Body(endpoints).
		Do().
		Into(result)
	return result, err
}

// ServerVersion retrieves and parses the server's version.
func (c *Client) ServerVersion() (*version.Info, error) {
	body, err := c.Get().AbsPath("/version").Do().Raw()
	if err != nil {
		return nil, err
	}
	var info version.Info
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, fmt.Errorf("Got '%s': %v", string(body), err)
	}
	return &info, nil
}

// ListMinions lists all the minions in the cluster.
func (c *Client) ListMinions() (result *api.MinionList, err error) {
	result = &api.MinionList{}
	err = c.Get().Path("minions").Do().Into(result)
	return
}

func (c *Client) GetMinion(id string) (result *api.Minion, err error) {
	result = &api.Minion{}
	err = c.Get().Path("minions").Path(id).Do().Into(result)
	return
}

// CreateEvent makes a new event. Returns the copy of the event the server returns, or an error.
func (c *Client) CreateEvent(event *api.Event) (*api.Event, error) {
	result := &api.Event{}
	err := c.Post().Path("events").Body(event).Do().Into(result)
	return result, err
}

// ListEvents returns a list of events matching the selectors.
func (c *Client) ListEvents(label, field labels.Selector) (*api.EventList, error) {
	result := &api.EventList{}
	err := c.Get().
		Path("events").
		SelectorParam("labels", label).
		SelectorParam("fields", field).
		Do().
		Into(result)
	return result, err
}

// GetEvent returns the given event, or an error.
func (c *Client) GetEvent(id string) (*api.Event, error) {
	result := &api.Event{}
	err := c.Get().Path("events").Path(id).Do().Into(result)
	return result, err
}

// WatchEvents starts watching for events matching the given selectors.
func (c *Client) WatchEvents(label, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	return c.Get().
		Path("watch").
		Path("events").
		Param("resourceVersion", resourceVersion).
		SelectorParam("labels", label).
		SelectorParam("fields", field).
		Watch()
}
