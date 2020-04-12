package zerosvc

import (
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var currentFQDN string
var currentFQDNTS time.Time
var currentFQDNTSLock sync.Mutex

// GetFQDN tries to get fqdn from os (via hostname) and fails back on system resolver.
func GetFQDN() string {
	if time.Now().Before(currentFQDNTS.Add(time.Minute*10)) && currentFQDN != "" {
		currentFQDNTSLock.Lock()
		defer currentFQDNTSLock.Unlock()
		return currentFQDN
	}
	goHostname, _ := os.Hostname()
	// ask system.
	out, err := exec.Command("hostname", "--fqdn").Output()
	if err == nil {
		sysFQDN := strings.TrimSpace(string(out))
		if strings.Contains(sysFQDN, goHostname) {
			currentFQDNTSLock.Lock()
			currentFQDN = sysFQDN
			currentFQDNTS = time.Now()
			currentFQDNTSLock.Unlock()
			return sysFQDN
		}
	}
	addrs, err := net.LookupIP(goHostname)
	if err != nil {
		return goHostname
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return goHostname
			}
			hosts, err := net.LookupAddr(string(ip))
			if err != nil || len(hosts) == 0 {
				return goHostname
			}
			fqdn := hosts[0]
			return strings.TrimSuffix(fqdn, ".")
		}
	}
	currentFQDNTSLock.Lock()
	currentFQDN = goHostname
	currentFQDNTS = time.Now()
	currentFQDNTSLock.Unlock()
	return goHostname
}
