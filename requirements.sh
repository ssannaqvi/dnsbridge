#!/bin/bash
#

# Copyright 2015 Crunchy Data Solutions, Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

sudo subscription-manager repos --enable=rhel-7-server-extras-rpms --enable=rhel-7-server-optional-rpms
sudo yum -y update
sudo yum -y install gcc make golang docker-io
sudo yum -y install bind
sudo rpm -Uvh http://dl.fedoraproject.org/pub/epel/7/x86_64/e/epel-release-7-5.noarch.rpm
sudo rpm -Uvh http://yum.postgresql.org/9.3/redhat/rhel-7-x86_64/pgdg-redhat93-9.3-1.noarch.rpm
sudo yum install -y postgresql93 postgresql93-contrib postgresql93-server libxslt unzip openssh-clients hostname bind-utils net-tools

# set up the postgres directory
sudo su - postgres -c '/usr/pgsql-9.3/bin/initdb -D /var/lib/pgsql/9.3/data'
sudo systemctl enable postgresql-9.3.service

sudo systemctl start postgresql-9.3.service
