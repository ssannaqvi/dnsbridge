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

// dnsbridgeserver is an RPC server that is invoked by all running
// dnsbridgeclients.  Each client will listen for Docker events, and
// invoke dnsbridgeserver when DNS entries need to be added or removed.

import (
	"crunchy.com/dnsbridge"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
)

var PORT string

func init() {
	flag.StringVar(&PORT, "p", "", "port to listen on default is 14000")
	flag.Parse()
}

func main() {

	fmt.Printf("dnsbridge server starting\n")
	command := new(dnsbridge.Command)
	rpc.Register(command)
	fmt.Printf("Command registered\n")
	rpc.HandleHTTP()
	fmt.Printf("listening on port\n" + PORT)
	l, e := net.Listen("tcp", ":"+PORT)
	if e != nil {
		fmt.Printf("listen error:%s\n", e.Error())
	}
	fmt.Printf("about to serve\n")
	http.Serve(l, nil)
	fmt.Printf("after serve\n")
}
