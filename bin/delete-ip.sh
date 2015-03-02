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

# param 1 is the ip address
#
# delete ALL the A record and the PTR record for this IP address

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
echo "param 1 = ["$1"]"
DOMAIN=crunchy.lab

host ${1}

IFS='
'
IParray=(`host ${1}`)

REVERSEIP=`reverseIp $1`
echo $REVERSEIP is REVERSEIP
REVERSEIPZONE=`reverseZone ${REVERSEIP}`
REVERSEZONE=$REVERSEIPZONE.in-addr.arpa
echo $REVERSEZONE is REVERSEZONE


for i in "${IParray[@]}"
do

IP=`echo $i | sed 's/domain name pointer/3500 IN PTR/'`
echo "deleting dns entries for IP " $IP


tmpfile=$(mktemp)
cat > $tmpfile <<END
zone  $REVERSEZONE
update delete $IP
send
END
cat $tmpfile
nsupdate $tmpfile

done

# we need to exit with a zero in all cases
exit 0
