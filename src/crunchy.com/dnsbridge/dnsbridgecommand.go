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

package dnsbridge

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

type Args struct {
	IPAddress, Name, ContainerID string
}

type Command struct {
	Output string
}

// Delete executes the DNS delete logic on the DNS server.
// argument: IPAddress of the container
// argument: the name of the container
//
func (t *Command) Delete(args *Args, reply *Command) error {

	fmt.Println("Delete called IPAddress=" + args.IPAddress + " Name=" + args.Name + " ID=" + args.ContainerID)

	if args.ContainerID == "" {
		fmt.Println("ContainerID was nil")
		return errors.New("Arg ContainerID was nil")
	}

	var cmd *exec.Cmd
	var err error
	var b BridgeEntry
	var out bytes.Buffer
	var stderr bytes.Buffer

	fmt.Println("deleting the following:" + args.ContainerID)
	b, err = Get(args.ContainerID)
	if err != nil {
		fmt.Println(err.Error())
		reply.Output = err.Error()
		return nil
	} else {
		if b.Name != "" {
			fmt.Println("deleting the following:" + b.Name)
			err = Delete(args.ContainerID)
			cmd = exec.Command("/cluster/bin/delete-host.sh", b.Name)
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err = cmd.Run()
			if err != nil {
				fmt.Println("here at four")
				fmt.Println(err.Error())
				errorString := fmt.Sprintf("%s\n%s\n%s\n", err.Error(), out.String(), stderr.String())
				reply.Output = errorString
				return nil
			} else {
				reply.Output = out.String()
				return nil
			}
		} else {
			reply.Output = "success"
			return nil
		}
	}

	reply.Output = "error in delete invalid option"
	return nil

}

// Add executes the DNS add logic on the DNS server.
// argument: IPAddress of the container
// argument: the name of the container
//
func (t *Command) Add(args *Args, reply *Command) error {

	fmt.Println("Add called IPAddress=" + args.IPAddress + " Name=" + args.Name + " ID=" + args.ContainerID)

	var cmd *exec.Cmd
	var err error
	var b BridgeEntry
	var out bytes.Buffer
	var stderr bytes.Buffer

	if args.Name == "" {
		fmt.Println("Name was nil and not valid on add")
		return errors.New("Arg Name was nil on add is not valid")
	}

	b = BridgeEntry{args.ContainerID, args.Name, args.IPAddress, ""}
	err = Insert(b)
	fmt.Println("added the following:" + args.ContainerID)
	fmt.Println("calling add-host with ip=" + args.IPAddress + " name=" + args.Name)
	cmd = exec.Command("/cluster/bin/add-host.sh", args.IPAddress, args.Name)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println("here at four")
		fmt.Println(err.Error())
		errorString := fmt.Sprintf("%s\n%s\n%s\n", err.Error(), out.String(), stderr.String())
		reply.Output = errorString
		return nil
	} else {
		reply.Output = out.String()
		return nil
	}

	reply.Output = "error in update invalid option"
	return nil

}
