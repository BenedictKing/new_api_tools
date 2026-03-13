package service

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// IPGeoInfo represents IP geolocation information
// 仅保留 Country 级别信息
// ASN 由 pkg/geoip 处理（用于双栈识别）
type IPGeoInfo struct {
	IP          string
	Country     string
	CountryCode string
	Success     bool
}

// IPGeoService provides IP geolocation queries using MaxMind GeoLite2
// 使用 GeoLite2-Country.mmdb
// 用于基础国家识别（城市信息已移除）
type IPGeoService struct {
	countryReader *geoip2.Reader
	mu            sync.RWMutex
	available     bool
}

var (
	geoService     *IPGeoService
	geoServiceOnce sync.Once
)

// GetIPGeoService returns the singleton IPGeoService
func GetIPGeoService() *IPGeoService {
	geoServiceOnce.Do(func() {
		geoService = &IPGeoService{}
		geoService.init()
	})
	return geoService
}

func (s *IPGeoService) init() {
	// Try to find GeoLite2-Country.mmdb in common paths
	paths := []string{
		os.Getenv("GEOIP_DATA_DIR") + "/GeoLite2-Country.mmdb",
		"/app/data/geoip/GeoLite2-Country.mmdb",
		"./data/geoip/GeoLite2-Country.mmdb",
		"/usr/share/GeoIP/GeoLite2-Country.mmdb",
	}

	for _, path := range paths {
		if path == "/GeoLite2-Country.mmdb" {
			continue // skip empty GEOIP_DATA_DIR + path
		}
		if _, err := os.Stat(path); err == nil {
			reader, err := geoip2.Open(path)
			if err != nil {
				fmt.Printf("[GeoIP] Failed to open %s: %v\n", path, err)
				continue
			}
			s.countryReader = reader
			s.available = true
			fmt.Printf("[GeoIP] Loaded database: %s\n", path)
			return
		}
	}
	fmt.Println("[GeoIP] No GeoLite2-Country.mmdb found, IP geolocation disabled")
}

// IsAvailable returns whether the GeoIP service is available
func (s *IPGeoService) IsAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.available
}

// QuerySingle looks up a single IP address
func (s *IPGeoService) QuerySingle(ip string) IPGeoInfo {
	result := IPGeoInfo{IP: ip}

	if !s.available || s.countryReader == nil {
		return result
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return result
	}

	// Skip private IPs
	if parsedIP.IsPrivate() || parsedIP.IsLoopback() {
		result.Country = "本地网络"
		result.CountryCode = "LO"
		result.Success = true
		return result
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	record, err := s.countryReader.Country(parsedIP)
	if err != nil {
		return result
	}

	result.Success = true

	// Country
	if name, ok := record.Country.Names["zh-CN"]; ok {
		result.Country = name
	} else if name, ok := record.Country.Names["en"]; ok {
		result.Country = name
	}
	result.CountryCode = record.Country.IsoCode

	return result
}

// QueryBatch looks up multiple IPs and returns a map of IP -> IPGeoInfo
func (s *IPGeoService) QueryBatch(ips []string) map[string]IPGeoInfo {
	results := make(map[string]IPGeoInfo, len(ips))
	for _, ip := range ips {
		results[ip] = s.QuerySingle(ip)
	}
	return results
}

// Close releases the GeoIP database resources
func (s *IPGeoService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.countryReader != nil {
		s.countryReader.Close()
		s.countryReader = nil
		s.available = false
	}
}
