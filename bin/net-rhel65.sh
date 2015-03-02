
#!/bin/bash -x

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

#
# script to test the networking configuration of the POC
#
errorMsg="EPIC FAIL!!"
successMsg="SWEET SUCCESS!!!"

admin=(rh65-admin.crunchy.lab)
workers=(rh65-server1.crunchy.lab rh65-server2.crunchy.lab)
servers=(${admin[@]} ${workers[@]})

for i in "${servers[@]}"
do
	ping -q -c 1 $i > /dev/null
	if [ $? -ne 0 ]; then
		echo "could not reach server host at " $i
		echo $errorMsg
		exit
	fi
	echo -e "\e[32mping "$i"\e[0m"
done

dockerbridges=(172.17.42.1 )
for i in "${dockerbridges[@]}"
do
	ping -q -c 1 $i > /dev/null
	if [ $? -ne 0 ]; then
		echo "could not reach docker bridge at " $i
		echo -e "\e[91m"$errorMsg
		exit
	fi
	echo -e "\e[32mping "$i"\e[0m"
done

for i in "${servers[@]}"
do
	RESULT=`ssh root@$i 'pgrep dnsbridgeclient'`
	if [ "${RESULT:-null}" = null ]; then
		echo -e "\e[91m" "dnsbridgeclient not running at" $i "\e[0m"
		echo -e "\e[91m"$errorMsg"\e[0m"
		exit
	fi
	echo -e "\e[32mdnsbridgeclient running on "$i"\e[0m"
	RESULT=`ssh root@$i 'pgrep docker'`
	if [ "${RESULT:-null}" = null ]; then
		echo -e "\e[91m" "docker not running at" $i "\e[0m"
		echo -e "\e[91m"$errorMsg"\e[0m"
		exit
	fi
	echo -e "\e[32mdocker running on "$i"\e[0m"
done

for i in "${admin[@]}"
do
	RESULT=`ssh root@$i 'pgrep named'`
	if [ "${RESULT:-null}" = null ]; then
		echo -e "\e[91m" "named not running at" $i "\e[0m"
		echo -e "\e[91m"$errorMsg"\e[0m"
		exit
	fi
	echo -e "\e[32m named running on "$i"\e[0m"

	RESULT=`ssh root@$i 'pgrep -f dnsbridgeserver'`
	if [ "${RESULT:-null}" = null ]; then
		echo -e "\e[91m" "dnsbridgeserver not running at" $i "\e[0m"
		echo -e "\e[91m"$errorMsg"\e[0m"
		exit
	fi
	echo -e "\e[32m dnsbridgeserver running on "$i"\e[0m"
done

#check that pg is installed on all servers
for i in "${servers[@]}"
do
	ssh root@$i 'rpm -q postgresql93 > /dev/null'
	if [ $? -ne 0 ]; then
		echo "postgresql not installed on server host at " $i
		echo $errorMsg
		exit
	fi
	echo -e "\e[32mpostgres installed on  "$i"\e[0m"
done
echo -e "\e[32m"$successMsg"\e[0m"
