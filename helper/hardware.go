package helper

import (
	"crypto/md5"
	"fmt"
	"net"
	"os"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/mem"
)

func GetPrivateIPs() ([]string, error) {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip.IsPrivate() {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips, nil
}

func GetMACAddresses() ([]string, error) {
	var macs []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.HardwareAddr == nil {
			continue
		}
		macs = append(macs, iface.HardwareAddr.String())
	}
	return macs, nil
}

func GetMachineID() (string, error) {
	uid, err := machineid.ID()
	if err == nil {
		return uid, nil
	}

	// docker容器读取
	data, err := os.ReadFile("/proc/self/cgroup")
	if err == nil {
		return fmt.Sprintf("%x", md5.Sum(data)), nil
	}

	// 读取失败，自定义生成
	ipList, err := GetPrivateIPs()
	if err != nil {
		hlog.Errorf("GetPrivateIPErr:%v", err)
		return "", err
	}

	hostname, _ := os.Hostname()
	sumData := ipList[0] + hostname
	return fmt.Sprintf("%x", md5.Sum([]byte(sumData))), nil
}

func GetSdkServerId(mixStr string) string {
	// 服务器server-id
	serverMachineId, _ := GetMachineID()
	if serverMachineId != "" {
		sumStr := md5.Sum([]byte(serverMachineId + mixStr))
		sumUid, _ := uuid.FromBytes(sumStr[:])
		serverMachineId = sumUid.String()
	}
	appId := os.Getenv("SERVER_MACHINE_ID")
	if appId != "" {
		serverMachineId = appId
	}

	return serverMachineId
}

func GetSysMemory() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}
