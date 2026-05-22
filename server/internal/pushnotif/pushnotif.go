package pushnotif

import (
	"fmt"
	"os"

	"github.com/SherClockHolmes/webpush-go"
)

func NotifyEmbeddingCompleted(clientSub *webpush.Subscription) error {
	payload := fmt.Append(
		nil,
		`{"title": "Processing complete", "body": "File has been saved and embedded."}`,
	)
	VAPID_PUBLIC_KEY := os.Getenv("VAPID_PUBLIC_KEY")
	VAPID_PRIVATE_KEY := os.Getenv("VAPID_PRIVATE_KEY")

	resp, err := webpush.SendNotification(payload, clientSub, &webpush.Options{
		Subscriber:      "https://example.com",
		VAPIDPublicKey:  VAPID_PUBLIC_KEY,
		VAPIDPrivateKey: VAPID_PRIVATE_KEY,
		TTL:             60,
	})

	if err != nil {
		// retry if you want
		fmt.Printf("error sending push notification %+v", err)
		return err
	}

	defer resp.Body.Close()
	return nil
}

func NotifyEmbeddingFailed(clientSub *webpush.Subscription) error {
	payload := fmt.Append(
		nil,
		`{
			"title": "Processing failed",
			"body": "File could not be indexed. Try again."
		}`,
	)
	VAPID_PUBLIC_KEY := os.Getenv("VAPID_PUBLIC_KEY")
	VAPID_PRIVATE_KEY := os.Getenv("VAPID_PRIVATE_KEY")

	resp, err := webpush.SendNotification(payload, clientSub, &webpush.Options{
		Subscriber:      "https://example.com",
		VAPIDPublicKey:  VAPID_PUBLIC_KEY,
		VAPIDPrivateKey: VAPID_PRIVATE_KEY,
		TTL:             60,
	})

	if err != nil {
		// retry if you want
		fmt.Printf("error sending push notification %+v", err)
		return err
	}

	defer resp.Body.Close()
	return nil
}
