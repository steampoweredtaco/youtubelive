package youtubelive

import (
	"fmt"
	"github.com/libp2p/go-reuseport"
	"net"
	"sort"
	"strconv"
)

type listenResolve struct {
	listenAddr    string
	stickyPort    net.Listener
	effectiveAddr string
}

// categorizeIPs collects and sorts IPs by priority
func categorizeIPs() (public []net.IP, private []net.IP, loopback []net.IP) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			if ip4 := ip.To4(); ip4 != nil {
				ip = ip4
			} else if ip6 := ip.To16(); ip6 != nil {
				ip = ip6
			} else {
				continue
			}

			switch {
			case ip.IsLoopback():
				loopback = append(loopback, ip)
			case isPrivate(ip):
				private = append(private, ip)
			default:
				public = append(public, ip)
			}
		}
	}

	// Sorting logic
	sortIPs := func(ips []net.IP) {
		sort.Slice(ips, func(i, j int) bool {
			// Prefer IPv4 over IPv6
			ip4i := ips[i].To4() != nil
			ip4j := ips[j].To4() != nil
			if ip4i != ip4j {
				return ip4i
			}

			// Prefer global unicast IPv6
			if !ip4i && ips[i].IsGlobalUnicast() && !ips[j].IsGlobalUnicast() {
				return true
			}

			// Stable string comparison
			return ips[i].String() < ips[j].String()
		})
	}

	sortIPs(public)
	sortIPs(private)
	sortIPs(loopback)

	return public, private, loopback
}

func isPrivate(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			(ip4[0] == 169 && ip4[1] == 254)
	}
	if len(ip) == net.IPv6len {
		return (ip[0]&0xfe == 0xfc) || (ip[0] == 0xfe && (ip[1]&0xc0) == 0x80)
	}
	return false
}

func isValid(ip net.IP) bool {
	return !ip.IsLoopback() && !ip.IsMulticast() && !ip.IsUnspecified()
}

func selectBestIP(public, private, loopback []net.IP) net.IP {
	for _, ip := range public {
		if isValid(ip) {
			return ip
		}
	}
	for _, ip := range private {
		if isValid(ip) {
			return ip
		}
	}
	for _, ip := range loopback {
		if isValid(ip) {
			return ip
		}
	}
	return net.IPv4(127, 0, 0, 1)
}

func (yt *listenResolve) setupListener() error {
	if yt.stickyPort != nil {
		return nil
	}
	listener, err := reuseport.Listen("tcp", yt.listenAddr)
	if err != nil {
		return err
	}
	yt.stickyPort = listener

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("could not determine listening address")
	}

	var selectedIP net.IP
	if tcpAddr.IP.IsUnspecified() {
		public, private, loopback := categorizeIPs()
		selectedIP = selectBestIP(public, private, loopback)
	} else {
		selectedIP = tcpAddr.IP
	}

	yt.effectiveAddr = net.JoinHostPort(
		selectedIP.String(),
		strconv.Itoa(tcpAddr.Port),
	)
	return nil
}
