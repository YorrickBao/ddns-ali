package main

import (
	"io"
	"log"
	"net/http"
	"os"

	alidns "github.com/alibabacloud-go/alidns-20150109/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/joho/godotenv"
)

func createDnsClient() *alidns.Client {
	accessKeyId := os.Getenv("BAO_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("BAO_ACCESS_KEY_SECRET")
	endpoint := os.Getenv("BAO_ENDPOINT")

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
	client := &http.Client{}
	resp, err := client.Get(os.Getenv("BAO_IP_SERVICE_URL"))
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	return string(body)
}

func fetchDnsRecord(client *alidns.Client) *alidns.DescribeDomainRecordInfoResponseBody {
	recordId := os.Getenv("BAO_RECORD_ID")
	request := alidns.DescribeDomainRecordInfoRequest{
		RecordId: &recordId,
	}
	resq, err := client.DescribeDomainRecordInfo(&request)

	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}

	return resq.Body
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
		log.Fatalf("Error making request: %v", err)
	}
}

func main() {
	log.SetPrefix("[BAO] ")

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	realIp := fetchRealIp()
	log.Println("Real IP: " + realIp)

	dnsClient := createDnsClient()

	dnsRecord := fetchDnsRecord(dnsClient)
	log.Println("DNS Record IP: " + *dnsRecord.Value)
	// recordId := tea.String(os.Getenv("BAO_RECORD_ID"))

	if realIp == *dnsRecord.Value {
		log.Println("IP has not changed")
	} else {
		log.Println("IP has changed, updating record")
		updateDnsRecord(dnsClient, dnsRecord, realIp)
		log.Println("Record updated")
	}
}
