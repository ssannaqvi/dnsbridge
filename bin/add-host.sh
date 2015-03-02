#!/bin/bash

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

# param 1 is the IP address
# param 2 is the hostname
#
# add the A record and the PTR record

reverseIp() {
    local a i n
    IFS=. read -r -a a <<< "$1"
    n=${#a[@]}
    for (( i = n-1; i > 0; i-- )); do
        printf '%s.' "${a[i]}"
    done
    printf '%s' "${a[0]}"
}
reverseZone() {
    local a i n
    IFS=. read -r -a a <<< "$1"
    n=${#a[@]}
    for (( i = 1; i < n-1; i++ )); do
        printf '%s.' "${a[i]}"
    done
    printf '%s' "${a[n-1]}"
}

DOMAIN=crunchy.lab
IP=$1
HOST=$2
REVERSEIP=`reverseIp $IP`
REVERSEIPZONE=`reverseZone ${REVERSEIP}`
REVERSEZONE=$REVERSEIPZONE.in-addr.arpa
REVERSEFQDN=$REVERSEIP.in-addr.arpa
echo $REVERSEZONE is REVERSEZONE
echo $REVERSEFQDN is REVERSEFQDN

/cluster/bin/delete-host.sh $2
/cluster/bin/delete-ip.sh $1


#server ns.$DOMAIN
tmpfile=$(mktemp)
cat > $tmpfile <<END
zone $DOMAIN.
update add $HOST.$DOMAIN 60 A $IP
send
zone  $REVERSEZONE
update add $REVERSEFQDN 3500 IN PTR $HOST.$DOMAIN.
send
END
cat $tmpfile
nsupdate $tmpfile
