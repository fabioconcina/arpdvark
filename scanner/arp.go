package scanner

import (
	"net"
	"net/netip"
	"time"

	"github.com/mdlayher/arp"
)

type arpReply struct {
	IP  net.IP
	MAC net.HardwareAddr
}

func dialARP(iface *net.Interface) (*arp.Client, error) {
	return arp.Dial(iface)
}

func sendARP(client *arp.Client, targetIP, srcIP net.IP, srcMAC net.HardwareAddr) error {
	src, ok := netip.AddrFromSlice(srcIP.To4())
	if !ok {
		return nil
	}
	dst, ok := netip.AddrFromSlice(targetIP.To4())
	if !ok {
		return nil
	}
	pkt, err := arp.NewPacket(
		arp.OperationRequest,
		srcMAC,
		src,
		net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		dst,
	)
	if err != nil {
		return err
	}
	return client.WriteTo(pkt, net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
}

func collectReplies(client *arp.Client, deadline time.Time) []arpReply {
	var replies []arpReply
	if err := client.SetReadDeadline(deadline); err != nil {
		return replies
	}
	for {
		pkt, _, err := client.Read()
		if err != nil {
			break
		}
		if pkt.Operation == arp.OperationReply {
			ip := net.IP(pkt.SenderIP.AsSlice())
			replies = append(replies, arpReply{IP: ip, MAC: pkt.SenderHardwareAddr})
		}
	}
	return replies
}
