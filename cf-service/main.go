package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"
	docker "github.com/fsouza/go-dockerclient"
	"golang.org/x/net/publicsuffix"
)

func getZoneIDByDomain(api *cloudflare.API, domain string) (string, error) {
	zone, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return "", err
	}

	id, err := api.ZoneIDByName(zone)
	if err != nil {
		return "", err
	}
	return id, nil
}

func addARecord(api *cloudflare.API, domain string, ip string) error {
	zoneID, err := getZoneIDByDomain(api, domain)
	if err != nil {
		return err
	}

	_, err = api.CreateDNSRecord(zoneID, cloudflare.DNSRecord{Type: "A", Name: domain, Content: ip, Proxied: true})
	if err != nil {
		return err
	}
	return nil
}

func removeARecord(api *cloudflare.API, domain string, ip string) error {
	zoneID, err := getZoneIDByDomain(api, domain)
	if err != nil {
		return err
	}

	records, err := api.DNSRecords(zoneID, cloudflare.DNSRecord{Name: domain, Content: ip})
	if err != nil {
		return err
	}

	if len(records) != 1 {
		return fmt.Errorf("Records count error: %v", len(records))
	}

	return api.DeleteDNSRecord(zoneID, records[0].ID)
}

func getVirtualHost(client *docker.Client, id string) (string, error) {
	container, err := client.InspectContainer(id)
	if err != nil {
		return "", err
	}

	for _, ev := range container.Config.Env {
		if strings.HasPrefix(ev, "VIRTUAL_HOST") {
			return strings.Split(ev, "=")[1], nil
		}
	}
	return "", errors.New("No VIRTUAL_HOST env var")
}

func main() {
	endpoint := "unix:///tmp/docker.sock"
	client, err := docker.NewClient(endpoint)

	if err != nil {
		panic(err)
	}

	cfAPI, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		panic(err)
	}

	events := make(chan *docker.APIEvents)

	client.AddEventListener(events)

	proxyIP := os.Getenv("PROXY_IP")
	for msg := range events {
		if msg.Type == "container" {
			if msg.Action == "start" {
				domain, err := getVirtualHost(client, msg.ID)
				if err != nil {
					log.Printf("%v: %v", domain, err)
					continue
				}
				err = addARecord(cfAPI, domain, proxyIP)
				if err != nil {
					log.Printf("%v: %v", domain, err)
					continue
				}

				log.Printf("Added %s %s", domain, proxyIP)
			} else if msg.Action == "die" {
				domain, err := getVirtualHost(client, msg.ID)
				if err != nil {
					log.Printf("%v: %v", domain, err)
					continue
				}
				err = removeARecord(cfAPI, domain, proxyIP)
				if err != nil {
					log.Printf("%v: %v", domain, err)
					continue
				}

				log.Printf("Removed %s %s", domain, proxyIP)
			}
		}

	}
}
