package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/vishvananda/netlink"
)

const (
	// SubnetSizeDefault is the default subnet size when unspecified in the subnet-size flag
	SubnetSizeDefault = 22

	// SubnetRange represents the cidr range from which we can allocate subnets
	SubnetRange = "10.0.0.0/8"
)

func main() {
	subnetSize := flag.Int("subnet-size", SubnetSizeDefault, "subnet size ... TODO")
	flag.Parse()

	if subnetSize == nil || *subnetSize < 1 || *subnetSize > 32 {
		panic(fmt.Sprintf("subnet size %v invalid", subnetSize))
	}

	routes, err := getRoutes(nil)
	if err != nil {
		log.Panic(err)
	}

	_, subnetRange, err := net.ParseCIDR(SubnetRange)
	if err != nil {
		log.Panic(err)
	}

	subnet, err := FindAvailableSubnet(*subnetSize, subnetRange, routes)
	if err != nil {
		log.Panic(err)
	}

	fmt.Println(subnet)
}

// FindAvailableSubnet will find an available subnet for a given size in a
// given range.
func FindAvailableSubnet(subnetSize int, subnetRange *net.IPNet, routes []netlink.Route) (*net.IPNet, error) {
	startIP, _ := cidr.AddressRange(subnetRange)

	_, subnet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", startIP, subnetSize))
	if err != nil {
		return nil, err
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
			fmt.Println("FREE", subnet)
			return subnet, nil
		}
	}
}

// checkNetworkFree will return true if it does not overlap any route
// from the routes passed in as an argument.
func checkNetworkFree(subnet *net.IPNet, routes []netlink.Route) bool {
	for _, route := range routes {
		if route.Dst != nil && overlaps(route.Dst, subnet) {
			return false
		}
	}
	return true
}

// getRoutes will return all routes from the host with ignoreIfaceNames
// excluded.
func getRoutes(ignoreIfaceNames map[string]struct{}) ([]netlink.Route, error) {
	ignoreIfaceIndices := make(map[int]struct{})
	for ifaceName := range ignoreIfaceNames {
		if iface, err := net.InterfaceByName(ifaceName); err == nil {
			ignoreIfaceIndices[iface.Index] = struct{}{}
		}
	}
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}
	var filtered []netlink.Route
	for _, route := range routes {
		if _, found := ignoreIfaceIndices[route.LinkIndex]; found {
			continue
		}
		filtered = append(filtered, route)
	}
	return filtered, nil
}

func overlaps(n1, n2 *net.IPNet) bool {
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}
