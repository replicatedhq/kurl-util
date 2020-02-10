package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

const (
	// SubnetSizeDefault is the default subnet size when unspecified in the subnet-size flag
	SubnetSizeDefault = 22

	// SubnetAllocRangeDefault represents the default ip range from which to allocate subnets
	SubnetAllocRangeDefault = "10.0.0.0/8"
)

func main() {
	subnetSize := flag.Int("subnet-size", SubnetSizeDefault, "subnet size request from subnet range")
	subnetAllocRangeStr := flag.String("subnet-alloc-range", SubnetAllocRangeDefault, "ip range from which to allocate subnets")
	flag.Parse()

	if *subnetSize < 1 || *subnetSize > 32 {
		panic(fmt.Sprintf("subnet size %d invalid", *subnetSize))
	}

	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		panic(errors.Wrap(err, "failed to list routes"))
	}

	_, subnetAllocRange, err := net.ParseCIDR(*subnetAllocRangeStr)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse cidr"))
	}

	subnet, err := FindAvailableSubnet(*subnetSize, subnetAllocRange, routes)
	if err != nil {
		panic(errors.Wrap(err, "failed to find available subnet"))
	}

	fmt.Println(subnet)
}

// FindAvailableSubnet will find an available subnet for a given size in a given range.
func FindAvailableSubnet(subnetSize int, subnetRange *net.IPNet, routes []netlink.Route) (*net.IPNet, error) {
	startIP, _ := cidr.AddressRange(subnetRange)

	_, subnet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", startIP, subnetSize))
	if err != nil {
		return nil, errors.Wrap(err, "parse cidr")
	}
	if checkNetworkFree(subnet, routes) {
		return subnet, nil
	}

	for {
		subnet, _ = cidr.NextSubnet(subnet, subnetSize)
		firstIP, lastIP := cidr.AddressRange(subnet)
		if !subnetRange.Contains(firstIP) || !subnetRange.Contains(lastIP) {
			return nil, errors.New("no available subnet found")
		}

		if checkNetworkFree(subnet, routes) {
			return subnet, nil
		}
	}
}

// checkNetworkFree will return true if it does not overlap any route from the routes passed in as
// an argument.
func checkNetworkFree(subnet *net.IPNet, routes []netlink.Route) bool {
	for _, route := range routes {
		if route.Dst != nil && overlaps(route.Dst, subnet) {
			return false
		}
	}
	return true
}

func overlaps(n1, n2 *net.IPNet) bool {
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}