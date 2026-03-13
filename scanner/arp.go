package scanner

import (
	"fmt"
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
		return fmt.Errorf("invalid source IP: %v", srcIP)
	}
	dst, ok := netip.AddrFromSlice(targetIP.To4())
	if !ok {
		return fmt.Errorf("invalid target IP: %v", targetIP)
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

func collectReplies(client *arp.Client, subnet *net.IPNet, done <-chan struct{}) []arpReply {
	var replies []arpReply
	for {
		// Use a short read deadline so we can check the done signal between reads.
		if err := client.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
			return replies
		}
		pkt, _, err := client.Read()
		if err != nil {
			// Check if we've been told to stop.
			select {
			case <-done:
				return replies
			default:
				continue
			}
		}
		if pkt.Operation == arp.OperationReply {
			ip := net.IP(pkt.SenderIP.AsSlice())
			if !subnet.Contains(ip) {
				continue
			}
			replies = append(replies, arpReply{IP: ip, MAC: pkt.SenderHardwareAddr})
		}
	}
}
