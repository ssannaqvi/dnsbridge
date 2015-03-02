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

package kubecfg

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/latest"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/golang/glog"
	"gopkg.in/v1/yaml"
)

// ResourcePrinter is an interface that knows how to print API resources.
type ResourcePrinter interface {
	// Print receives an arbitrary JSON body, formats it and prints it to a writer.
	Print([]byte, io.Writer) error
	PrintObj(runtime.Object, io.Writer) error
}

// IdentityPrinter is an implementation of ResourcePrinter which simply copies the body out to the output stream.
type IdentityPrinter struct{}

// Print is an implementation of ResourcePrinter.Print which simply writes the data to the Writer.
func (i *IdentityPrinter) Print(data []byte, w io.Writer) error {
	_, err := w.Write(data)
	return err
}

// PrintObj is an implementation of ResourcePrinter.PrintObj which simply writes the object to the Writer.
func (i *IdentityPrinter) PrintObj(obj runtime.Object, output io.Writer) error {
	data, err := latest.Codec.Encode(obj)
	if err != nil {
		return err
	}
	return i.Print(data, output)
}

// YAMLPrinter is an implementation of ResourcePrinter which parsess JSON, and re-formats as YAML.
type YAMLPrinter struct{}

// Print parses the data as JSON, re-formats as YAML and prints the YAML.
func (y *YAMLPrinter) Print(data []byte, w io.Writer) error {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	output, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(output))
	return err
}

// PrintObj prints the data as YAML.
func (y *YAMLPrinter) PrintObj(obj runtime.Object, w io.Writer) error {
	output, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(output))
	return err
}

type handlerEntry struct {
	columns   []string
	printFunc reflect.Value
}

// HumanReadablePrinter is an implementation of ResourcePrinter which attempts to provide more elegant output.
type HumanReadablePrinter struct {
	handlerMap map[reflect.Type]*handlerEntry
}

// NewHumanReadablePrinter creates a HumanReadablePrinter.
func NewHumanReadablePrinter() *HumanReadablePrinter {
	printer := &HumanReadablePrinter{make(map[reflect.Type]*handlerEntry)}
	printer.addDefaultHandlers()
	return printer
}

// Handler adds a print handler with a given set of columns to HumanReadablePrinter instance.
// printFunc is the function that will be called to print an object.
// It must be of the following type:
//  func printFunc(object ObjectType, w io.Writer) error
// where ObjectType is the type of the object that will be printed.
func (h *HumanReadablePrinter) Handler(columns []string, printFunc interface{}) error {
	printFuncValue := reflect.ValueOf(printFunc)
	if err := h.validatePrintHandlerFunc(printFuncValue); err != nil {
		glog.Errorf("Unable to add print handler: %v", err)
		return err
	}
	objType := printFuncValue.Type().In(0)
	h.handlerMap[objType] = &handlerEntry{
		columns:   columns,
		printFunc: printFuncValue,
	}
	return nil
}

func (h *HumanReadablePrinter) validatePrintHandlerFunc(printFunc reflect.Value) error {
	if printFunc.Kind() != reflect.Func {
		return fmt.Errorf("Invalid print handler. %#v is not a function.", printFunc)
	}
	funcType := printFunc.Type()
	if funcType.NumIn() != 2 || funcType.NumOut() != 1 {
		return fmt.Errorf("Invalid print handler." +
			"Must accept 2 parameters and return 1 value.")
	}
	if funcType.In(1) != reflect.TypeOf((*io.Writer)(nil)).Elem() ||
		funcType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("Invalid print handler. The expected signature is: "+
			"func handler(obj %v, w io.Writer) error", funcType.In(0))
	}
	return nil
}

var podColumns = []string{"ID", "Image(s)", "Host", "Labels", "Status"}
var replicationControllerColumns = []string{"ID", "Image(s)", "Selector", "Replicas"}
var serviceColumns = []string{"ID", "Labels", "Selector", "IP", "Port"}
var minionColumns = []string{"Minion identifier"}
var statusColumns = []string{"Status"}
var eventColumns = []string{"Name", "Kind", "Status", "Reason", "Message"}

// addDefaultHandlers adds print handlers for default Kubernetes types.
func (h *HumanReadablePrinter) addDefaultHandlers() {
	h.Handler(podColumns, printPod)
	h.Handler(podColumns, printPodList)
	h.Handler(replicationControllerColumns, printReplicationController)
	h.Handler(replicationControllerColumns, printReplicationControllerList)
	h.Handler(serviceColumns, printService)
	h.Handler(serviceColumns, printServiceList)
	h.Handler(minionColumns, printMinion)
	h.Handler(minionColumns, printMinionList)
	h.Handler(statusColumns, printStatus)
	h.Handler(eventColumns, printEvent)
	h.Handler(eventColumns, printEventList)
}

func (h *HumanReadablePrinter) unknown(data []byte, w io.Writer) error {
	_, err := fmt.Fprintf(w, "Unknown object: %s", string(data))
	return err
}

func (h *HumanReadablePrinter) printHeader(columnNames []string, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s\n", strings.Join(columnNames, "\t")); err != nil {
		return err
	}
	var lines []string
	for _ = range columnNames {
		lines = append(lines, "----------")
	}
	_, err := fmt.Fprintf(w, "%s\n", strings.Join(lines, "\t"))
	return err
}

func makeImageList(manifest api.ContainerManifest) string {
	var images []string
	for _, container := range manifest.Containers {
		images = append(images, container.Image)
	}
	return strings.Join(images, ",")
}

func podHostString(host, ip string) string {
	if host == "" && ip == "" {
		return "<unassigned>"
	}
	return host + "/" + ip
}

func printPod(pod *api.Pod, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		pod.ID, makeImageList(pod.DesiredState.Manifest),
		podHostString(pod.CurrentState.Host, pod.CurrentState.HostIP),
		labels.Set(pod.Labels), pod.CurrentState.Status)
	return err
}

func printPodList(podList *api.PodList, w io.Writer) error {
	for _, pod := range podList.Items {
		if err := printPod(&pod, w); err != nil {
			return err
		}
	}
	return nil
}

func printReplicationController(ctrl *api.ReplicationController, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
		ctrl.ID, makeImageList(ctrl.DesiredState.PodTemplate.DesiredState.Manifest),
		labels.Set(ctrl.DesiredState.ReplicaSelector), ctrl.DesiredState.Replicas)
	return err
}

func printReplicationControllerList(list *api.ReplicationControllerList, w io.Writer) error {
	for _, ctrl := range list.Items {
		if err := printReplicationController(&ctrl, w); err != nil {
			return err
		}
	}
	return nil
}

func printService(svc *api.Service, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", svc.ID, labels.Set(svc.Labels),
		labels.Set(svc.Selector), svc.PortalIP, svc.Port)
	return err
}

func printServiceList(list *api.ServiceList, w io.Writer) error {
	for _, svc := range list.Items {
		if err := printService(&svc, w); err != nil {
			return err
		}
	}
	return nil
}

func printMinion(minion *api.Minion, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s\n", minion.ID)
	return err
}

func printMinionList(list *api.MinionList, w io.Writer) error {
	for _, minion := range list.Items {
		if err := printMinion(&minion, w); err != nil {
			return err
		}
	}
	return nil
}

func printStatus(status *api.Status, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%v\n", status.Status)
	return err
}

func printEvent(event *api.Event, w io.Writer) error {
	_, err := fmt.Fprintf(
		w, "%s\t%s\t%s\t%s\t%s\n",
		event.InvolvedObject.Name,
		event.InvolvedObject.Kind,
		event.Status,
		event.Reason,
		event.Message,
	)
	return err
}

func printEventList(list *api.EventList, w io.Writer) error {
	for i := range list.Items {
		if err := printEvent(&list.Items[i], w); err != nil {
			return err
		}
	}
	return nil
}

// Print parses the data as JSON, then prints the parsed data in a human-friendly
// format according to the type of the data.
func (h *HumanReadablePrinter) Print(data []byte, output io.Writer) error {
	var mapObj map[string]runtime.Object
	if err := json.Unmarshal([]byte(data), &mapObj); err != nil {
		return err
	}

	if _, contains := mapObj["kind"]; !contains {
		return fmt.Errorf("unexpected object with no 'kind' field: %s", data)
	}

	obj, err := latest.Codec.Decode(data)
	if err != nil {
		return err
	}
	return h.PrintObj(obj, output)
}

// PrintObj prints the obj in a human-friendly format according to the type of the obj.
func (h *HumanReadablePrinter) PrintObj(obj runtime.Object, output io.Writer) error {
	w := tabwriter.NewWriter(output, 20, 5, 3, ' ', 0)
	defer w.Flush()
	if handler := h.handlerMap[reflect.TypeOf(obj)]; handler != nil {
		h.printHeader(handler.columns, w)
		args := []reflect.Value{reflect.ValueOf(obj), reflect.ValueOf(w)}
		resultValue := handler.printFunc.Call(args)[0]
		if resultValue.IsNil() {
			return nil
		} else {
			return resultValue.Interface().(error)
		}
	} else {
		return fmt.Errorf("Error: unknown type %#v", obj)
	}
}

// TemplatePrinter is an implementation of ResourcePrinter which formats data with a Go Template.
type TemplatePrinter struct {
	Template *template.Template
}

// Print parses the data as JSON, and re-formats it with the Go Template.
func (t *TemplatePrinter) Print(data []byte, w io.Writer) error {
	obj, err := latest.Codec.Decode(data)
	if err != nil {
		return err
	}
	return t.PrintObj(obj, w)
}

// PrintObj formats the obj with the Go Template.
func (t *TemplatePrinter) PrintObj(obj runtime.Object, w io.Writer) error {
	return t.Template.Execute(w, obj)
}
