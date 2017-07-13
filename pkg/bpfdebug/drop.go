// Copyright 2016-2017 Authors of Cilium
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bpfdebug

import (
	"fmt"
	clientPkg "github.com/cilium/cilium/pkg/client"
	"github.com/cilium/cilium/api/v1/models"
	"strconv"
	"github.com/spf13/viper"
)

const (
	// DropNotifyLen is the amount of packet data provided in a drop notification
	DropNotifyLen = 32
)

var (
	endpointInfoCache = make(map[int]*models.Endpoint)
	client, _ = clientPkg.NewClient(viper.GetString("host"))
)
// DropNotify is the message format of a drop notification in the BPF ring buffer
type DropNotify struct {
	Type     uint8
	SubType  uint8
	Source   uint16
	Hash     uint32
	OrigLen  uint32
	CapLen   uint32
	SrcLabel uint32
	DstLabel uint32
	DstID    uint32
	Ifindex  uint32
	// data
}

var errors = map[uint8]string{
	0:   "Success",
	2:   "Invalid packet",
	130: "Invalid source mac",
	131: "Invalid destination mac",
	132: "Invalid source ip",
	133: "Policy denied",
	134: "Invalid packet",
	135: "CT: Truncated or invalid header",
	136: "CT: Missing TCP ACK flag",
	137: "CT: Unknown L4 protocol",
	138: "CT: Can't create entry from packet",
	139: "Unsupported L3 protocol",
	140: "Missed tail call",
	141: "Error writing to packet",
	142: "Unknown L4 protocol",
	143: "Unknown ICMPv4 code",
	144: "Unknown ICMPv4 type",
	145: "Unknown ICMPv6 code",
	146: "Unknown ICMPv6 type",
	147: "Error retrieving tunnel key",
	148: "Error retrieving tunnel options",
	149: "Invalid Geneve option",
	150: "Unknown L3 target address",
	151: "Not a local target address",
	152: "No matching local container found",
	153: "Error while correcting L3 checksum",
	154: "Error while correcting L4 checksum",
	155: "CT: Map insertion failed",
	156: "Invalid IPv6 extension header",
	157: "IPv6 fragmentation not supported",
	158: "Service backend not found",
	159: "Policy denied (L4)",
	160: "No tunnel/encapsulation endpoint",
}

func dropReason(reason uint8) string {
	if err, ok := errors[reason]; ok {
		return err
	}
	return fmt.Sprintf("%d", reason)
}

func (n *DropNotify) DumpInfo(data []byte) {
	var epGet *models.Endpoint
	var err error
	if ep, ok := endpointInfoCache[int(n.Source)]; !ok {
		fmt.Printf("cache miss\n")
		epGet, err = client.EndpointGet(strconv.Itoa(int(n.Source)))
		if err != nil {
			fmt.Printf("\tunable to get information for endpoint %d\n", n.Source)
		}
		fmt.Printf("\t adding to cache")
		endpointInfoCache[int(n.Source)] = epGet
	} else {
		fmt.Printf("cache hit\n")
		if ep.Identity == nil {
			fmt.Printf("\tendpointInfoCache identity nil - getting it again to see if identity updated\n")
			epGet, err = client.EndpointGet(strconv.Itoa(int(n.Source)))
			if err != nil {
				fmt.Printf("\tunable to get endpoint %d from API server\n", n.Source)
			}
			if epGet.Identity != nil {
				fmt.Printf("\tidentity not nil after it was nil, so updating endpoint in cache\n")
				endpointInfoCache[int(n.Source)] = epGet
			}
		}
	}
	ep2 := endpointInfoCache[int(n.Source)]
	if ep2.Identity == nil {
		fmt.Printf("identity nil, so not accessing identity\n")
		fmt.Printf("\t[%v]:%d (nil secID} (%s), srcLabel=%d, dstLabel=%d, dstId=%d\n", ep2.Addressing.IPV4, n.Source, dropReason(n.SubType), n.SrcLabel, n.DstLabel, n.DstID)
	} else {
		//fmt.Printf("DROP: FROM: [ifindex %d / endpoint %d] (%s) %d bytes\n",  n.Ifindex, n.Source, dropReason(n.SubType), n.OrigLen)
		fmt.Printf("\t[%v]:%d (id %d) (%s), , srcLabel=%d, dstLabel=%d, dstId=%d\n", ep2.Identity.Labels, n.Source, ep2.Identity.ID, dropReason(n.SubType), n.SrcLabel, n.DstLabel, n.DstID)
	}
	}

// Dump prints the drop notification in human readable form
func (n *DropNotify) DumpVerbose(dissect bool, data []byte, prefix string) {
	fmt.Printf("%s MARK %#x FROM %d Packet dropped %d (%s) %d bytes ifindex=%d",
		prefix, n.Hash, n.Source, n.SubType, dropReason(n.SubType), n.OrigLen, n.Ifindex)

	if n.SrcLabel != 0 || n.DstLabel != 0 {
		fmt.Printf(" %d->%d", n.SrcLabel, n.DstLabel)
	}

	if n.DstID != 0 {
		fmt.Printf(" to lxc %d\n", n.DstID)
	} else {
		fmt.Printf("\n")
	}

	if n.CapLen > 0 && len(data) > DropNotifyLen {
		Dissect(dissect, data[DropNotifyLen:])
	}
}
