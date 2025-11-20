package osinfo

import (
	"net"
	"os/user"
	"runtime"
)

type Sysinfo struct {
	OS        string
	Uptime    int64
	PrivateIP string
	UserInfo  string
}

func getPrivateIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func getUserInfo() string {
	user, err := user.Current()
	if err != nil {
		return ""
	}
	return user.Username
}
func GetCurrentOSInfo() Sysinfo {
	return Sysinfo{
		OS:        runtime.GOOS,
		Uptime:    0,
		PrivateIP: getPrivateIP(),
		UserInfo:  getUserInfo(),
	}
}
