package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"net"
	"os"
	"strings"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

const (
	// CIDRRangeDefault is the default subnet size when unspecified in the subnet-size flag
	CIDRRangeDefault = 22

	// SubnetAllocRangeDefault represents the default ip range from which to allocate subnets
	SubnetAllocRangeDefault = "10.0.0.0/8"
)

func main() {
	cidrRangeFlag := flag.Int("cidr-range", CIDRRangeDefault, "the cidr range to request from the ip range specified by subnet-alloc-range")
	subnetAllocRangeFlag := flag.String("subnet-alloc-range", SubnetAllocRangeDefault, "ip range from which to allocate subnets")
	excludeSubnetFlag := flag.String("exclude-subnet", "", "comma separated list of subnets to exclude")
	debugFlag := flag.Bool("debug", false, "enable debug logging")

	flag.Parse()

	cidrRange := *cidrRangeFlag
	debug := *debugFlag

	if cidrRange < 1 || cidrRange > 32 {
		panic(fmt.Sprintf("subnet-size %d invalid", cidrRange))
	}

	_, subnetAllocRange, err := net.ParseCIDR(*subnetAllocRangeFlag)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse subnet-alloc-range cidr"))
	}

	var excludeSubnets []*net.IPNet
	if *excludeSubnetFlag != "" {
		for _, s := range strings.Split(*excludeSubnetFlag, ",") {
			_, subnet, err := net.ParseCIDR(s)
			if err != nil {
				panic(errors.Wrap(err, "failed to parse exclude-subnet cidr"))
			}
			excludeSubnets = append(excludeSubnets, subnet)
		}
	}

	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		panic(errors.Wrap(err, "failed to list routes"))
	}
	if debug {
		for _, route := range routes {
			fmt.Fprintf(os.Stderr, "Found route %s\n", route)
		}
	}

	for _, subnet := range excludeSubnets {
		route := netlink.Route{
			Src: subnet.IP,
			Dst: subnet,
		}
		if debug {
			fmt.Fprintf(os.Stderr, "Exluding additional route %s\n", route)
		}
		routes = append(routes, route)
	}

	subnet, err := FindAvailableSubnet(cidrRange, subnetAllocRange, routes, debug)
	if err != nil {
		panic(errors.Wrap(err, "failed to find available subnet"))
	}

	fmt.Print(subnet)
}

// FindAvailableSubnet will find an available subnet for a given size in a given range.
func FindAvailableSubnet(cidrRange int, subnetRange *net.IPNet, routes []netlink.Route, debug bool) (*net.IPNet, error) {
	startIP, _ := cidr.AddressRange(subnetRange)

	_, subnet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", startIP, cidrRange))
	if err != nil {
		return nil, errors.Wrap(err, "parse cidr")
	}
	if debug {
		fmt.Fprintf(os.Stderr, "First subnet %s\n", subnet)
	}

	for {
		firstIP, lastIP := cidr.AddressRange(subnet)
		if !subnetRange.Contains(firstIP) || !subnetRange.Contains(lastIP) {
			return nil, errors.New("no available subnet found")
		}

		route := findFirstOverlappingRoute(subnet, routes)
		if route == nil {
			return subnet, nil
		}
		if len(subnet.IP) == net.IPv4len {
			if route.Dst.IP.To4() == nil {
				continue
			}
			route.Dst.IP = route.Dst.IP.To4()
		}
		if debug {
			fmt.Fprintf(os.Stderr, "Route %s overlaps with subnet %s\n", *route, subnet)
		}

		firstIP, lastIP = cidr.AddressRange(route.Dst)
		subnet, _ = cidr.NextSubnet(route.Dst, cidrRange)
		if debug {
			fmt.Fprintf(os.Stderr, "Next subnet %s\n", subnet)
		}
	}
}

func getLastIP(subnet *net.IPNet) net.IP {
	ones, _ := subnet.Mask.Size()
	networksize := math.Pow(2, float64(32-ones))

	// Reduce network size by one as we really need last IP address in the range,
	// not first one of subsequent range
	return int2ip(ip2int(subnet.IP) + uint32(networksize) - 1)
}

func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

// findFirstOverlappingRoute will return the first overlapping route with the subnet specified
func findFirstOverlappingRoute(subnet *net.IPNet, routes []netlink.Route) *netlink.Route {
	for _, route := range routes {
		if route.Dst != nil && overlaps(route.Dst, subnet) {
			return &route
		}
	}
	return nil
}

func overlaps(n1, n2 *net.IPNet) bool {
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}
