package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/625tk/chottogpt/pkg/openai"
	"github.com/bwmarrin/discordgo"
)

func main() {
	// initialize
	token := os.Getenv("DISCORD_BOT_TOKEN")
	cli := openai.NewOpenaiClient(os.Getenv("OPENAI_API_KEY"))
	h, err := hex.DecodeString(os.Getenv("DISCORD_PUBLIC_KEY"))
	listen := flag.String("p", ":8081", "listen")
	flag.Parse()

	if err != nil {
		log.Fatal(err)
	}

	pubKey := ed25519.PublicKey(h)
	s, err := discordgo.New(fmt.Sprintf("Bot %s", token))

	// ping endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = fmt.Fprintln(w, "ok")
	})

	// discord interaction endpoint
	http.HandleFunc("/callback/d/interaction", func(w http.ResponseWriter, r *http.Request) {
		verified := discordgo.VerifyInteraction(r, pubKey)
		if !verified {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println("b", string(b))
			log.Println("invalid verify")
			w.WriteHeader(401)
			_, _ = w.Write([]byte("error"))
			return
		}

		var req discordgo.Interaction
		b, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("b", string(b))
		err = json.Unmarshal(b, &req)
		if err != nil {
			log.Println(err)
			return
		}

		switch req.Type {
		case discordgo.InteractionPing:
			resp, err := json.Marshal(discordgo.InteractionResponse{
				Type: 1,
			})
			if err != nil {
				log.Println(err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(resp)
			log.Println(string(resp))
		case discordgo.InteractionMessageComponent:
			log.Println("InteractionMessageComponent")
		case discordgo.InteractionApplicationCommand:
			log.Println("InteractionApplicationCommand")
			uid := ""
			input := ""
			if req.Member != nil && req.Member.User != nil {
				uid = req.Member.User.ID
				log.Println(req.Member.User.ID)
				log.Println(req.Member.User.Username)
			}
			if v, ok := req.Data.(discordgo.ApplicationCommandInteractionData); ok {
				if len(v.Options) == 1 {
					if v, ok := v.Options[0].Value.(string); ok {
						input = v
					}
				}
			}

			// use goroutine
			go func() {
				answer := "invalid request"
				if uid != "" && input != "" {
					answer = cli.Prompt(context.Background(), input, uid)
				}
				m, err := s.FollowupMessageCreate(&req, false, &discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "prompt",
							Description: input,
							Color:       0xff0000,
						},
					},
					Content: answer,
				})
				if err != nil {
					log.Println(err)
					return
				}
				log.Println(m)
			}()

			resp, err := json.Marshal(discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				log.Println(err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(resp)
			log.Println(string(resp))
		}
	})

	err = http.ListenAndServe(*listen, nil)
	log.Println(err)
}
