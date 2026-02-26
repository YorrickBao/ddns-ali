package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	alidns "github.com/alibabacloud-go/alidns-20150109/v5/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/joho/godotenv"
)

func createDnsClient() *alidns.Client {
	accessKeyId := os.Getenv("BAO_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("BAO_ACCESS_KEY_SECRET")
	endpoint := os.Getenv("BAO_ENDPOINT")

	if accessKeyId == "" || accessKeySecret == "" || endpoint == "" {
		log.Fatalf("Missing required environment variables: BAO_ACCESS_KEY_ID, BAO_ACCESS_KEY_SECRET, BAO_ENDPOINT")
	}

	config := openapi.Config{
		AccessKeyId:     &accessKeyId,
		AccessKeySecret: &accessKeySecret,
		Endpoint:        &endpoint,
	}

	client, err := alidns.NewClient(&config)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	return client
}

func fetchRealIp() string {
	url := os.Getenv("BAO_IP_SERVICE_URL")
	if url == "" {
		log.Fatalf("BAO_IP_SERVICE_URL is not set")
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}
	req.Header.Set("User-Agent", "curl")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		log.Fatalf("Invalid IP address: %s", ip)
	}

	return ip
}

func fetchDnsRecord(client *alidns.Client) *alidns.DescribeDomainRecordInfoResponseBody {
	recordId := os.Getenv("BAO_RECORD_ID")
	if recordId == "" {
		log.Fatalf("BAO_RECORD_ID is not set")
	}

	request := alidns.DescribeDomainRecordInfoRequest{
		RecordId: &recordId,
	}

	maxRetries := 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := client.DescribeDomainRecordInfo(&request)
		if err == nil {
			return resp.Body
		}
		lastErr = err
		log.Printf("DescribeDomainRecordInfo attempt %d failed: %v", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}

	log.Fatalf("DescribeDomainRecordInfo failed after %d retries: %v", maxRetries, lastErr)
	return nil
}

func updateDnsRecord(client *alidns.Client, originalRecord *alidns.DescribeDomainRecordInfoResponseBody, ip string) {
	request := alidns.UpdateDomainRecordRequest{
		RecordId: originalRecord.RecordId,
		Type:     originalRecord.Type,
		Value:    &ip,
		RR:       originalRecord.RR,
		TTL:      originalRecord.TTL,
	}
	_, err := client.UpdateDomainRecord(&request)
	if err != nil {
		log.Fatalf("Error updating DNS record: %v", err)
	}
}

func main() {
	log.SetPrefix("[BAO] ")

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	realIp := fetchRealIp()
	log.Println("Real IP: " + realIp)

	dnsClient := createDnsClient()

	dnsRecord := fetchDnsRecord(dnsClient)
	log.Println("DNS Record IP: " + *dnsRecord.Value)

	if realIp == *dnsRecord.Value {
		log.Println("IP has not changed")
	} else {
		log.Println("IP has changed, updating record")
		updateDnsRecord(dnsClient, dnsRecord, realIp)
		log.Println("Record updated")
	}
}
