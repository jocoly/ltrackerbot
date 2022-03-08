//build
//export BOT_TOKEN=<TOKEN GOES HERE!!!>
//go run main.go -t $BOT_TOKEN
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token string
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

const LMessage = " has taken another L"
const WMessage = " has secured another W"

// var userID = -1
// var lCount = -1
// var wCount = -1

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	// dg.AddHandler(LWCount)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!L" && m.Type == discordgo.MessageTypeReply {
		referencedAuthor := m.ReferencedMessage.Author.Username
		if referencedAuthor != "" {
			s.ChannelMessageSend(m.ChannelID, referencedAuthor+LMessage)
		} else {
			return
		}
	}

	if m.Content == "!W" && m.Type == discordgo.MessageTypeReply {
		referencedAuthor := m.ReferencedMessage.Author.Username
		if referencedAuthor != "" {
			s.ChannelMessageSend(m.ChannelID, referencedAuthor+WMessage)
		} else {
			return
		}
	}
}

func LWCount(s *discordgo.Session) {
	//guildID := s.Guild

	//slice1 = guild members by userID
	//slice2 = LCount values where index position corresponds to a guild member
	//slice3 = WCount values where index position corresponds to a guild member
	//initializes with members and 0s for counts when the bot is first booted up
	//appends new guild member + 0s when a new user joins
	//returns the 3 slices
}

//func fetchLCount(userID, slice1, slice2)
//finds userID in slice1 and stores its index position
//returns slice2 value at that index position

//func fetchWCount(userID, slice1, slice3)
//finds userID in slice1 and stores its index position
//returns slice3 value at that index position

//func firstLCheck(userID, slice1, slice2)
//runs when !L is replied to a message
//if fetchLCount == 0, true, trigger flavor text
//else false, trigger standard text

//func firstWCheck(userID, slice1, slice3)
//runs when !L is replied to a message
//if fetchWCount == 0, true, trigger flavor text
//else false, trigger standard text
