package main

import (
	"encoding/binary"
	"encoding/csv"
	"math/big"
	"net"
	"net/http"
	"sort"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/guregu/null/v5"
)

type IpAddressRange struct {
	start   *big.Int
	end     *big.Int
	country string
}

func downloadCsv() []IpAddressRange {
	arr := []IpAddressRange{}

	urls := []string{
		"https://cdn.jsdelivr.net/gh/sapics/ip-location-db/geo-whois-asn-country/geo-whois-asn-country-ipv4-num.csv",
		"https://cdn.jsdelivr.net/gh/sapics/ip-location-db/geo-asn-country/geo-asn-country-ipv6-num.csv",
	}

	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			return arr
		}
		defer resp.Body.Close()

		r := csv.NewReader(resp.Body)

		for {
			record, err := r.Read()
			if err != nil {
				break
			}
			start, ok := new(big.Int).SetString(record[0], 10)
			if !ok {
				continue
			}
			end, ok := new(big.Int).SetString(record[1], 10)
			if !ok {
				continue
			}
			arr = append(arr, IpAddressRange{start, end, record[2]})
		}
	}

	return arr
}

type IpAddress struct {
	IpAddr null.String `json:"ip_addr"`
	IpV6   null.Bool   `json:"ip_v6"`
}

func parseIpAddress(rawIpAddr string) *IpAddress {
	v := validator.New()

	err := v.Var(rawIpAddr, "required,ip4_addr")
	if err == nil {
		return &IpAddress{
			IpAddr: null.StringFrom(rawIpAddr),
			IpV6:   null.BoolFrom(false),
		}
	}

	err = v.Var(rawIpAddr, "required,ip6_addr")
	if err == nil {
		return &IpAddress{
			IpAddr: null.StringFrom(rawIpAddr),
			IpV6:   null.BoolFrom(true),
		}
	}

	return nil
}

type ApiResponse struct {
	Ok      bool        `json:"ok"`
	Country null.String `json:"country"`
	IpAddress
}

func main() {
	arr := downloadCsv()

	sort.Slice(arr, func(i, j int) bool {
		return arr[i].start.Cmp(arr[j].start) == -1
	})

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{},
		AllowCredentials: true,
	}))

	r.GET("/getIpInfo", func(c *gin.Context) {
		ipAddress := parseIpAddress(c.Query("addr"))

		if ipAddress != nil && ipAddress.IpAddr.Valid {
			addr := net.ParseIP(ipAddress.IpAddr.String)
			if addr != nil {
				ipNum := big.NewInt(0)

				if addr.To4() != nil {
					ipNum = new(big.Int).SetUint64(uint64(binary.BigEndian.Uint32(addr.To4())))
				} else {
					ipNum.SetBytes(addr)
				}

				idx := sort.Search(len(arr), func(i int) bool {
					return arr[i].start.Cmp(ipNum) > 0
				})

				if idx > 0 && arr[idx-1].end.Cmp(ipNum) >= 0 && ipNum.Sign() != 0 {
					c.JSON(http.StatusOK, ApiResponse{
						Ok:        true,
						Country:   null.StringFrom(arr[idx-1].country),
						IpAddress: *ipAddress,
					})
					return
				}
			}
		}

		c.JSON(http.StatusOK, ApiResponse{
			Ok: false,
		})
	})

	r.Run(":8080")
}
