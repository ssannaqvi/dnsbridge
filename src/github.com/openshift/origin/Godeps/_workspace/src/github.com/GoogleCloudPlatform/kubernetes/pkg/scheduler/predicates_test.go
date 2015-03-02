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

package scheduler

import (
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/resources"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

type FakeNodeInfo api.Minion

func (n FakeNodeInfo) GetNodeInfo(nodeName string) (*api.Minion, error) {
	node := api.Minion(n)
	return &node, nil
}

func makeResources(milliCPU int, memory int) api.NodeResources {
	return api.NodeResources{
		Capacity: api.ResourceList{
			resources.CPU: util.IntOrString{
				IntVal: milliCPU,
				Kind:   util.IntstrInt,
			},
			resources.Memory: util.IntOrString{
				IntVal: memory,
				Kind:   util.IntstrInt,
			},
		},
	}
}

func newResourcePod(usage ...resourceRequest) api.Pod {
	containers := []api.Container{}
	for _, req := range usage {
		containers = append(containers, api.Container{
			Memory: req.memory,
			CPU:    req.milliCPU,
		})
	}
	return api.Pod{
		DesiredState: api.PodState{
			Manifest: api.ContainerManifest{
				Containers: containers,
			},
		},
	}
}

func TestPodFitsResources(t *testing.T) {
	tests := []struct {
		pod          api.Pod
		existingPods []api.Pod
		fits         bool
		test         string
	}{
		{
			pod: api.Pod{},
			existingPods: []api.Pod{
				newResourcePod(resourceRequest{milliCPU: 10, memory: 20}),
			},
			fits: true,
			test: "no resources requested always fits",
		},
		{
			pod: newResourcePod(resourceRequest{milliCPU: 1, memory: 1}),
			existingPods: []api.Pod{
				newResourcePod(resourceRequest{milliCPU: 10, memory: 20}),
			},
			fits: false,
			test: "too many resources fails",
		},
		{
			pod: newResourcePod(resourceRequest{milliCPU: 1, memory: 1}),
			existingPods: []api.Pod{
				newResourcePod(resourceRequest{milliCPU: 5, memory: 5}),
			},
			fits: true,
			test: "both resources fit",
		},
		{
			pod: newResourcePod(resourceRequest{milliCPU: 1, memory: 2}),
			existingPods: []api.Pod{
				newResourcePod(resourceRequest{milliCPU: 5, memory: 19}),
			},
			fits: false,
			test: "one resources fits",
		},
		{
			pod: newResourcePod(resourceRequest{milliCPU: 5, memory: 1}),
			existingPods: []api.Pod{
				newResourcePod(resourceRequest{milliCPU: 5, memory: 19}),
			},
			fits: true,
			test: "equal edge case",
		},
	}
	for _, test := range tests {
		node := api.Minion{NodeResources: makeResources(10, 20)}

		fit := ResourceFit{FakeNodeInfo(node)}
		fits, err := fit.PodFitsResources(test.pod, test.existingPods, "machine")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if fits != test.fits {
			t.Errorf("%s: expected: %v got %v", test.test, test.fits, fits)
		}
	}
}

func TestPodFitsPorts(t *testing.T) {
	tests := []struct {
		pod          api.Pod
		existingPods []api.Pod
		fits         bool
		test         string
	}{
		{
			pod:          api.Pod{},
			existingPods: []api.Pod{},
			fits:         true,
			test:         "nothing running",
		},
		{
			pod: newPod("m1", 8080),
			existingPods: []api.Pod{
				newPod("m1", 9090),
			},
			fits: true,
			test: "other port",
		},
		{
			pod: newPod("m1", 8080),
			existingPods: []api.Pod{
				newPod("m1", 8080),
			},
			fits: false,
			test: "same port",
		},
		{
			pod: newPod("m1", 8000, 8080),
			existingPods: []api.Pod{
				newPod("m1", 8080),
			},
			fits: false,
			test: "second port",
		},
		{
			pod: newPod("m1", 8000, 8080),
			existingPods: []api.Pod{
				newPod("m1", 8001, 8080),
			},
			fits: false,
			test: "second port",
		},
	}
	for _, test := range tests {
		fits, err := PodFitsPorts(test.pod, test.existingPods, "machine")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if test.fits != fits {
			t.Errorf("%s: expected %v, saw %v", test.test, test.fits, fits)
		}
	}
}

func TestDiskConflicts(t *testing.T) {
	volState := api.PodState{
		Manifest: api.ContainerManifest{
			Volumes: []api.Volume{
				{
					Source: &api.VolumeSource{
						GCEPersistentDisk: &api.GCEPersistentDisk{
							PDName: "foo",
						},
					},
				},
			},
		},
	}
	volState2 := api.PodState{
		Manifest: api.ContainerManifest{
			Volumes: []api.Volume{
				{
					Source: &api.VolumeSource{
						GCEPersistentDisk: &api.GCEPersistentDisk{
							PDName: "bar",
						},
					},
				},
			},
		},
	}
	tests := []struct {
		pod          api.Pod
		existingPods []api.Pod
		isOk         bool
		test         string
	}{
		{api.Pod{}, []api.Pod{}, true, "nothing"},
		{api.Pod{}, []api.Pod{{DesiredState: volState}}, true, "one state"},
		{api.Pod{DesiredState: volState}, []api.Pod{{DesiredState: volState}}, false, "same state"},
		{api.Pod{DesiredState: volState2}, []api.Pod{{DesiredState: volState}}, true, "different state"},
	}

	for _, test := range tests {
		ok, err := NoDiskConflict(test.pod, test.existingPods, "machine")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if test.isOk && !ok {
			t.Errorf("expected ok, got none.  %v %v %s", test.pod, test.existingPods, test.test)
		}
		if !test.isOk && ok {
			t.Errorf("expected no ok, got one.  %v %v %s", test.pod, test.existingPods, test.test)
		}
	}
}
