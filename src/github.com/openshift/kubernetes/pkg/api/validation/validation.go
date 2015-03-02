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

package validation

import (
	"strings"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	errs "github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

func validateVolumes(volumes []api.Volume) (util.StringSet, errs.ErrorList) {
	allErrs := errs.ErrorList{}

	allNames := util.StringSet{}
	for i := range volumes {
		vol := &volumes[i] // so we can set default values
		el := errs.ErrorList{}
		// TODO(thockin) enforce that a source is set once we deprecate the implied form.
		if vol.Source != nil {
			el = validateSource(vol.Source).Prefix("source")
		}
		if len(vol.Name) == 0 {
			el = append(el, errs.NewFieldRequired("name", vol.Name))
		} else if !util.IsDNSLabel(vol.Name) {
			el = append(el, errs.NewFieldInvalid("name", vol.Name))
		} else if allNames.Has(vol.Name) {
			el = append(el, errs.NewFieldDuplicate("name", vol.Name))
		}
		if len(el) == 0 {
			allNames.Insert(vol.Name)
		} else {
			allErrs = append(allErrs, el.PrefixIndex(i)...)
		}
	}
	return allNames, allErrs
}

func validateSource(source *api.VolumeSource) errs.ErrorList {
	numVolumes := 0
	allErrs := errs.ErrorList{}
	if source.HostDirectory != nil {
		numVolumes++
		allErrs = append(allErrs, validateHostDir(source.HostDirectory).Prefix("hostDirectory")...)
	}
	if source.EmptyDirectory != nil {
		numVolumes++
		//EmptyDirs have nothing to validate
	}
	if numVolumes != 1 {
		allErrs = append(allErrs, errs.NewFieldInvalid("", source))
	}
	return allErrs
}

func validateHostDir(hostDir *api.HostDirectory) errs.ErrorList {
	allErrs := errs.ErrorList{}
	if hostDir.Path == "" {
		allErrs = append(allErrs, errs.NewNotFound("path", hostDir.Path))
	}
	return allErrs
}

var supportedPortProtocols = util.NewStringSet("TCP", "UDP")

func validatePorts(ports []api.Port) errs.ErrorList {
	allErrs := errs.ErrorList{}

	allNames := util.StringSet{}
	for i := range ports {
		pErrs := errs.ErrorList{}
		port := &ports[i] // so we can set default values
		if len(port.Name) > 0 {
			if len(port.Name) > 63 || !util.IsDNSLabel(port.Name) {
				pErrs = append(pErrs, errs.NewFieldInvalid("name", port.Name))
			} else if allNames.Has(port.Name) {
				pErrs = append(pErrs, errs.NewFieldDuplicate("name", port.Name))
			} else {
				allNames.Insert(port.Name)
			}
		}
		if port.ContainerPort == 0 {
			pErrs = append(pErrs, errs.NewFieldRequired("containerPort", port.ContainerPort))
		} else if !util.IsValidPortNum(port.ContainerPort) {
			pErrs = append(pErrs, errs.NewFieldInvalid("containerPort", port.ContainerPort))
		}
		if port.HostPort != 0 && !util.IsValidPortNum(port.HostPort) {
			pErrs = append(pErrs, errs.NewFieldInvalid("hostPort", port.HostPort))
		}
		if len(port.Protocol) == 0 {
			port.Protocol = "TCP"
		} else if !supportedPortProtocols.Has(strings.ToUpper(port.Protocol)) {
			pErrs = append(pErrs, errs.NewFieldNotSupported("protocol", port.Protocol))
		}
		allErrs = append(allErrs, pErrs.PrefixIndex(i)...)
	}
	return allErrs
}

func validateEnv(vars []api.EnvVar) errs.ErrorList {
	allErrs := errs.ErrorList{}

	for i := range vars {
		vErrs := errs.ErrorList{}
		ev := &vars[i] // so we can set default values
		if len(ev.Name) == 0 {
			vErrs = append(vErrs, errs.NewFieldRequired("name", ev.Name))
		}
		if !util.IsCIdentifier(ev.Name) {
			vErrs = append(vErrs, errs.NewFieldInvalid("name", ev.Name))
		}
		allErrs = append(allErrs, vErrs.PrefixIndex(i)...)
	}
	return allErrs
}

func validateVolumeMounts(mounts []api.VolumeMount, volumes util.StringSet) errs.ErrorList {
	allErrs := errs.ErrorList{}

	for i := range mounts {
		mErrs := errs.ErrorList{}
		mnt := &mounts[i] // so we can set default values
		if len(mnt.Name) == 0 {
			mErrs = append(mErrs, errs.NewFieldRequired("name", mnt.Name))
		} else if !volumes.Has(mnt.Name) {
			mErrs = append(mErrs, errs.NewNotFound("name", mnt.Name))
		}
		if len(mnt.MountPath) == 0 {
			mErrs = append(mErrs, errs.NewFieldRequired("mountPath", mnt.MountPath))
		}
		allErrs = append(allErrs, mErrs.PrefixIndex(i)...)
	}
	return allErrs
}

// AccumulateUniquePorts runs an extraction function on each Port of each Container,
// accumulating the results and returning an error if any ports conflict.
func AccumulateUniquePorts(containers []api.Container, accumulator map[int]bool, extract func(*api.Port) int) errs.ErrorList {
	allErrs := errs.ErrorList{}

	for ci := range containers {
		cErrs := errs.ErrorList{}
		ctr := &containers[ci]
		for pi := range ctr.Ports {
			port := extract(&ctr.Ports[pi])
			if port == 0 {
				continue
			}
			if accumulator[port] {
				cErrs = append(cErrs, errs.NewFieldDuplicate("Port", port))
			} else {
				accumulator[port] = true
			}
		}
		allErrs = append(allErrs, cErrs.PrefixIndex(ci)...)
	}
	return allErrs
}

// checkHostPortConflicts checks for colliding Port.HostPort values across
// a slice of containers.
func checkHostPortConflicts(containers []api.Container) errs.ErrorList {
	allPorts := map[int]bool{}
	return AccumulateUniquePorts(containers, allPorts, func(p *api.Port) int { return p.HostPort })
}

func validateContainers(containers []api.Container, volumes util.StringSet) errs.ErrorList {
	allErrs := errs.ErrorList{}

	allNames := util.StringSet{}
	for i := range containers {
		cErrs := errs.ErrorList{}
		ctr := &containers[i] // so we can set default values
		if len(ctr.Name) == 0 {
			cErrs = append(cErrs, errs.NewFieldRequired("name", ctr.Name))
		} else if !util.IsDNSLabel(ctr.Name) {
			cErrs = append(cErrs, errs.NewFieldInvalid("name", ctr.Name))
		} else if allNames.Has(ctr.Name) {
			cErrs = append(cErrs, errs.NewFieldDuplicate("name", ctr.Name))
		} else {
			allNames.Insert(ctr.Name)
		}
		if len(ctr.Image) == 0 {
			cErrs = append(cErrs, errs.NewFieldRequired("image", ctr.Image))
		}
		cErrs = append(cErrs, validatePorts(ctr.Ports).Prefix("ports")...)
		cErrs = append(cErrs, validateEnv(ctr.Env).Prefix("env")...)
		cErrs = append(cErrs, validateVolumeMounts(ctr.VolumeMounts, volumes).Prefix("volumeMounts")...)
		allErrs = append(allErrs, cErrs.PrefixIndex(i)...)
	}
	// Check for colliding ports across all containers.
	// TODO(thockin): This really is dependent on the network config of the host (IP per pod?)
	// and the config of the new manifest.  But we have not specced that out yet, so we'll just
	// make some assumptions for now.  As of now, pods share a network namespace, which means that
	// every Port.HostPort across the whole pod must be unique.
	allErrs = append(allErrs, checkHostPortConflicts(containers)...)

	return allErrs
}

var supportedManifestVersions = util.NewStringSet("v1beta1", "v1beta2")

// ValidateManifest tests that the specified ContainerManifest has valid data.
// This includes checking formatting and uniqueness.  It also canonicalizes the
// structure by setting default values and implementing any backwards-compatibility
// tricks.
func ValidateManifest(manifest *api.ContainerManifest) errs.ErrorList {
	allErrs := errs.ErrorList{}

	if len(manifest.Version) == 0 {
		allErrs = append(allErrs, errs.NewFieldRequired("version", manifest.Version))
	} else if !supportedManifestVersions.Has(strings.ToLower(manifest.Version)) {
		allErrs = append(allErrs, errs.NewFieldNotSupported("version", manifest.Version))
	}
	allVolumes, errs := validateVolumes(manifest.Volumes)
	allErrs = append(allErrs, errs.Prefix("volumes")...)
	allErrs = append(allErrs, validateContainers(manifest.Containers, allVolumes).Prefix("containers")...)
	return allErrs
}

func ValidatePodState(podState *api.PodState) errs.ErrorList {
	allErrs := errs.ErrorList(ValidateManifest(&podState.Manifest)).Prefix("manifest")
	if podState.RestartPolicy.Type == "" {
		podState.RestartPolicy.Type = api.RestartAlways
	} else if podState.RestartPolicy.Type != api.RestartAlways &&
		podState.RestartPolicy.Type != api.RestartOnFailure &&
		podState.RestartPolicy.Type != api.RestartNever {
		allErrs = append(allErrs, errs.NewFieldNotSupported("restartPolicy.type", podState.RestartPolicy.Type))
	}

	return allErrs
}

// ValidatePod tests if required fields in the pod are set.
func ValidatePod(pod *api.Pod) errs.ErrorList {
	allErrs := errs.ErrorList{}
	if len(pod.ID) == 0 {
		allErrs = append(allErrs, errs.NewFieldRequired("id", pod.ID))
	}
	allErrs = append(allErrs, ValidatePodState(&pod.DesiredState).Prefix("desiredState")...)
	return allErrs
}

// ValidateService tests if required fields in the service are set.
func ValidateService(service *api.Service) errs.ErrorList {
	allErrs := errs.ErrorList{}
	if len(service.ID) == 0 {
		allErrs = append(allErrs, errs.NewFieldRequired("id", service.ID))
	} else if !util.IsDNS952Label(service.ID) {
		allErrs = append(allErrs, errs.NewFieldInvalid("id", service.ID))
	}
	if !util.IsValidPortNum(service.Port) {
		allErrs = append(allErrs, errs.NewFieldInvalid("Service.Port", service.Port))
	}
	if labels.Set(service.Selector).AsSelector().Empty() {
		allErrs = append(allErrs, errs.NewFieldRequired("selector", service.Selector))
	}
	return allErrs
}

// ValidateReplicationController tests if required fields in the replication controller are set.
func ValidateReplicationController(controller *api.ReplicationController) errs.ErrorList {
	allErrs := errs.ErrorList{}
	if len(controller.ID) == 0 {
		allErrs = append(allErrs, errs.NewFieldRequired("id", controller.ID))
	}
	if labels.Set(controller.DesiredState.ReplicaSelector).AsSelector().Empty() {
		allErrs = append(allErrs, errs.NewFieldRequired("desiredState.replicaSelector", controller.DesiredState.ReplicaSelector))
	}
	selector := labels.Set(controller.DesiredState.ReplicaSelector).AsSelector()
	labels := labels.Set(controller.DesiredState.PodTemplate.Labels)
	if !selector.Matches(labels) {
		allErrs = append(allErrs, errs.NewFieldInvalid("desiredState.podTemplate.labels", controller.DesiredState.PodTemplate))
	}
	if controller.DesiredState.Replicas < 0 {
		allErrs = append(allErrs, errs.NewFieldInvalid("desiredState.replicas", controller.DesiredState.Replicas))
	}
	allErrs = append(allErrs, ValidateManifest(&controller.DesiredState.PodTemplate.DesiredState.Manifest).Prefix("desiredState.podTemplate.desiredState.manifest")...)
	return allErrs
}
