/*
Package include imports all protos packages so that they register with the global
registry. This package can be imported in the main package to automatically register
all of the standard supported Packetbeat protocols.
*/
package include

import (
	// This list is automatically generated by `make imports`
	_ "github.com/elastic/beats/packetbeat/protos/amqp"
	_ "github.com/elastic/beats/packetbeat/protos/applayer"
	_ "github.com/elastic/beats/packetbeat/protos/cassandra"
	_ "github.com/elastic/beats/packetbeat/protos/dns"
	_ "github.com/elastic/beats/packetbeat/protos/http"
	_ "github.com/elastic/beats/packetbeat/protos/icmp"
	_ "github.com/elastic/beats/packetbeat/protos/memcache"
	_ "github.com/elastic/beats/packetbeat/protos/mongodb"
	_ "github.com/elastic/beats/packetbeat/protos/mysql"
	_ "github.com/elastic/beats/packetbeat/protos/nfs"
	_ "github.com/elastic/beats/packetbeat/protos/pgsql"
	_ "github.com/elastic/beats/packetbeat/protos/redis"
	_ "github.com/elastic/beats/packetbeat/protos/smtp"
	_ "github.com/elastic/beats/packetbeat/protos/tcp"
	_ "github.com/elastic/beats/packetbeat/protos/thrift"
	_ "github.com/elastic/beats/packetbeat/protos/udp"
)
