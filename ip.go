package gonet

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

// IsIP 判断 host 字符串表达式是不是IP(v4/v6)的格式
func IsIP(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil
}

// IsIPv4 是否是IPv4
func IsIPv4(ipv4 string) bool {
	ip := net.ParseIP(ipv4)
	return ip != nil && ip.To4() != nil
}

// IsIPv6 tests if the str is an IPv6 format
func IsIPv6(str string) bool {
	ip := net.ParseIP(str)
	return ip != nil && strings.Contains(str, ":")
}

// ListIPv4Map 列出本机的IPv4地址Map
func ListIPv4Map() map[string]bool {
	return ListIPMap(IPv4)
}

// ListIPMap 列出本机的IP地址Map
func ListIPMap(mode ...ListMode) map[string]bool {
	m := make(map[string]bool)
	ifaces, _ := ListIfaces(mode...)

	for _, ifa := range ifaces {
		m[ifa.IP.String()] = true
	}

	return m
}

// ListIpsv4 列出本机的IPv4地址列表
func ListIpsv4() []string {
	return ListIps(IPv4)
}

// ListIps 列出本机的IP地址列表
func ListIps(mode ...ListMode) []string {
	ifaces, _ := ListIfaces(mode...)
	ips := make([]string, len(ifaces))

	for i, ifa := range ifaces {
		ips[i] = ifa.IP.String()
	}

	return ips
}

// IfaceAddr 表示一个IP地址和网卡名称的结构
type IfaceAddr struct {
	IP        string
	IfaceName string
}

// ListIfaceAddrsIPv4 列出本机所有IP和网卡名称
func ListIfaceAddrsIPv4() ([]IfaceAddr, error) {
	ifaces, err := ListIfaces(IPv4)
	if err != nil {
		return nil, err
	}

	addr := make([]IfaceAddr, len(ifaces))

	for i, iface := range ifaces {
		addr[i] = IfaceAddr{
			IP:        iface.IP.String(),
			IfaceName: iface.IfaceName,
		}
	}

	return addr, nil
}

// Iface 表示一个IP地址和网卡名称的结构
type Iface struct {
	IP        net.IP
	IfaceName string
}

// ListMode defines the mode for iface listing
type ListMode int

const (
	// IPv4v6 list all ipv4 and ipv6
	IPv4v6 ListMode = iota
	// IPv4 list only all ipv4
	IPv4
	// IPv6 list only all ipv6
	IPv6
)

// ListIfaces 根据mode 列出本机所有IP和网卡名称
func ListIfaces(mode ...ListMode) ([]Iface, error) {
	ret := make([]Iface, 0)
	list, err := net.Interfaces()

	if err != nil {
		return ret, err
	}

	modeMap := makeModeMap(mode)

	for _, iface := range list {
		addrs, err := iface.Addrs()
		if err != nil {
			return ret, err
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}

			if matchesMode(modeMap, ipnet) {
				ret = append(ret, Iface{
					IP:        ipnet.IP,
					IfaceName: iface.Name,
				})
			}
		}
	}

	return ret, nil
}

func matchesMode(modeMap map[ListMode]bool, ipnet *net.IPNet) bool {
	if _, all := modeMap[IPv4v6]; all {
		return true
	}

	if _, v4 := modeMap[IPv4]; v4 {
		return ipnet.IP.To4() != nil
	}

	if _, v6 := modeMap[IPv6]; v6 {
		return ipnet.IP.To16() != nil
	}

	return false
}

func makeModeMap(mode []ListMode) map[ListMode]bool {
	modeMap := make(map[ListMode]bool)

	for _, m := range mode {
		modeMap[m] = true
	}

	if len(modeMap) == 0 {
		modeMap[IPv4v6] = true
	}

	return modeMap
}

// GetClientIP ...
// When using Nginx as a reverse proxy you may want to pass through the IP address
// of the remote user to your backend web server.
// This must be done using the X-Forwarded-For header.
// You have a couple of options on how to set this information with Nginx.
// You can either append the remote hosts IP address to any existing X-Forwarded-For values, or you can simply
// set the X-Forwarded-For value, which clears out any previous IP’s that would have been on the request.
//
// Edit the nginx configuration file, and add one of the follow lines in where appropriate.
// To set the X-Forwarded-For to only contain the remote users IP:
// proxy_set_header X-Forwarded-For $remote_addr;
//
// To append the remote users IP to any existing X-Forwarded-For value:
// proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
func GetClientIP(req *http.Request) string {
	// "X-Forwarded-For"/ "x-forwarded-for"/"X-FORWARDED-FOR"  // capitalisation  doesn't matter
	xff := req.Header.Get("X-FORWARDED-FOR")
	if xff != "" {
		proxyIps := strings.Split(xff, ",")
		return proxyIps[0]
	}

	ip, _, _ := net.SplitHostPort(req.RemoteAddr)

	return ip
}

// IsPrivateIP ...
func IsPrivateIP(ip string) (bool, error) {
	IP := net.ParseIP(ip)
	if IP == nil {
		return false, errors.New("invalid IP")
	}

	networks := []string{"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15",
		"198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "255.255.255.255/32", "224.0.0.0/4"}

	for _, network := range networks {
		_, privateBitBlock, _ := net.ParseCIDR(network)
		if privateBitBlock.Contains(IP) {
			return true, nil
		}
	}

	return false, nil
}
