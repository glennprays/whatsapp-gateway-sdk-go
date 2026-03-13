// Example usage of the WhatsApp Gateway SDK
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/glennprays/whatsapp-gateway-sdk-go"
)

func main() {
	// Create client with custom options
	client := waga.NewClient(
		waga.WithBaseURL("http://localhost:3000/api/v1"),
		waga.WithTimeout(30*time.Second),
	)

	// Example 1: Register a new phone number and get JWT token
	fmt.Println("=== Registration Example ===")
	err := registrationExample(client)
	if err != nil {
		log.Printf("Registration example failed: %v", err)
	}

	// Example 2: Using a pre-existing token
	fmt.Println("\n=== Using Existing Token ===")
	existingTokenExample()

	// Example 3: WhatsApp Authentication
	fmt.Println("\n=== WhatsApp Authentication ===")
	whatsappAuthExample(client)

	// Example 4: Sending Messages
	fmt.Println("\n=== Sending Messages ===")
	sendMessageExample(client)

	// Example 5: Webhook Management
	fmt.Println("\n=== Webhook Management ===")
	webhookExample(client)

	// Example 6: Webhook Verification (server-side)
	fmt.Println("\n=== Webhook Verification ===")
	webhookVerificationExample()
}

func registrationExample(client *waga.Client) error {
	ctx := context.Background()

	// Register with phone number and secret key
	resp, err := client.Register(ctx, "6281234567890", "your_secret_key")
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	fmt.Printf("Registration successful! Token: %s...\n", resp.Token[:50])
	fmt.Println("Token is now automatically stored in the client.")

	return nil
}

func existingTokenExample() {
	// If you already have a JWT token, use WithToken option
	_ = waga.NewClient(
		waga.WithBaseURL("http://localhost:3000/api/v1"),
		waga.WithToken("your_existing_jwt_token"),
	)

	// Or set it later
	client2 := waga.NewClient()
	client2.SetToken("your_existing_jwt_token")

	fmt.Println("Client configured with existing token.")
}

func whatsappAuthExample(client *waga.Client) {
	ctx := context.Background()

	// Check login status
	status, err := client.GetLoginStatus(ctx)
	if err != nil {
		log.Printf("Failed to get login status: %v", err)
		return
	}
	fmt.Printf("Authenticated: %v\n", status.Authenticated)

	if !status.Authenticated {
		// Get QR code for login
		qr, err := client.GetQRCode(ctx, "json")
		if err != nil {
			log.Printf("Failed to get QR code: %v", err)
			return
		}
		fmt.Printf("QR Code (base64): %s...\n", qr.QrCode[:50])
		fmt.Printf("Expires in: %d seconds\n", qr.ExpiresIn)

		// Alternative: Get pair code
		pair, err := client.GetPairCode(ctx)
		if err != nil {
			log.Printf("Failed to get pair code: %v", err)
			return
		}
		fmt.Printf("Pair Code: %s (expires in %d seconds)\n", pair.PairCode, pair.ExpiresIn)
	}
}

func sendMessageExample(client *waga.Client) {
	ctx := context.Background()

	// Format recipient MSISDN
	recipient := waga.FormatMSISDN("6289876543210")

	// Send text message
	resp, err := client.SendText(ctx, recipient, "Hello from Go SDK!")
	if err != nil {
		log.Printf("Failed to send text: %v", err)
		return
	}
	fmt.Printf("Message sent! ID: %s, Success: %v\n", resp.MessageId, resp.Success)

	// Edit message
	err = client.EditMessage(ctx, recipient, resp.MessageId, "Hello from Go SDK! (edited)")
	if err != nil {
		log.Printf("Failed to edit message: %v", err)
	}

	// React to message
	err = client.React(ctx, recipient, resp.MessageId, "👍")
	if err != nil {
		log.Printf("Failed to react: %v", err)
	}

	// Delete message
	err = client.DeleteMessage(ctx, recipient, resp.MessageId)
	if err != nil {
		log.Printf("Failed to delete message: %v", err)
	}
}

func webhookExample(client *waga.Client) {
	ctx := context.Background()

	// Register webhook URL with HMAC secret
	err := client.RegisterWebhook(ctx, "https://example.com/webhook", "my_hmac_secret")
	if err != nil {
		log.Printf("Failed to register webhook: %v", err)
		return
	}
	fmt.Println("Webhook registered successfully!")

	// Get registered webhook
	webhook, err := client.GetWebhook(ctx)
	if err != nil {
		log.Printf("Failed to get webhook: %v", err)
		return
	}
	fmt.Printf("Registered webhook URL: %s\n", webhook.URL)

	// Unregister webhook when done
	// err = client.UnregisterWebhook(ctx)
	// if err != nil {
	// 	log.Printf("Failed to unregister webhook: %v", err)
	// }
}

func webhookVerificationExample() {
	// This is how you verify incoming webhooks on your server
	verifier := waga.NewWebhookVerifier("my_hmac_secret")

	// Example: HTTP handler for receiving webhooks
	// func handleWebhook(w http.ResponseWriter, r *http.Request) {
	//     body, _ := io.ReadAll(r.Body)
	//     signature := r.Header.Get("X-Webhook-Signature")
	//
	//     // Verify and parse incoming message webhook
	//     payload, err := verifier.ParseIncomingWebhook(body, signature)
	//     if err != nil {
	//         if errors.Is(err, sdk.ErrInvalidSignature) {
	//             http.Error(w, "Invalid signature", 401)
	//             return
	//         }
	//         http.Error(w, "Invalid payload", 400)
	//         return
	//     }
	//
	//     fmt.Printf("Received message from %s: %s\n", payload.From, payload.Text)
	// }

	// Simulated verification
	payload := []byte(`{"event":"message.incoming","from":"6281234567890","text":"Hello!"}`)
	signature := waga.ComputeSignature(payload, "my_hmac_secret")
	fmt.Printf("Computed signature: %s\n", signature)

	// Verify
	if verifier.VerifySignature(payload, signature) {
		fmt.Println("Signature verified successfully!")
	}
}

// Error handling example
func errorHandlingExample(client *waga.Client) {
	ctx := context.Background()

	_, err := client.SendText(ctx, "invalid", "test")
	if err != nil {
		var sdkErr *waga.SDKError
		if waga.IsUnauthorized(err) {
			fmt.Println("Token expired or invalid, need to re-register")
		} else if waga.IsBadRequest(err) {
			fmt.Println("Invalid request parameters")
		} else if waga.IsRateLimited(err) {
			fmt.Println("Rate limited, should retry later")
		} else if errors.As(err, &sdkErr) {
			fmt.Printf("API error: code=%d, message=%s\n", sdkErr.Code, sdkErr.Message)
		} else {
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}
}

// Reconnect and logout example
func sessionManagementExample(client *waga.Client) {
	ctx := context.Background()

	// Reconnect to existing session
	err := client.Reconnect(ctx)
	if err != nil {
		log.Printf("Failed to reconnect: %v", err)
	}

	// Logout from session
	err = client.Logout(ctx)
	if err != nil {
		log.Printf("Failed to logout: %v", err)
	}
}

// Health check example
func healthCheckExample(client *waga.Client) {
	ctx := context.Background()

	health, err := client.Health(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}
	fmt.Printf("Gateway status: %s, timestamp: %s\n", health.Status, health.Timestamp)
}
