package main

import (
	"context"
	"fmt"
	"log"
	// "net/http"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

var fcmClient *messaging.Client

// DATABASE SEMENTARA (Mapping UserID ke Token)
var userTokens = make(map[string]string) 

// Struct Request
type TokenReq struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

type SendUserReq struct {
	TargetUserID string `json:"target_user_id"` // Kita kirim ke ID, bukan Token
	Title        string `json:"title"`
	Body         string `json:"body"`
}

type BroadcastReq struct {
	Topic string `json:"topic"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func initFirebase() {
	opt := option.WithCredentialsFile("serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Gagal init Firebase: %v\n", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("Gagal init Messaging client: %v\n", err)
	}
	fcmClient = client
	fmt.Println("✅ Firebase Admin SDK Siap!")
}

func main() {
	initFirebase()
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// 1. REGISTER: Simpan pasangan UserID dan Token
	r.POST("/register", func(c *gin.Context) {
		var req TokenReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// LOGIC SIMPAN KE MAP (Database Memory)
		userTokens[req.UserID] = req.Token
		
		fmt.Printf("📝 User Terdaftar: [%s] -> Token: ...%s\n", req.UserID, req.Token[len(req.Token)-10:])
		c.JSON(200, gin.H{"message": "User registered successfully"})
	})

	// 2. KIRIM KE USER SPESIFIK (By ID)
	r.POST("/send-user", func(c *gin.Context) {
		var req SendUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// A. CARI TOKEN DI DATABASE (Map) BERDASARKAN USER ID
		token, exists := userTokens[req.TargetUserID]
		if !exists {
			// Jika User ID tidak ditemukan di memori
			c.JSON(404, gin.H{"error": fmt.Sprintf("User ID '%s' belum terdaftar/connect", req.TargetUserID)})
			return
		}

		// B. JIKA KETEMU, BUAT PESAN
		message := &messaging.Message{
			Token: token, // Gunakan token hasil pencarian sebelumnya
			Notification: &messaging.Notification{
				Title: req.Title,
				Body:  req.Body,
			},
			Data: map[string]string{
				"screen": "chat",
			},
		}

		// C. KIRIM
		responseID, err := fcmClient.Send(context.Background(), message)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		fmt.Printf("🚀 Terkirim ke User: %s (MsgID: %s)\n", req.TargetUserID, responseID)
		c.JSON(200, gin.H{"status": "sent", "target_user": req.TargetUserID})
	})

	// CONTOH HARDCODE untuk kirim pesan dg target id terterntu
	// Panggil url: http://localhost:8081/msg-custom
	r.GET("/msg-custom", func(c *gin.Context) {
		// Hardcode ID tujuan di sini
		targetID := "user_1" 
		
		token, exists := userTokens[targetID]
		if !exists {
			c.JSON(404, gin.H{"error": fmt.Sprintf("User ID '%s' belum terdaftar/connect", targetID)})
			return
		}

		msg := &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: "ALERT!",
				Body:  "Hello from GO!",
			},
		}
		fcmClient.Send(context.Background(), msg)
		c.JSON(200, gin.H{"message": fmt.Sprintf("Pesan terkirim ke User: %s", targetID)})
	})

	// API BROADCAST (Kirim ke Topic)
	r.POST("/broadcast", func(c *gin.Context) {
		var req BroadcastReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Pesan dikirim ke "Topic", BUKAN ke "Token"
		message := &messaging.Message{
			Topic: req.Topic, 
			Notification: &messaging.Notification{
				Title: req.Title,
				Body:  req.Body,
			},
			Data: map[string]string{
				"screen": "auction", // Misal broadcast news, arahkan ke screen auction
			},
		}

		// Kirim
		responseID, err := fcmClient.Send(context.Background(), message)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		fmt.Printf("📢 Broadcast ke topic [%s] sukses! ID: %s\n", req.Topic, responseID)
		c.JSON(200, gin.H{"status": "broadcast_sent", "topic": req.Topic})
	})

	r.Run("0.0.0.0:8081")
}