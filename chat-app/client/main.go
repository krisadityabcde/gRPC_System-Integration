package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"google.golang.org/grpc"
	pb "github.com/krisadityabcde/gRPC_System-Integration/chat-app/proto"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewChatServiceClient(conn)
	
	// Login
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	res, err := client.Login(context.Background(), &pb.LoginRequest{
		Username: username,
		Password: "password", // Implementasi autentikasi sebenarnya
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Logged in as %s\n", res.Token)

	// Stream pengguna online
	go func() {
		stream, _ := client.StreamUsers(context.Background(), &pb.Empty{})
		for {
			users, err := stream.Recv()
			if err == io.EOF {
				return
			}
			fmt.Printf("\nOnline users: %v\n> ", users.Users)
		}
	}()

	// Chat stream
	chatStream, _ := client.ChatStream(context.Background())
	
	// Input pesan
	go func() {
		for {
			fmt.Print("> ")
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			
			chatStream.Send(&pb.ChatMessage{
				From:    username,
				Message: text,
				To:      "all",
			})
		}
	}()

	// Terima pesan
	for {
		msg, err := chatStream.Recv()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\n[%s] %s\n> ", msg.From, msg.Message)
	}
}