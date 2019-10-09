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

// IsIP4 是否是IPv4
func IsIP4(ipv4 string) bool {
	ip := net.ParseIP(ipv4)
	return ip != nil && ip.To4() != nil
}

// ListLocalIPMap 列出本机的IP地址Map
func ListLocalIPMap() map[string]bool {
	m := make(map[string]bool)
	ifaces, _ := ListLocalIfaceAddrs()
	for _, ifa := range ifaces {
		m[ifa.IP] = true
	}

	return m
}

// ListLocalIps 列出本机的IP地址列表
func ListLocalIps() []string {
	ips := make([]string, 0)
	ifaces, _ := ListLocalIfaceAddrs()
	for _, ifa := range ifaces {
		ips = append(ips, ifa.IP)
	}

	return ips
}

// IfaceAddr 表示一个IP地址和网卡名称的结构
type IfaceAddr struct {
	IP        string
	IfaceName string
}

// ListLocalIfaceAddrs 列出本机所有IP和网卡名称
func ListLocalIfaceAddrs() ([]IfaceAddr, error) {
	ret := make([]IfaceAddr, 0)
	list, err := net.Interfaces()
	if err != nil {
		return ret, err
	}

	for _, iface := range list {
		addrs, err := iface.Addrs()
		if err != nil {
			return ret, err
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
				continue
			}

			ret = append(ret, IfaceAddr{
				IP:        ipnet.IP.String(),
				IfaceName: iface.Name,
			})
		}
	}

	return ret, nil
}

/*
When using Nginx as a reverse proxy you may want to pass through the IP address
of the remote user to your backend web server.
This must be done using the X-Forwarded-For header.
You have a couple of options on how to set this information with Nginx.
You can either append the remote hosts IP address to any existing X-Forwarded-For values, or you can simply
set the X-Forwarded-For value, which clears out any previous IP’s that would have been on the request.

Edit the nginx configuration file, and add one of the follow lines in where appropriate.
To set the X-Forwarded-For to only contain the remote users IP:
proxy_set_header X-Forwarded-For $remote_addr;

To append the remote users IP to any existing X-Forwarded-For value:
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
*/
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
