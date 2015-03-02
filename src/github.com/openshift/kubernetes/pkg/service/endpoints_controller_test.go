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

package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/registry/registrytest"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

func newPodList(count int) api.PodList {
	pods := []api.Pod{}
	for i := 0; i < count; i++ {
		pods = append(pods, api.Pod{
			JSONBase: api.JSONBase{
				ID:         fmt.Sprintf("pod%d", i),
				APIVersion: "v1beta1",
			},
			DesiredState: api.PodState{
				Manifest: api.ContainerManifest{
					Containers: []api.Container{
						{
							Ports: []api.Port{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
			CurrentState: api.PodState{
				PodIP: "1.2.3.4",
			},
		})
	}
	return api.PodList{
		JSONBase: api.JSONBase{APIVersion: "v1beta1", Kind: "PodList"},
		Items:    pods,
	}
}

func TestFindPort(t *testing.T) {
	manifest := api.ContainerManifest{
		Containers: []api.Container{
			{
				Ports: []api.Port{
					{
						Name:          "foo",
						ContainerPort: 8080,
						HostPort:      9090,
					},
					{
						Name:          "bar",
						ContainerPort: 8000,
						HostPort:      9000,
					},
				},
			},
		},
	}
	port, err := findPort(&manifest, util.IntOrString{Kind: util.IntstrString, StrVal: "foo"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected 8080, Got %d", port)
	}
	port, err = findPort(&manifest, util.IntOrString{Kind: util.IntstrString, StrVal: "bar"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if port != 8000 {
		t.Errorf("Expected 8000, Got %d", port)
	}
	port, err = findPort(&manifest, util.IntOrString{Kind: util.IntstrInt, IntVal: 8000})
	if port != 8000 {
		t.Errorf("Expected 8000, Got %d", port)
	}
	port, err = findPort(&manifest, util.IntOrString{Kind: util.IntstrInt, IntVal: 7000})
	if port != 7000 {
		t.Errorf("Expected 7000, Got %d", port)
	}
	port, err = findPort(&manifest, util.IntOrString{Kind: util.IntstrString, StrVal: "baz"})
	if err == nil {
		t.Error("unexpected non-error")
	}
	port, err = findPort(&manifest, util.IntOrString{Kind: util.IntstrString, StrVal: ""})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected 8080, Got %d", port)
	}
	port, err = findPort(&manifest, util.IntOrString{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected 8080, Got %d", port)
	}
}

type serverResponse struct {
	statusCode int
	obj        interface{}
}

func makeTestServer(t *testing.T, podResponse serverResponse, serviceResponse serverResponse) *httptest.Server {
	fakePodHandler := util.FakeHandler{
		StatusCode:   podResponse.statusCode,
		ResponseBody: util.EncodeJSON(podResponse.obj),
	}
	fakeServiceHandler := util.FakeHandler{
		StatusCode:   serviceResponse.statusCode,
		ResponseBody: util.EncodeJSON(serviceResponse.obj),
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1beta1/pods", &fakePodHandler)
	mux.Handle("/api/v1beta1/services", &fakeServiceHandler)
	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		t.Errorf("unexpected request: %v", req.RequestURI)
		res.WriteHeader(http.StatusNotFound)
	})
	return httptest.NewServer(mux)
}

func TestSyncEndpointsEmpty(t *testing.T) {
	testServer := makeTestServer(t,
		serverResponse{http.StatusOK, newPodList(0)},
		serverResponse{http.StatusOK, api.ServiceList{}})
	client := client.NewOrDie(testServer.URL, nil)
	serviceRegistry := registrytest.ServiceRegistry{}
	endpoints := NewEndpointController(&serviceRegistry, client)
	if err := endpoints.SyncServiceEndpoints(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSyncEndpointsError(t *testing.T) {
	testServer := makeTestServer(t,
		serverResponse{http.StatusOK, newPodList(0)},
		serverResponse{http.StatusInternalServerError, api.ServiceList{}})
	client := client.NewOrDie(testServer.URL, nil)
	serviceRegistry := registrytest.ServiceRegistry{
		Err: fmt.Errorf("test error"),
	}
	endpoints := NewEndpointController(&serviceRegistry, client)
	if err := endpoints.SyncServiceEndpoints(); err == nil {
		t.Errorf("unexpected non-error")
	}
}

func TestSyncEndpointsItems(t *testing.T) {
	serviceList := api.ServiceList{
		Items: []api.Service{
			{
				Selector: map[string]string{
					"foo": "bar",
				},
			},
		},
	}
	testServer := makeTestServer(t,
		serverResponse{http.StatusOK, newPodList(1)},
		serverResponse{http.StatusOK, serviceList})
	client := client.NewOrDie(testServer.URL, nil)
	serviceRegistry := registrytest.ServiceRegistry{}
	endpoints := NewEndpointController(&serviceRegistry, client)
	if err := endpoints.SyncServiceEndpoints(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(serviceRegistry.Endpoints.Endpoints) != 1 ||
		serviceRegistry.Endpoints.Endpoints[0] != "1.2.3.4:8080" {
		t.Errorf("Unexpected endpoints update: %#v", serviceRegistry.Endpoints)
	}
}

func TestSyncEndpointsPodError(t *testing.T) {
	serviceList := api.ServiceList{
		Items: []api.Service{
			{
				Selector: map[string]string{
					"foo": "bar",
				},
			},
		},
	}
	testServer := makeTestServer(t,
		serverResponse{http.StatusInternalServerError, api.PodList{}},
		serverResponse{http.StatusOK, serviceList})
	client := client.NewOrDie(testServer.URL, nil)
	serviceRegistry := registrytest.ServiceRegistry{
		List: api.ServiceList{
			Items: []api.Service{
				{
					Selector: map[string]string{
						"foo": "bar",
					},
				},
			},
		},
	}
	endpoints := NewEndpointController(&serviceRegistry, client)
	if err := endpoints.SyncServiceEndpoints(); err == nil {
		t.Error("Unexpected non-error")
	}
}
