package main

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
)

const (
	dataDir   = "/app/data"
	repoOwner = "sapics"
	repoName  = "ip-location-db"
	branch    = "main"
)

type fileInfo struct {
	RemotePath string
	LocalName  string
}

var files = []fileInfo{
	{"geo-whois-asn-country/geo-whois-asn-country-ipv4-num.csv", "geo-whois-asn-country-ipv4-num.csv"},
	{"geo-asn-country/geo-asn-country-ipv6-num.csv", "geo-asn-country-ipv6-num.csv"},
}

type githubContent struct {
	SHA         string `json:"sha"`
	DownloadURL string `json:"download_url"`
}

// updateCsvFiles ensures CSV files exist in dataDir and updates them if needed
func updateCsvFiles() error {
	autoUpdate := strings.ToLower(strings.TrimSpace(os.Getenv("AUTO_UPDATE"))) == "true"

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	for _, fi := range files {
		localPath := filepath.Join(dataDir, fi.LocalName)
		// Check local existence
		_, err := os.Stat(localPath)
		exists := err == nil

		// If file missing or auto-update enabled, check remote
		if !exists || autoUpdate {
			// Fetch remote metadata
			apiURL := fmt.Sprintf(
				"https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
				repoOwner, repoName, fi.RemotePath, branch,
			)
			resp, err := http.Get(apiURL)
			if err != nil {
				return fmt.Errorf("fetching remote metadata: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("bad status from GitHub API: %s", resp.Status)
			}

			var meta githubContent
			if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
				return fmt.Errorf("decoding GitHub response: %w", err)
			}

			download := true
			if exists {
				data, err := os.ReadFile(localPath)
				if err == nil {
					header := fmt.Sprintf("blob %d\x00", len(data))
					h := sha1.Sum(append([]byte(header), data...))
					localSha := hex.EncodeToString(h[:])
					download = localSha != meta.SHA
				}
			}

			if download {
				// Download new file
				dlResp, err := http.Get(meta.DownloadURL)
				if err != nil {
					return fmt.Errorf("downloading file: %w", err)
				}
				defer dlResp.Body.Close()

				out, err := os.Create(localPath)
				if err != nil {
					return fmt.Errorf("creating local file: %w", err)
				}
				defer out.Close()

				if _, err := io.Copy(out, dlResp.Body); err != nil {
					return fmt.Errorf("writing file: %w", err)
				}
				log.Printf("updated %s", fi.LocalName)
			}
		}
	}
	return nil
}

type IpAddressRange struct {
	start   *big.Int
	end     *big.Int
	country string
}

// loadCsv reads local CSVs and returns sorted ranges
func loadCsv() []IpAddressRange {
	arr := []IpAddressRange{}

	for _, fi := range files {
		path := filepath.Join(dataDir, fi.LocalName)
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()

		r := csv.NewReader(f)
		for {
			rec, err := r.Read()
			if err != nil {
				break
			}
			start, ok := new(big.Int).SetString(rec[0], 10)
			if !ok {
				continue
			}
			end, ok := new(big.Int).SetString(rec[1], 10)
			if !ok {
				continue
			}
			arr = append(arr, IpAddressRange{start, end, rec[2]})
		}
	}

	sort.Slice(arr, func(i, j int) bool {
		return arr[i].start.Cmp(arr[j].start) < 0
	})

	return arr
}

type IpAddress struct {
	IpAddr *string `json:"ip_addr"`
	IpV6   bool    `json:"ip_v6"`
}

func parseIpAddress(rawIpAddr string) *IpAddress {
	v := validator.New()

	err := v.Var(rawIpAddr, "required,ip4_addr")
	if err == nil {
		return &IpAddress{IpAddr: &rawIpAddr, IpV6: false}
	}
	err = v.Var(rawIpAddr, "required,ip6_addr")
	if err == nil {
		return &IpAddress{IpAddr: &rawIpAddr, IpV6: true}
	}
	return nil
}

type ApiResponse struct {
	Ok      bool    `json:"ok"`
	Country *string `json:"country"`
	IpAddress
}

func main() {
	if err := updateCsvFiles(); err != nil {
		log.Fatalf("failed to update CSVs: %v", err)
	}

	arr := loadCsv()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{},
		AllowCredentials: true,
	}))

	r.GET("/getIpInfo", func(c *gin.Context) {
		ipAddr := parseIpAddress(c.Query("addr"))
		if ipAddr != nil && ipAddr.IpAddr != nil {
			addr := net.ParseIP(*ipAddr.IpAddr)
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
					c.JSON(http.StatusOK, ApiResponse{Ok: true, Country: &arr[idx-1].country, IpAddress: *ipAddr})
					return
				}
			}
		}
		c.JSON(http.StatusOK, ApiResponse{Ok: false})
	})

	r.Run(":8080")
}
