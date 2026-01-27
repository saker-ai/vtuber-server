package runtime

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	appconfig "github.com/saker-ai/vtuber-server/internal/config"
	apphttp "github.com/saker-ai/vtuber-server/internal/http"
	applogger "github.com/saker-ai/vtuber-server/internal/logger"
	"github.com/saker-ai/vtuber-server/internal/ws"
)

// Server represents a server.
type Server struct {
	cfg    appconfig.Config
	logger *zap.Logger
	server *http.Server
}

// New executes the new function.
func New(configPath string) (*Server, error) {
	cfg, err := appconfig.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load vtuber config: %w", err)
	}

	logger, err := applogger.New(cfg.Log)
	if err != nil {
		logger, _ = zap.NewProduction()
	}
	logger.Info("vtuber logger configured",
		zap.String("level", cfg.Log.Level),
		zap.Bool("stdout", cfg.Log.Stdout),
		zap.Bool("file_enabled", cfg.Log.File.Enabled),
		zap.String("file_path", cfg.Log.File.Path),
		zap.String("file_name", cfg.Log.File.Name),
	)
	logger.Info("vtuber config loaded",
		zap.String("config_path", configPath),
		zap.String("root_dir", cfg.RootDir),
		zap.String("http_addr", cfg.HTTPAddr),
	)

	wsHandler := ws.NewHandler(logger, cfg)
	router := apphttp.NewRouter(cfg, wsHandler, logger)
	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	return &Server{
		cfg:    cfg,
		logger: logger,
		server: httpServer,
	}, nil
}

// Run executes the run method.
func (s *Server) Run() error {
	if s == nil || s.server == nil {
		return nil
	}

	err := listen(s.server, s.cfg, s.logger)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Addr executes the addr method.
func (s *Server) Addr() string {
	if s == nil || s.server == nil {
		return ""
	}
	return s.server.Addr
}

// Shutdown executes the shutdown method.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return ignoreServerClosed(s.server.Shutdown(ctx))
}

func ignoreServerClosed(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func listen(server *http.Server, cfg appconfig.Config, logger *zap.Logger) error {
	if cfg.TLSDisable {
		if logger != nil {
			logger.Info("starting http server", zap.String("addr", cfg.HTTPAddr))
		}
		return server.ListenAndServe()
	}

	certPath := filepath.Clean(cfg.TLSCertPath)
	keyPath := filepath.Clean(cfg.TLSKeyPath)
	certExists := fileExists(certPath)
	keyExists := fileExists(keyPath)

	if certExists && keyExists {
		if logger != nil {
			logger.Info("starting https server", zap.String("addr", cfg.HTTPAddr))
		}
		return server.ListenAndServeTLS(certPath, keyPath)
	}

	if cfg.TLSRequired {
		missing := []string{}
		if !certExists {
			missing = append(missing, certPath)
		}
		if !keyExists {
			missing = append(missing, keyPath)
		}
		if logger != nil {
			logger.Warn("tls required but certs missing; using in-memory cert", zap.Strings("missing", missing))
		}
	}

	cert, err := generateSelfSignedCert(cfg.SystemConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to generate tls cert: %w", err)
	}
	server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	if logger != nil {
		logger.Info("starting https server with in-memory cert", zap.String("addr", cfg.HTTPAddr))
	}
	return server.ListenAndServeTLS("", "")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func generateSelfSignedCert(host string) (tls.Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	notBefore := time.Now().Add(-time.Minute)
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	dnsNames := []string{"localhost"}
	ipAddresses := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("::1"),
	}

	if host != "" && host != "0.0.0.0" && host != "::" {
		if ip := net.ParseIP(host); ip != nil {
			ipAddresses = appendIP(ipAddresses, ip)
		} else {
			dnsNames = append(dnsNames, host)
		}
	}

	ifaces, _ := net.InterfaceAddrs()
	for _, addr := range ifaces {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsUnspecified() {
			continue
		}
		ipAddresses = appendIP(ipAddresses, ip)
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkixName("mio-local"),
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     uniqueStrings(dnsNames),
		IPAddresses:  uniqueIPs(ipAddresses),
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	return tls.X509KeyPair(certPEM, keyPEM)
}

func pkixName(commonName string) pkix.Name {
	return pkix.Name{
		CommonName:   commonName,
		Organization: []string{"mio"},
	}
}

func appendIP(list []net.IP, ip net.IP) []net.IP {
	for _, existing := range list {
		if existing.Equal(ip) {
			return list
		}
	}
	return append(list, ip)
}

func uniqueIPs(list []net.IP) []net.IP {
	unique := make([]net.IP, 0, len(list))
	for _, ip := range list {
		if ip == nil {
			continue
		}
		unique = appendIP(unique, ip)
	}
	return unique
}

func uniqueStrings(list []string) []string {
	unique := make([]string, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}
