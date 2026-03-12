package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

func main() {
	listenAddr := flag.String("listen", "0.0.0.0:8088", "listen address")
	trayEnabled := flag.Bool("tray", true, "show a system tray icon")
	flag.Parse()

	if err := InitX11(); err != nil {
		log.Fatalf("X11 init failed: %v", err)
	}
	defer CloseX11()

	typer := NewTyper()
	server := NewServer(typer)
	httpServer := &http.Server{
		Addr:              *listenAddr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	printURLs(*listenAddr)

	var tray *Tray
	if *trayEnabled {
		tray = StartTray("goremotetype", trayText(*listenAddr))
		typer.SetTray(tray)
		defer tray.Close()
		log.Printf("tray started")
	}

	errCh := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errCh:
			if err != nil {
				log.Fatalf("server error: %v", err)
			}
			goto shutdown
		case sig := <-sigCh:
			log.Printf("received signal %s, shutting down", sig)
			goto shutdown
		case enabled := <-trayToggleRequested(tray):
			typer.SetEnabled(enabled)
			log.Printf("typing enabled=%t", enabled)
		case <-trayQuitRequested(tray):
			log.Printf("tray requested shutdown")
			goto shutdown
		}
	}

shutdown:
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}

func printURLs(listenAddr string) {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		log.Printf("serving on http://%s", listenAddr)
		return
	}

	log.Printf("goremotetype listening on http://localhost:%s", port)
	for _, ip := range lanIPs(host) {
		log.Printf("goremotetype LAN URL: http://%s:%s", ip, port)
	}
}

func trayText(listenAddr string) string {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return fmt.Sprintf("goremotetype running on %s", listenAddr)
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		return fmt.Sprintf("goremotetype listening on http://localhost:%s", port)
	}

	return fmt.Sprintf("goremotetype listening on http://%s:%s", host, port)
}

func trayQuitRequested(tray *Tray) <-chan struct{} {
	if tray == nil {
		return nil
	}
	return tray.QuitRequested()
}

func trayToggleRequested(tray *Tray) <-chan bool {
	if tray == nil {
		return nil
	}
	return tray.ToggleRequested()
}

func lanIPs(host string) []string {
	host = strings.TrimSpace(host)
	if host != "" && host != "0.0.0.0" && host != "::" {
		if ip := net.ParseIP(host); ip != nil {
			return []string{ip.String()}
		}
		return []string{host}
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	set := map[string]struct{}{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil || ip == nil {
				continue
			}
			v4 := ip.To4()
			if v4 == nil {
				continue
			}
			set[v4.String()] = struct{}{}
		}
	}

	ips := make([]string, 0, len(set))
	for ip := range set {
		ips = append(ips, ip)
	}
	sort.Strings(ips)
	return ips
}

func usageText() string {
	return `goremotetype relays phone typing to your X11 desktop via XTEST.`
}

func init() {
	flag.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, usageText())
		flag.PrintDefaults()
	}
}
