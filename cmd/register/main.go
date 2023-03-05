package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

func main() {
	GuildID := os.Getenv("GUILD_ID")
	Token := os.Getenv("DISCORD_BOT_TOKEN")
	appID := flag.String("app", "", "app-id: 100000")
	flag.Parse()

	if *appID == "" {
		log.Fatal("app id is required")
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	err := register(context.Background(), *appID, GuildID, Token)
	log.Println(err)

}

func register(ctx context.Context, appID, guildId, token string) error {
	url := fmt.Sprintf("https://discord.com/api/v10/applications/%s/guilds/%s/commands", appID, guildId)
	cmd := discordgo.ApplicationCommand{
		Name:        "chat",
		Type:        discordgo.ChatApplicationCommand,
		Description: "chat-gpt",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "prompt",
				Description: "prompt",
				Type:        3,
			},
		},
	}
	b, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", fmt.Sprintf("Bot %s", token))

	res, err := http.DefaultClient.Do(r)

	if err != nil {
		return err
	}
	defer res.Body.Close()

	a, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	log.Println(string(a))
	if res.StatusCode != http.StatusOK {
		return errors.New(res.Status)
	}
	return nil
}
