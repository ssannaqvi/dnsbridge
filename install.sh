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

# this install script assumes a registered RHEL 7 server is the installation host OS
#
# before running this script you will need to:
# 1) enable the extras and optional RH repos by running the following:
# 	subscription-manager repos --enable=rhel-7-server-extras-rpms --enable=rhel-7-server-optional-rpms
# 2) set up a static ip address on the installation host
# 3) set up a hostname using the domain name of your choosing (e.g. ns.crunchy.lab)
# 4) edit this script, add any extra remote servers that will
#    be used as dnsbridge clients, look for 'remoteservers' at the
#    end of this script, add any extra servers within the parens
# 5) edit ./config/zonefiles and alter the ip addresses/domain name,
#    and number of zones for your configuration
# 6) edit ./config/named.conf and edit the ip address/domain names
#    of the dnsbridge server for your configuration
# 7) edit ./bin/add-host.sh and ./bin/delete-host.sh change domain names
#    if required for your installation
# 8) edit /etc/hosts and add your hostname and IP address
# 9) edit /etc/resolv.conf and add your IP address as the primary nameserver
# 10) edit ./config/docker and add your static IP address if you are not
#    using the assumed IP address of 192.168.56.103
#
#
# install deps

#Check if current user is member to the wheel group
username= whoami
if groups $username | grep &>/dev/null 'wheel'; then
        echo "Group permissions ok"
else
        echo "You must have sudo privledges to run this install"
        exit
fi

export INSTALLDIR=`pwd`

$INSTALLDIR/requirements.sh

sudo cp $INSTALLDIR/config/docker /etc/sysconfig/docker

sudo systemctl enable docker.service
sudo systemctl start docker.service

# load up the bridge database
cp $INSTALLDIR/sql/bridge.sql /tmp
sudo su - postgres -c '/usr/pgsql-9.3/bin/psql -U postgres postgres < /tmp/bridge.sql'

# set the gopath
source $INSTALLDIR/bin/setpath.sh

# compile the source
cd src/crunchy.com
make

#
# install bind
sudo yum -y install bind

#
# copy bind config files and sample zonefiles 
sudo cp  $INSTALLDIR/config/named.conf /etc/named.conf
sudo cp  $INSTALLDIR/config/zonefiles/* /var/named/dynamic
sudo chown -R named:named  /var/named/dynamic
sudo sh -c 'chmod 644  /var/named/dynamic/*.db'

#
# deploy dnsbridge binaries and systemd files 

adminserver=`hostname`
remoteservers=($adminserver)

for i in "${remoteservers[@]}"
do
        echo $i
        ssh root@$i "mkdir -p /cluster/bin"
        scp $INSTALLDIR/bin/dnsbridgeclient  \
        root@$i:/cluster/bin/
        scp  $INSTALLDIR/config/dnsbridgeclient.service root@$i:/usr/lib/systemd/system
        ssh root@$i "systemctl enable dnsbridgeclient.service"
done

# copy all required admin files to the admin server

ssh root@$adminserver "mkdir -p /cluster/bin"
scp $INSTALLDIR/bin/add-host.sh $INSTALLDIR/bin/delete-host.sh \
$INSTALLDIR/bin/delete-ip.sh \
$INSTALLDIR/bin/dnsbridgeclient \
$INSTALLDIR/bin/dnsbridgeserver \
root@$adminserver:/cluster/bin

scp $INSTALLDIR/config/dnsbridgeserver.service  \
$INSTALLDIR/config/dnsbridgeclient.service  \
root@$adminserver:/usr/lib/systemd/system

ssh root@$adminserver "systemctl enable named.service"
ssh root@$adminserver "named-checkzone 0.17.172.in-addr.arpa  /var/named/dynamic/0.17.172.zone.db"
ssh root@$adminserver "named-checkzone crunchy.lab  /var/named/dynamic/crunchy.lab.db"
ssh root@$adminserver "systemctl start named.service"
ssh root@$adminserver "systemctl enable dnsbridgeserver.service"
ssh root@$adminserver "systemctl enable dnsbridgeclient.service"

