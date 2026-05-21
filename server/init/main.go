package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SherClockHolmes/webpush-go"
)

func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("error generating VAPID keys:%v", err)
	}

	err = os.WriteFile(
		".env",
		fmt.Appendf(nil, `VAPID_PUBLIC_KEY=%s
VAPID_PRIVATE_KEY=%s`, publicKey, privateKey),
		0644,
	)
	if err != nil {
		log.Fatalf("error writing VAPID keys to .env:%v", err)
	}
}
