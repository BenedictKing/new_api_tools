package service

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// IPGeoInfo represents IP geolocation information
type IPGeoInfo struct {
	IP          string `json:"ip"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	Region      string `json:"region"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	Org         string `json:"org"`
	ASN         string `json:"asn"`
	IPVersion   string `json:"ip_version,omitempty"`
	Success     bool   `json:"success"`
}

// GeoIP database download URLs (multiple mirrors for reliability)
var geoipDownloadURLs = []string{
	"https://raw.githubusercontent.com/adysec/IP_database/main/geolite/GeoLite2-City.mmdb",
	"https://raw.gitmirror.com/adysec/IP_database/main/geolite/GeoLite2-City.mmdb",
	"https://cdn.jsdelivr.net/gh/adysec/IP_database@main/geolite/GeoLite2-City.mmdb",
}

var geoipASNDownloadURLs = []string{
	"https://raw.githubusercontent.com/adysec/IP_database/main/geolite/GeoLite2-ASN.mmdb",
	"https://raw.gitmirror.com/adysec/IP_database/main/geolite/GeoLite2-ASN.mmdb",
	"https://cdn.jsdelivr.net/gh/adysec/IP_database@main/geolite/GeoLite2-ASN.mmdb",
}

// geoipUpdateInterval is the interval between automatic database updates (24 hours)
const geoipUpdateInterval = 24 * time.Hour

// geoipMinFileSize is the minimum valid database file size (1 MB)
const geoipMinFileSize = 1024 * 1024

// IPGeoService provides IP geolocation queries using MaxMind GeoLite2
type IPGeoService struct {
	cityReader   *geoip2.Reader
	asnReader    *geoip2.Reader
	dbPath       string
	asnDBPath    string
	mu           sync.RWMutex
	available    bool
	asnAvailable bool
	lastError    string
	stopCh       chan struct{}
}

var (
	geoService     *IPGeoService
	geoServiceOnce sync.Once
)

var ipGeoServiceProvider = func() *IPGeoService {
	return GetIPGeoService()
}

// domesticCountryCodes defines Chinese domestic country codes
var domesticCountryCodes = map[string]bool{
	"CN": true,
	"HK": true,
	"MO": true,
	"TW": true,
}

// GetIPGeoService returns the singleton IPGeoService
func GetIPGeoService() *IPGeoService {
	geoServiceOnce.Do(func() {
		geoService = &IPGeoService{}
		geoService.init()
	})
	return geoService
}

func (s *IPGeoService) init() {
	s.stopCh = make(chan struct{})

	// Determine the preferred database directory
	geoipDir := os.Getenv("GEOIP_DATA_DIR")
	if geoipDir == "" {
		geoipDir = "/app/data/geoip"
	}

	// Try to find GeoLite2-City.mmdb in common paths
	paths := []string{
		filepath.Join(geoipDir, "GeoLite2-City.mmdb"),
		"/app/data/geoip/GeoLite2-City.mmdb",
		"./data/geoip/GeoLite2-City.mmdb",
		"/usr/share/GeoIP/GeoLite2-City.mmdb",
	}

	for _, path := range paths {
		if path == "/GeoLite2-City.mmdb" || path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			reader, err := geoip2.Open(path)
			if err != nil {
				fmt.Printf("[GeoIP] Failed to open %s: %v\n", path, err)
				continue
			}
			s.cityReader = reader
			s.dbPath = path
			s.available = true
			s.loadASNReader(filepath.Join(filepath.Dir(path), "GeoLite2-ASN.mmdb"))
			fmt.Printf("[GeoIP] Loaded database: %s\n", path)
			// Start background updater
			go s.backgroundUpdater()
			return
		}
	}

	// Database not found — try to download it
	fmt.Println("[GeoIP] No GeoLite2-City.mmdb found, attempting auto-download...")
	downloadPath := filepath.Join(geoipDir, "GeoLite2-City.mmdb")
	if err := s.downloadDatabase(downloadPath); err != nil {
		fmt.Printf("[GeoIP] Auto-download failed: %v\n", err)
		fmt.Println("[GeoIP] IP geolocation disabled. Will retry in background.")
		s.dbPath = downloadPath
		// Start background updater which will keep retrying
		go s.backgroundUpdater()
		return
	}

	// Load the downloaded database
	reader, err := geoip2.Open(downloadPath)
	if err != nil {
		fmt.Printf("[GeoIP] Failed to open downloaded database: %v\n", err)
		return
	}
	s.cityReader = reader
	s.dbPath = downloadPath
	s.available = true
	s.loadASNReader(filepath.Join(filepath.Dir(downloadPath), "GeoLite2-ASN.mmdb"))
	fmt.Printf("[GeoIP] Database downloaded and loaded: %s\n", downloadPath)

	// Start background updater
	go s.backgroundUpdater()
}

// downloadDatabase downloads the GeoLite2-City.mmdb file from mirror URLs
func (s *IPGeoService) downloadDatabase(destPath string) error {
	return s.downloadDatabaseFrom(destPath, geoipDownloadURLs, geoipMinFileSize)
}

func (s *IPGeoService) downloadASNDatabase(destPath string) error {
	return s.downloadDatabaseFrom(destPath, geoipASNDownloadURLs, geoipMinFileSize)
}

func (s *IPGeoService) downloadDatabaseFrom(destPath string, urls []string, minSize int64) error {
	// Ensure directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	tempPath := destPath + ".tmp"
	defer os.Remove(tempPath) // clean up temp file on any failure

	client := &http.Client{Timeout: 120 * time.Second}

	for _, url := range urls {
		fmt.Printf("[GeoIP] Downloading from %s ...\n", url)

		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("[GeoIP] Download failed from %s: %v\n", url, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			fmt.Printf("[GeoIP] Download failed from %s: HTTP %d\n", url, resp.StatusCode)
			continue
		}

		out, err := os.Create(tempPath)
		if err != nil {
			resp.Body.Close()
			return fmt.Errorf("create temp file: %w", err)
		}

		written, err := io.Copy(out, resp.Body)
		out.Close()
		resp.Body.Close()

		if err != nil {
			fmt.Printf("[GeoIP] Download write failed from %s: %v\n", url, err)
			os.Remove(tempPath)
			continue
		}

		// Validate file size
		if written < minSize {
			fmt.Printf("[GeoIP] Downloaded file too small (%d bytes), skipping\n", written)
			os.Remove(tempPath)
			continue
		}

		// Validate it's a valid mmdb by trying to open it
		testReader, err := geoip2.Open(tempPath)
		if err != nil {
			fmt.Printf("[GeoIP] Downloaded file is not valid mmdb: %v\n", err)
			os.Remove(tempPath)
			continue
		}
		testReader.Close()

		// Atomically replace the old file
		if err := os.Rename(tempPath, destPath); err != nil {
			return fmt.Errorf("rename %s -> %s: %w", tempPath, destPath, err)
		}

		sizeMB := float64(written) / (1024 * 1024)
		fmt.Printf("[GeoIP] Download complete: %.1f MB\n", sizeMB)
		return nil
	}

	return fmt.Errorf("all download mirrors failed")
}

// backgroundUpdater periodically checks and updates the GeoIP database
func (s *IPGeoService) backgroundUpdater() {
	// First check: if database is not available, retry download after 5 minutes
	if !s.IsAvailable() {
		select {
		case <-time.After(5 * time.Minute):
		case <-s.stopCh:
			return
		}
		s.tryUpdateDatabase()
	}

	ticker := time.NewTicker(geoipUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tryUpdateDatabase()
		case <-s.stopCh:
			return
		}
	}
}

// tryUpdateDatabase attempts to download and reload the GeoIP database
func (s *IPGeoService) tryUpdateDatabase() {
	if s.dbPath == "" {
		return
	}

	// Check if the existing database is fresh enough
	if info, err := os.Stat(s.dbPath); err == nil {
		age := time.Since(info.ModTime())
		if age < geoipUpdateInterval {
			return // database is fresh, skip update
		}
	}

	fmt.Println("[GeoIP] Checking for database update...")

	if err := s.downloadDatabase(s.dbPath); err != nil {
		s.setLastError(err)
		fmt.Printf("[GeoIP] Update failed: %v\n", err)
		return
	}

	if s.asnDBPath == "" {
		s.asnDBPath = filepath.Join(filepath.Dir(s.dbPath), "GeoLite2-ASN.mmdb")
	}
	if err := s.downloadASNDatabase(s.asnDBPath); err != nil {
		fmt.Printf("[GeoIP] ASN update skipped: %v\n", err)
	}

	if err := s.Reload(); err != nil {
		s.setLastError(err)
		fmt.Printf("[GeoIP] Failed to reload updated database: %v\n", err)
		return
	}

	fmt.Println("[GeoIP] Database updated and reloaded successfully")
}

func (s *IPGeoService) loadASNReader(path string) {
	s.asnDBPath = path
	if path == "" {
		return
	}
	reader, err := geoip2.Open(path)
	if err != nil {
		return
	}
	s.asnReader = reader
	s.asnAvailable = true
	fmt.Printf("[GeoIP] Loaded ASN database: %s\n", path)
}

func (s *IPGeoService) setLastError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.lastError = ""
		return
	}
	s.lastError = err.Error()
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

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		result.IPVersion = "unknown"
		return result
	}
	result.IPVersion = GetIPVersion(ip)

	// Skip private IPs
	if parsedIP.IsPrivate() || parsedIP.IsLoopback() {
		result.Country = "本地网络"
		result.CountryCode = "LO"
		result.Success = true
		return result
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.available && s.cityReader != nil {
		record, err := s.cityReader.City(parsedIP)
		if err == nil {
			result.Success = true

			// Country
			if name, ok := record.Country.Names["zh-CN"]; ok {
				result.Country = name
			} else if name, ok := record.Country.Names["en"]; ok {
				result.Country = name
			}
			result.CountryCode = record.Country.IsoCode

			// Region/Province
			if len(record.Subdivisions) > 0 {
				if name, ok := record.Subdivisions[0].Names["zh-CN"]; ok {
					result.Region = name
				} else if name, ok := record.Subdivisions[0].Names["en"]; ok {
					result.Region = name
				}
			}

			// City
			if name, ok := record.City.Names["zh-CN"]; ok {
				result.City = name
			} else if name, ok := record.City.Names["en"]; ok {
				result.City = name
			}
		}
	}

	if s.asnAvailable && s.asnReader != nil {
		if asn, err := s.asnReader.ASN(parsedIP); err == nil {
			if asn.AutonomousSystemNumber > 0 {
				result.ASN = fmt.Sprintf("AS%d", asn.AutonomousSystemNumber)
			}
			result.Org = asn.AutonomousSystemOrganization
			result.ISP = asn.AutonomousSystemOrganization
			if result.Org != "" {
				result.Success = true
			}
		}
	}

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

// LookupIPGeo looks up one IP through the configured GeoIP service provider.
func LookupIPGeo(ip string) IPGeoInfo {
	svc := ipGeoServiceProvider()
	if svc == nil {
		return IPGeoInfo{IP: ip}
	}
	return svc.QuerySingle(ip)
}

// LookupIPGeoBatch looks up multiple IPs through the configured GeoIP service provider.
func LookupIPGeoBatch(ips []string) map[string]IPGeoInfo {
	svc := ipGeoServiceProvider()
	if svc == nil {
		results := make(map[string]IPGeoInfo, len(ips))
		for _, ip := range ips {
			results[ip] = IPGeoInfo{IP: ip}
		}
		return results
	}
	return svc.QueryBatch(ips)
}

// IsIPGeoAvailable reports whether the configured GeoIP service is ready.
func IsIPGeoAvailable() bool {
	svc := ipGeoServiceProvider()
	return svc != nil && svc.IsAvailable()
}

// FormatIPGeoInfo returns the stable snake_case response shape used by IP APIs.
func FormatIPGeoInfo(info IPGeoInfo) map[string]interface{} {
	return map[string]interface{}{
		"ip":           info.IP,
		"country":      info.Country,
		"country_code": info.CountryCode,
		"region":       info.Region,
		"city":         info.City,
		"isp":          info.ISP,
		"org":          info.Org,
		"asn":          info.ASN,
		"success":      info.Success,
	}
}

// SetIPGeoServiceProviderForTesting replaces the GeoIP provider and returns a restore function.
func SetIPGeoServiceProviderForTesting(provider func() *IPGeoService) func() {
	old := ipGeoServiceProvider
	ipGeoServiceProvider = provider
	return func() {
		ipGeoServiceProvider = old
	}
}

// Close releases the GeoIP database resources and stops the background updater
// GetIPVersion returns v4, v6, or unknown.
func GetIPVersion(ip string) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "unknown"
	}
	if parsedIP.To4() != nil {
		return "v4"
	}
	return "v6"
}

// GetStatus returns GeoIP database status.
func (s *IPGeoService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]interface{}{
		"city_available": s.available,
		"asn_available":  s.asnAvailable,
		"city_db_path":   s.dbPath,
		"asn_db_path":    s.asnDBPath,
		"last_error":     s.lastError,
	}
}

// Reload reloads City and ASN databases from disk.
func (s *IPGeoService) Reload() error {
	if s.dbPath == "" {
		return fmt.Errorf("city database path is empty")
	}
	cityReader, err := geoip2.Open(s.dbPath)
	if err != nil {
		return err
	}
	var asnReader *geoip2.Reader
	asnAvailable := false
	if s.asnDBPath == "" {
		s.asnDBPath = filepath.Join(filepath.Dir(s.dbPath), "GeoLite2-ASN.mmdb")
	}
	if s.asnDBPath != "" {
		if reader, asnErr := geoip2.Open(s.asnDBPath); asnErr == nil {
			asnReader = reader
			asnAvailable = true
		}
	}

	s.mu.Lock()
	oldCity := s.cityReader
	oldASN := s.asnReader
	s.cityReader = cityReader
	s.asnReader = asnReader
	s.available = true
	s.asnAvailable = asnAvailable
	s.lastError = ""
	s.mu.Unlock()

	if oldCity != nil {
		oldCity.Close()
	}
	if oldASN != nil {
		oldASN.Close()
	}
	return nil
}

// IsDualStackPair checks whether IPv4 and IPv6 likely belong to the same network location.
func (s *IPGeoService) IsDualStackPair(ip1, ip2 string) bool {
	version1 := GetIPVersion(ip1)
	version2 := GetIPVersion(ip2)
	if version1 == "unknown" || version2 == "unknown" || version1 == version2 {
		return false
	}
	info1 := s.QuerySingle(ip1)
	info2 := s.QuerySingle(ip2)
	if !info1.Success || !info2.Success {
		return false
	}
	if info1.ASN != "" && info2.ASN != "" {
		return info1.ASN == info2.ASN && info1.CountryCode == info2.CountryCode
	}
	return info1.CountryCode != "" && info1.CountryCode == info2.CountryCode && info1.Region == info2.Region && info1.City == info2.City
}

// IsDualStackPair checks whether IPv4 and IPv6 likely belong to the same network location.
func IsDualStackPair(ip1, ip2 string) bool {
	return ipGeoServiceProvider().IsDualStackPair(ip1, ip2)
}

func (s *IPGeoService) Close() {
	// Stop background updater
	if s.stopCh != nil {
		select {
		case <-s.stopCh:
			// already closed
		default:
			close(s.stopCh)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cityReader != nil {
		s.cityReader.Close()
		s.cityReader = nil
		s.available = false
	}
	if s.asnReader != nil {
		s.asnReader.Close()
		s.asnReader = nil
		s.asnAvailable = false
	}
}
