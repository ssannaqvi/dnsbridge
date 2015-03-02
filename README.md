dnsbridge
===========

Build Requirements
==================

Obtaining this code or updates to it may require the Git package manager.
It can be installed on Fedora/RH with this command:

    yum install -y git

The main requirements to build and use this software require root
access, and for RHEL7 Linux this setup script will do all of them:

   ./requirements.sh

Installation
============

this install script assumes the following:

* IP address of 192.168.56.103 for this server
* Domain name of crunchy.lab 
* Host name of ns.crunchy.lab 
* you will be installing under your userid, not root
* your userid has sudo priviledge
 
You should change the IP address, hostname, and domain name to suit your local
requirements.  You will need to edit some of the installed scripts/files
as specified below.

Before running this script you will need to:

* set up a static ip address on the installation host
* set up a hostname using the domain name of your choosing (e.g. ns.crunchy.lab)
* edit this script, add any extra remote servers that will
  be used as dnsbridge clients, look for 'remoteservers' at the
  end of this script, add any extra servers within the parens, by default,
  the script will include the dnsbridge server as the only remote server, this
  allows this server to be used as a CPM deployment server as well as a DNS server
* edit ./config/zonefiles and alter the ip addresses/domain name,
  and number of zones for your configuration
* edit ./config/named.conf and edit the ip address/domain names
  of the dnsbridge server for your configuration
* edit ./bin/add-host.sh and ./bin/delete-host.sh change domain names
  if required for your installation
* edit /etc/hosts and add your hostname and IP address, add an entry for 
  the hostname that you have chosen for this server and which is specified
  in the installation files  (e.g. ns.crunchy.lab)
* edit /etc/resolv.conf and add your IP address as the primary nameserver
* run the install.sh script
* modify /etc/sysconfig/docker to add the DNS server to Docker's list
  of DNS servers to use, your /etc/sysconfig/docker file should look
  like this:

      OPTIONS=--selinux-enabled --dns=192.168.56.103 --dns=192.168.0.1 -H fd://

  Adjust these DNS server IP addresses to match your environment, these
  values need to be correct in order to build the CPM Docker images
  during a CPM intallation.

  Restart Docker after making these changes.

* test the install!  See Below for details on testing

Details
-------
dnsbridge is a docker-to-DNS bridge system used
for registering Docker events with a BIND system.

This is useful because IP addresses are dynamically
assigned by Docker and some applications require
a persistent way of locating a service, (e.g. Postgresql).

Some PaaS environments provide something akin to
dnsbridge, but using dnsbridge allows for standalone
deployments of some PaaS/DaaS types of applications
(e.g. Crunchy PostgreSQL Manager).

dnsbridgeclient
---------------
The client is run on each Docker host, it connects
to the Docker event stream, listens for start/stop/destroy
messages, and when received, it forwards them to the
dnsbridgeserver. 

dnsbridgeserver
---------------

The server receives events (start, stop, destroy) messages
from the dnsbridgeclients.

Upon receiving an event, the server will register new
A and PTR records in BIND that refer to the newly created
Docker containers.

Upon receiving a stop or destroy event, the server will
remove the associated A and PTR records from named.

Configuration
-------------

Bind (named) must be installed on the server's host.

Postgresql must be installed locally to the dnsbridgeserver
to be used as a persistent storage of all registered
containers.  As the server processes, it writes/deletes
registered container values from the Postgresql database.

The bridge.sql script must be run to create the dnsbridge
database in Postgresql.

The zonefiles distributed with dnsbridge can be used
as a model for creating your own DNS zones.  The samples
include a 3 node setup with a domain name of crunchy.lab
as an example.

Note, the named.conf file specifes that no recursion is allowed, you
might want to turn this on for your environment.

Testing
-------------

Verify your /etc/resolv.conf file has your as your primary DNS
the host you have just installed dnsbridge server upon, you should
see something similar to:
~~~~~~~~~~~~
search crunchy.lab
nameserver 192.168.56.103
nameserver 192.168.0.1
~~~~~~~~~~~~

if you notice that your primary DNS server is not in the /etc/resolv.conf
after a reboot, it means your PEERDNS setting in your networking
is not set to "no", see /etc/sysconfig/network-script/ifcfg files for
details.

Verify that dnsbridgeserver and dnsbridgeclient are running:
~~~~~~~~~~
ps ax | grep dnsbridge
~~~~~~~~~~

After installation, you can test the install by issuing the following (as root):

/cluster/bin/add-host.sh 192.168.56.103 test.crunchy.lab

This command should register a new host.

You can then use the 'dig' command to make sure it resolves the name:

dig test.crunchy.lab

This should show that the name resolves using ns.crunchy.lab as the DNS nameserver.

