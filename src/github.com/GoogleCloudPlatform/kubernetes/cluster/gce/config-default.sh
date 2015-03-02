#!/bin/bash

# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# TODO(jbeda): Provide a way to override project
ZONE=us-central1-b
MASTER_SIZE=n1-standard-1
MINION_SIZE=n1-standard-1
NUM_MINIONS=4
# TODO(dchen1107): Filed an internal issue to create an alias
# for containervm image, so that gcloud/gcutil will expand this
# to the latest supported image.
IMAGE=container-vm-v20141016
IMAGE_PROJECT=google-containers
NETWORK=default
INSTANCE_PREFIX=kubernetes
MASTER_NAME="${INSTANCE_PREFIX}-master"
MASTER_TAG="${INSTANCE_PREFIX}-master"
MINION_TAG="${INSTANCE_PREFIX}-minion"
MINION_NAMES=($(eval echo ${INSTANCE_PREFIX}-minion-{1..${NUM_MINIONS}}))
MINION_IP_RANGES=($(eval echo "10.244.{1..${NUM_MINIONS}}.0/24"))
MINION_SCOPES="compute-rw"
# Increase the sleep interval value if concerned about API rate limits. 3, in seconds, is the default.
POLL_SLEEP_INTERVAL=3
PORTAL_NET="10.0.0.0/16"
# When set to true, heapster will be setup as part of the cluster bring up.
MONITORING=true
