/*
 Copyright 2015 Crunchy Data Solutions, Inc.
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

package main

// dnsbridgeclient is meant to run on any Docker host that
// needs to register DNS entries for any started or stopped container
// command line options (required)
// DNSBRIDGE_SERVER_ADDR is on the -d command flag
// -r 192.168.56.103:11001
// DOCKER_HOST is on the -h command flag
// -h unix://var/run/docker.sock
import (
	"crunchy.com/dnsbridge"
	"errors"
	"flag"
	"fmt"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"strings"
	//"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	dockerapi "github.com/fsouza/go-dockerclient"
	"net/rpc"
	"time"
)

var MAX_TRIES = 3

const delaySeconds = 5
const delay = (delaySeconds * 1000) * time.Millisecond

var KUBE_URL string
var DNSBRIDGE_SERVER_ADDR string
var DOCKER_HOST string

//var rpcclient *rpc.Client

func init() {
	flag.StringVar(&KUBE_URL, "k", "http://192.168.0.106:8080", "address of Kube ex: http://192.168.56.106:8080")
	flag.StringVar(&DNSBRIDGE_SERVER_ADDR, "d", "", "address of DNS RPC example  192.168.56.103:11001")
	flag.StringVar(&DOCKER_HOST, "h", "unix:///var/run/docker.sock", "docker socket url")
	flag.Parse()
}

func main() {

	if DNSBRIDGE_SERVER_ADDR == "" {
		fmt.Printf("dnsbridge: ", errors.New("FATAL ERROR: -d <ipaddr:port> required.  dnsbridgeserver rpc bind addr"))
	}

	var dockerConnected = false
	fmt.Printf("DOCKER_HOST=[%s]\n", DOCKER_HOST)
	var tries = 0
	var docker *dockerapi.Client
	var err error
	for tries = 0; tries < MAX_TRIES; tries++ {
		docker, err = dockerapi.NewClient(DOCKER_HOST)
		err = docker.Ping()
		if err != nil {
			fmt.Println("could not ping docker host")
			fmt.Printf("sleeping and will retry in %d sec\n", delaySeconds)
			time.Sleep(delay)
		} else {
			fmt.Println("no err in connecting to docker")
			dockerConnected = true
			break
		}
	}
	if dockerConnected == false {
		fmt.Println("failing, could not connect to docker after retries")
		panic("cant connect to docker")
	}

	events := make(chan *dockerapi.APIEvents)
	assert(docker.AddEventListener(events))
	fmt.Println("dnsbridge: Listening for Docker events...")
	for msg := range events {
		switch msg.Status {
		//case "start", "create":
		case "start":
			fmt.Printf("event:%s -adding - ID=%s from=%s\n", msg.Status, msg.ID, msg.From)
			Action("add", msg.ID, docker)
		case "stop":
			fmt.Printf("event:%s -removing -ID=%s from=%s\n", msg.Status, msg.ID, msg.From)
			Action("delete", msg.ID, docker)
		case "destroy":
			fmt.Printf("event:%s -destroy- ID=%s \n", msg.Status, msg.ID)
			Action("destroy", msg.ID, docker)
		case "die":
			fmt.Printf("event:%s -die- ID=%s \n", msg.Status, msg.ID)
		default:
			fmt.Printf("event: %s\n", msg.Status)
		}
	}

}

// Action invokes the dnsbridgeserver RPC call to perform
// DNS maintenance.  It will attempt to connect to the DNS server
// a few times before giving up, this gives the DNS server bridge
// time to start.
func Action(action string, containerId string, docker *dockerapi.Client) {

	var tries = 0
	var dnsbridgeConnected = false
	var err error
	var client *rpc.Client
	for tries = 0; tries < MAX_TRIES; tries++ {
		client, err = rpc.DialHTTP("tcp", DNSBRIDGE_SERVER_ADDR)
		if err != nil || client == nil {
			fmt.Println("error in dialing:" + err.Error())
			tries++
			fmt.Printf("sleeping and will retry in %d ms\n", delay)
			time.Sleep(delay)
		} else {
			fmt.Printf("got connection to dnsbridge server")
			dnsbridgeConnected = true
			break
		}
	}
	if dnsbridgeConnected == false {
		fmt.Printf("can't connect to dnsbridge server after retries, shutting down")
		panic("giving up....cant connect to dnsbridge after several tries")
	}

	fmt.Printf("DNSBRIDGE_SERVER_ADDR=[%s]\n", DNSBRIDGE_SERVER_ADDR)

	//if we fail on inspection, that is ok because we might
	//be checking for a crufty container that no longer exists
	//due to docker being shutdown uncleanly
	var ipaddress = ""
	var args dnsbridge.Args
	var command dnsbridge.Command

	if action == "add" {
		container, dockerErr := docker.InspectContainer(containerId)
		if dockerErr != nil {
			fmt.Printf("dnsbridge: unable to inspect container:%s %s", containerId, dockerErr)
			return
		}
		fmt.Printf("newly added container name=[%s]\n", container.Name[1:])
		if strings.HasPrefix(container.Name[1:], "k8s_net.") {
			fmt.Printf("skipping newly added kube service name=[%s]\n", container.Name[1:])
			err = client.Close()
			return

		} else if strings.HasPrefix(container.Name[1:], "k8s_") {
			podName := getPodID(container.Name[1:])
			ipaddress := getPodIPAddress(podName)
			args = dnsbridge.Args{ipaddress, podName, containerId}
			fmt.Printf("adding kube name=[%s] ip=[%s] containerid=[%s]\n", podName, ipaddress, containerId)
		} else {

			fmt.Printf("IPAddress =[%s]\n", container.NetworkSettings.IPAddress)
			ipaddress = container.NetworkSettings.IPAddress
			args = dnsbridge.Args{ipaddress, container.Name[1:], containerId}
		}

		err = client.Call("Command.Add", args, &command)
		if err != nil {
			fmt.Println("update error: " + err.Error())
		} else {
			fmt.Println("command output: " + command.Output)
			fmt.Println(" ")
		}

		err = client.Close()
	} else {
		args = dnsbridge.Args{"", "", containerId}
		err = client.Call("Command.Delete", args, &command)
		if err != nil {
			fmt.Println("delete error: " + err.Error())
		} else {
			fmt.Println("command output: " + command.Output)
			fmt.Println(" ")
		}

		err = client.Close()
	}

}

func assert(err error) {
	if err != nil {
		fmt.Println("dnsbridge: ", err)
		panic("can't continue")
	}
}

func getPodID(fullname string) string {
	things := strings.Split(fullname, ".")
	parts := strings.Split(things[1], "_")
	return parts[1]
}

func getPodIPAddress(podName string) string {
	var c *client.Client
	c = client.NewOrDie(&client.Config{
		Host:    KUBE_URL,
		Version: "v1beta1",
	})
	if c != nil {
		fmt.Println("connection to kube started....")
	}

	var pod *api.Pod
	ns := api.NamespaceDefault
	pod, err3 := c.Pods(ns).Get(podName)
	if err3 != nil {
		fmt.Println(err3.Error())
		return ""
	}

	return pod.CurrentState.PodIP
}

func listPods(c *client.Client) {

	var podList *api.PodList
	ns := api.NamespaceDefault
	podList, err3 := c.Pods(ns).List(labels.Everything())
	if err3 != nil {
		fmt.Println(err3.Error())
	}

	var pods []api.Pod
	pods = podList.Items
	fmt.Printf("number of pods=%d\n", len(pods))
	for _, element := range pods {
		fmt.Printf("pod state=%s\n", element.CurrentState.Status)
		fmt.Printf("pod name=%s\n", element.Name)
		fmt.Printf("pod IP=%s\n", element.CurrentState.PodIP)
		fmt.Printf("pod Host=%s\n", element.CurrentState.Host)
	}

}
