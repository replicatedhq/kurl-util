package main

import (
	"net"
	"reflect"
	"testing"

	"github.com/vishvananda/netlink"
)

func TestFindAvailableSubnet(t *testing.T) {
	type args struct {
		subnetSize  int
		subnetRange *net.IPNet
		routes      []netlink.Route
	}
	tests := []struct {
		name    string
		args    args
		want    *net.IPNet
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				subnetSize:  16,
				subnetRange: mustParseCIDR("10.0.0.0/8"),
				routes:      []netlink.Route{},
			},
			want: mustParseCIDR("10.0.0.0/16"),
		},
		{
			name: "taken",
			args: args{
				subnetSize:  16,
				subnetRange: mustParseCIDR("10.0.0.0/8"),
				routes: []netlink.Route{
					makeRoute("10.0.0.0", 16),
				},
			},
			want: mustParseCIDR("10.1.0.0/16"),
		},
		{
			name: "smaller",
			args: args{
				subnetSize:  22,
				subnetRange: mustParseCIDR("10.0.0.0/8"),
				routes: []netlink.Route{
					makeRoute("10.0.0.0", 22),
					makeRoute("10.0.4.0", 22),
				},
			},
			want: mustParseCIDR("10.0.8.0/22"),
		},
		{
			name: "gap",
			args: args{
				subnetSize:  24,
				subnetRange: mustParseCIDR("10.0.0.0/8"),
				routes: []netlink.Route{
					makeRoute("10.0.0.0", 24),
					makeRoute("10.0.2.0", 24),
				},
			},
			want: mustParseCIDR("10.0.1.0/24"),
		},
		{
			name: "none available",
			args: args{
				subnetSize:  16,
				subnetRange: mustParseCIDR("10.0.0.0/8"),
				routes: []netlink.Route{
					makeRoute("10.0.0.0", 8),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindAvailableSubnet(tt.args.subnetSize, tt.args.subnetRange, tt.args.routes)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAvailableSubnet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindAvailableSubnet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustParseCIDR(s string) *net.IPNet {
	_, subnet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return subnet
}

func makeRoute(ip string, bits int) netlink.Route {
	dst := &net.IPNet{
		IP:   net.ParseIP(ip),
		Mask: net.CIDRMask(bits, 32),
	}

	src := net.ParseIP(ip)
	return netlink.Route{LinkIndex: 0, Dst: dst, Src: src}
}
