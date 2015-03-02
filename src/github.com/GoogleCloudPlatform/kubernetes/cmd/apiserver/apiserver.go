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

// apiserver is the main api server and master for the cluster.
// it is responsible for serving the cluster management API.
package main

import (
	"flag"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/apiserver"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/capabilities"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/cloudprovider"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/master"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/tools"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/version/verflag"

	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
)

var (
	// Note: the weird ""+ in below lines seems to be the only way to get gofmt to
	// arrange these text blocks sensibly. Grrr.
	port = flag.Int("port", 8080, ""+
		"The port to listen on. Default 8080. It is assumed that firewall rules are "+
		"set up such that this port is not reachable from outside of the cluster. It is "+
		"further assumed that port 443 on the cluster's public address is proxied to this "+
		"port. This is performed by nginx in the default setup.")
	address               = util.IP(net.ParseIP("127.0.0.1"))
	publicAddressOverride = flag.String("public_address_override", "", ""+
		"Public serving address. Read only port will be opened on this address, "+
		"and it is assumed that port 443 at this address will be proxied/redirected "+
		"to '-address':'-port'. If blank, the address in the first listed interface "+
		"will be used.")
	readOnlyPort = flag.Int("read_only_port", 7080, ""+
		"The port from which to serve read-only resources. If 0, don't serve on a "+
		"read-only address. It is assumed that firewall rules are set up such that "+
		"this port is not reachable from outside of the cluster.")
	apiPrefix               = flag.String("api_prefix", "/api", "The prefix for API requests on the server. Default '/api'.")
	storageVersion          = flag.String("storage_version", "", "The version to store resources with. Defaults to server preferred")
	cloudProvider           = flag.String("cloud_provider", "", "The provider for cloud services.  Empty string for no provider.")
	cloudConfigFile         = flag.String("cloud_config", "", "The path to the cloud provider configuration file.  Empty string for no configuration file.")
	healthCheckMinions      = flag.Bool("health_check_minions", true, "If true, health check minions and filter unhealthy ones. Default true.")
	eventTTL                = flag.Duration("event_ttl", 48*time.Hour, "Amount of time to retain events. Default 2 days.")
	tokenAuthFile           = flag.String("token_auth_file", "", "If set, the file that will be used to secure the API server via token authentication.")
	authorizationMode       = flag.String("authorization_mode", "AlwaysAllow", "Selects how to do authorization.  One of: "+strings.Join(apiserver.AuthorizationModeChoices, ","))
	authorizationPolicyFile = flag.String("authorization_policy_file", "", "File with authorization policy in csv format, used with --authorization_mode=ABAC.")
	etcdServerList          util.StringList
	etcdConfigFile          = flag.String("etcd_config", "", "The config file for the etcd client. Mutually exclusive with -etcd_servers.")
	corsAllowedOriginList   util.StringList
	allowPrivileged         = flag.Bool("allow_privileged", false, "If true, allow privileged containers.")
	portalNet               util.IPNet // TODO: make this a list
	enableLogsSupport       = flag.Bool("enable_logs_support", true, "Enables server endpoint for log collection")
	kubeletConfig           = client.KubeletConfig{
		Port:        10250,
		EnableHttps: false,
	}
)

func init() {
	flag.Var(&address, "address", "The IP address on to serve on (set to 0.0.0.0 for all interfaces)")
	flag.Var(&etcdServerList, "etcd_servers", "List of etcd servers to watch (http://ip:port), comma separated. Mutually exclusive with -etcd_config")
	flag.Var(&corsAllowedOriginList, "cors_allowed_origins", "List of allowed origins for CORS, comma separated.  An allowed origin can be a regular expression to support subdomain matching.  If this list is empty CORS will not be enabled.")
	flag.Var(&portalNet, "portal_net", "A CIDR notation IP range from which to assign portal IPs. This must not overlap with any IP ranges assigned to nodes for pods.")
	client.BindKubeletClientConfigFlags(flag.CommandLine, &kubeletConfig)
}

// TODO: Longer term we should read this from some config store, rather than a flag.
func verifyPortalFlags() {
	if portalNet.IP == nil {
		glog.Fatal("No -portal_net specified")
	}
}

func newEtcd(etcdConfigFile string, etcdServerList util.StringList) (helper tools.EtcdHelper, err error) {
	var client tools.EtcdGetSet
	if etcdConfigFile != "" {
		client, err = etcd.NewClientFromFile(etcdConfigFile)
		if err != nil {
			return helper, err
		}
	} else {
		client = etcd.NewClient(etcdServerList)
	}

	return master.NewEtcdHelper(client, *storageVersion)
}

func main() {
	flag.Parse()
	util.InitLogs()
	defer util.FlushLogs()

	verflag.PrintAndExitIfRequested()
	verifyPortalFlags()

	if (*etcdConfigFile != "" && len(etcdServerList) != 0) || (*etcdConfigFile == "" && len(etcdServerList) == 0) {
		glog.Fatalf("specify either -etcd_servers or -etcd_config")
	}

	capabilities.Initialize(capabilities.Capabilities{
		AllowPrivileged: *allowPrivileged,
	})

	cloud := cloudprovider.InitCloudProvider(*cloudProvider, *cloudConfigFile)

	kubeletClient, err := client.NewKubeletClient(&kubeletConfig)
	if err != nil {
		glog.Fatalf("Failure to start kubelet client: %v", err)
	}

	// TODO: expose same flags as client.BindClientConfigFlags but for a server
	clientConfig := &client.Config{
		Host:    net.JoinHostPort(address.String(), strconv.Itoa(int(*port))),
		Version: *storageVersion,
	}
	client, err := client.New(clientConfig)
	if err != nil {
		glog.Fatalf("Invalid server address: %v", err)
	}

	helper, err := newEtcd(*etcdConfigFile, etcdServerList)
	if err != nil {
		glog.Fatalf("Invalid storage version or misconfigured etcd: %v", err)
	}

	n := net.IPNet(portalNet)

	authorizer, err := apiserver.NewAuthorizerFromAuthorizationConfig(*authorizationMode, *authorizationPolicyFile)
	if err != nil {
		glog.Fatalf("Invalid Authorization Config: %v", err)
	}

	config := &master.Config{
		Client:                client,
		Cloud:                 cloud,
		EtcdHelper:            helper,
		HealthCheckMinions:    *healthCheckMinions,
		EventTTL:              *eventTTL,
		KubeletClient:         kubeletClient,
		PortalNet:             &n,
		EnableLogsSupport:     *enableLogsSupport,
		EnableUISupport:       true,
		APIPrefix:             *apiPrefix,
		CorsAllowedOriginList: corsAllowedOriginList,
		TokenAuthFile:         *tokenAuthFile,
		ReadOnlyPort:          *readOnlyPort,
		ReadWritePort:         *port,
		PublicAddress:         *publicAddressOverride,
		Authorizer:            authorizer,
	}
	m := master.New(config)

	roLocation := ""
	if *readOnlyPort != 0 {
		roLocation = net.JoinHostPort(config.PublicAddress, strconv.Itoa(config.ReadOnlyPort))
	}
	rwLocation := net.JoinHostPort(address.String(), strconv.Itoa(int(*port)))

	// See the flag commentary to understand our assumptions when opening the read-only and read-write ports.

	if roLocation != "" {
		// Allow 1 read-only request per second, allow up to 20 in a burst before enforcing.
		rl := util.NewTokenBucketRateLimiter(1.0, 20)
		readOnlyServer := &http.Server{
			Addr:           roLocation,
			Handler:        apiserver.RecoverPanics(apiserver.ReadOnly(apiserver.RateLimit(rl, m.InsecureHandler))),
			ReadTimeout:    5 * time.Minute,
			WriteTimeout:   5 * time.Minute,
			MaxHeaderBytes: 1 << 20,
		}
		go func() {
			defer util.HandleCrash()
			for {
				if err := readOnlyServer.ListenAndServe(); err != nil {
					glog.Errorf("Unable to listen for read only traffic (%v); will try again.", err)
				}
				time.Sleep(15 * time.Second)
			}
		}()
	}

	s := &http.Server{
		Addr:           rwLocation,
		Handler:        apiserver.RecoverPanics(m.InsecureHandler),
		ReadTimeout:    5 * time.Minute,
		WriteTimeout:   5 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}
	glog.Fatal(s.ListenAndServe())
}
