// Will soon migrate from mysql to sqlite

// Future goals:
// !fetch @user Ls
// random flavor text
// bot intro message

//TO RUN BOT:

//	1. Delete old database if you want new data
//		-open terminal
//		-set root path variable: export PATH=${PATH}:/usr/local/mysql/bin

//		-default username is root
//		-default password is rootpass
//		-***change constants below if yours are different***

//		-mysql -u <username> -p
//		-enter password
//		-DROP DATABASE countDB;
//
//	2. Run bot from project folder
//		-open new terminal (don't close the mysql one)
//		-cd into project directory
//		-export token: export BOT_TOKEN=<BOT TOKEN GOES HERE>
//		-start bot: go run main.go -t $BOT_TOKEN

//TO CLOSE BOT:
//	CTRL-C in the terminal running the bot (in the project folder)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

// variables used for command line parameters
var Token string
var DB *sql.DB

const (
	//for mysql database
	username = "root"
	password = "rootpass"
	hostname = "localhost"
	dbname   = "countDB"

	//bot commands
	commandL       = "!L"
	commandW       = "!W"
	commandFetchLs = "!FetchLs"
	commandFetchWs = "!FetchWs"

	commandCommands = "!Commands"

	//bot messages
	LMessage       = " has taken another L."
	WMessage       = " has secured another W."
	firstLMessage  = " has taken their first L."
	firstWMessage  = " has secured their first W."
	botsDontTakeLs = "Foolish mortal. I don't take L's; I give them."
	botGetsAW      = "Thanks for the W! :) I work hard to keep track."
	commandsList   = "Hi! I'm the L Tracker!\n\nReply !L to a user to give them an L\nReply !W to a user to give them a W\nSay !Commands to see a list of commands"
)

type Ltracker struct {
	userID   string
	username string
	Ls       int
	Ws       int
}

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func dbConnection() (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn(""))
	if err != nil {
		log.Printf("Error %s when opening database\n", err)
		return nil, err
	}

	//create database
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	res, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname)
	if err != nil {
		log.Printf("Error %s when creating database\n", err)
	}
	no, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when fetching rows affected during database update", err)
		return nil, err
	}
	log.Printf("%d rows affected by database update", no)

	//close unnamed database and connect to the one we just created
	db.Close()
	db, err = sql.Open("mysql", dsn(dbname))
	if err != nil {
		log.Printf("Error %s when opening database", err)
		return nil, err
	}

	//database config options
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(time.Hour * 24)

	//verify database connection
	ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	err = db.PingContext(ctx)
	if err != nil {
		log.Printf("Errors %s pinging database", err)
		return nil, err
	}
	log.Printf("Connected to %s successfully\n", dbname)
	return db, nil
}

func dsn(dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, hostname, dbName)
}

func createLTrackerTable(db *sql.DB) error {
	query := "CREATE TABLE IF NOT EXISTS lTracker(userID varchar(50) primary key, username varchar(50), Ls int, Ws int, created_at datetime default CURRENT_TIMESTAMP, updated_at datetime default CURRENT_TIMESTAMP)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when creating LTrackerTable", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when fetching rows affected during table creation", err)
		return err
	}
	log.Printf("%d rows affected when creating table", rows)
	return nil
}

func insertRow(db *sql.DB, t Ltracker) error {
	query := "INSERT IGNORE INTO Ltracker(userID, username, Ls, Ws) VALUES (?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL insert statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, t.userID, t.username, t.Ls, t.Ws)
	if err != nil {
		log.Printf("Error %s when inserting row into LTrackerTable", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when fetching rows affected while inserting row", err)
		return err
	}
	log.Printf("%d entries created", rows)
	return nil
}

func selectLs(db *sql.DB, userID string) int {
	query := "SELECT Ls FROM lTracker WHERE userID = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing selectLs statement", err)
		return -1
	}
	defer stmt.Close()
	var Ls int
	row := stmt.QueryRowContext(ctx, userID)
	if err := row.Scan(&Ls); err != nil {
		return -1
	}
	return Ls
}

func updateLs(db *sql.DB, userID string) int {
	var Ls int = selectLs(db, userID)
	if Ls == -1 {
		Ls = 1
	}
	Ls += 1
	query := "UPDATE lTracker SET Ls = ? WHERE userID = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing updateLs statement", err)
		return -1
	}
	defer stmt.Close()
	var LsNormalized int = Ls - 1
	LString := strconv.Itoa(LsNormalized)
	log.Println("Updating userID:" + userID + "'s Ls to " + LString)
	row := stmt.QueryRowContext(ctx, Ls, userID)
	if err := row.Scan(&Ls); err != nil {
		return -1
	}
	return Ls
}

func selectWs(db *sql.DB, userID string) int {
	query := "SELECT Ws FROM lTracker WHERE userID = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing selectWs statement", err)
		return -1
	}
	defer stmt.Close()
	var Ws int
	row := stmt.QueryRowContext(ctx, userID)
	if err := row.Scan(&Ws); err != nil {
		return -1
	}
	return Ws
}

func updateWs(db *sql.DB, userID string) int {
	var Ws int = selectWs(db, userID)
	if Ws == -1 {
		Ws = 1
	}
	Ws += 1
	query := "UPDATE lTracker SET Ws = ? WHERE userID = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing updateWs statement", err)
		return -1
	}
	defer stmt.Close()
	var WsNormalized int = Ws - 1
	WString := strconv.Itoa(WsNormalized)
	log.Println("Updating userID:" + userID + "'s Ws to " + WString)
	row := stmt.QueryRowContext(ctx, Ws, userID)
	if err := row.Scan(&Ws); err != nil {
		return -1
	}
	return Ws
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	//for Ls:

	var messageContent = m.Content
	var messageType = m.Type

	if strings.EqualFold(messageContent, commandL) && messageType == discordgo.MessageTypeReply {
		//connect to countDB
		db, err := dbConnection()
		if err != nil {
			log.Printf("Error %s when getting database connection", err)
			return
		}
		defer db.Close()
		//plug user info into variables
		referencedAuthorUsername := m.ReferencedMessage.Author.Username
		referencedAuthorID := m.ReferencedMessage.Author.ID
		var WsForFirstL int = selectWs(db, referencedAuthorID)
		firstL := Ltracker{
			userID:   referencedAuthorID,
			username: referencedAuthorUsername,
			Ls:       1,
			Ws:       WsForFirstL,
		}
		//insert userid, username and first L into LTrackerTable if they are not already in it
		err1 := insertRow(db, firstL)
		if err1 != nil {
			log.Printf("Insert Ls failed with error %s", err)
			return
		}
		//SELECT Ls FROM countDB WHERE userID=referencedAuthorID
		var Ls int = selectLs(db, referencedAuthorID)
		if referencedAuthorUsername != "" {
			//bots dont take Ls
			if referencedAuthorID == s.State.User.ID {
				s.ChannelMessageSend(m.ChannelID, botsDontTakeLs)
			} else {
				if Ls < 1 { //if Ls is initialized but has no data, this is the first L
					Ls = 1
				}
				count := strconv.Itoa(Ls)
				//first L:
				if count == "1" {
					updateLs(db, referencedAuthorID)
					s.ChannelMessageSend(m.ChannelID, referencedAuthorUsername+firstLMessage)
				} else
				//subsequent Ls:
				{
					updateLs(db, referencedAuthorID)
					LString := strconv.Itoa(Ls)
					s.ChannelMessageSend(m.ChannelID, referencedAuthorUsername+LMessage+" That's "+LString+" Ls now.")
				}
			}
		} else {
			return
		}
	}
	//for Ws
	if strings.EqualFold(messageContent, commandW) && messageType == discordgo.MessageTypeReply {
		//connect to countDB
		db, err := dbConnection()
		if err != nil {
			log.Printf("Error %s when getting database connection", err)
			return
		}
		defer db.Close()
		//plug user info into variables
		referencedAuthorUsername := m.ReferencedMessage.Author.Username
		referencedAuthorID := m.ReferencedMessage.Author.ID
		var LsForFirstW int = selectLs(db, referencedAuthorID)
		firstW := Ltracker{
			userID:   referencedAuthorID,
			username: referencedAuthorUsername,
			Ls:       LsForFirstW,
			Ws:       1,
		}
		//insert userid, username and first W into Ltracker if they are not already in it
		err1 := insertRow(db, firstW)
		if err1 != nil {
			log.Printf("Insert Ws failed with error %s", err)
			return
		}
		//SELECT Ws FROM countDB WHERE userID=referencedAuthorID
		var Ws int = selectWs(db, referencedAuthorID)
		if referencedAuthorUsername != "" {
			//bots do take Ws
			if referencedAuthorID == s.State.User.ID {
				s.ChannelMessageSend(m.ChannelID, botGetsAW)
			} else {
				if Ws < 1 { //if Ws is initialized but has no data, this is the first W
					Ws = 1
				}
				count := strconv.Itoa(Ws)
				//first W:
				if count == "1" {
					updateWs(db, referencedAuthorID)
					s.ChannelMessageSend(m.ChannelID, referencedAuthorUsername+firstWMessage)
				} else
				//subsequent Ws:
				{
					updateWs(db, referencedAuthorID)
					WString := strconv.Itoa(Ws)
					s.ChannelMessageSend(m.ChannelID, referencedAuthorUsername+WMessage+" That's "+WString+" Ws now.")
				}
			}
		} else {
			return
		}
	}

	if strings.EqualFold(messageContent, commandCommands) {
		s.ChannelMessageSend(m.ChannelID, commandsList)
	}
}

func main() {
	//Create a new Discord session using the provided bot token
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session, ", err)
		return
	}
	dg.Identify.Intents = discordgo.IntentsGuildMessages
	//Open a websocket connection to Discord and begin listening
	err = dg.Open()
	if err != nil {
		fmt.Println("Error connecting to Discord session", err)
		return
	}
	//Connect to countDB database
	db, err := dbConnection()
	if err != nil {
		log.Printf("Error %s when getting database connection", err)
		return
	}
	defer db.Close()
	log.Printf("Successfully connected to database")
	//Create LTrackerTable
	err = createLTrackerTable(db)
	if err != nil {
		log.Printf("Create LTrackerTable failed with error %s", err)
		return
	}
	//Register the messageCreate func as a callback for MessageCreate events
	dg.AddHandler(messageCreate)
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	//Close the Discord session.
	dg.Close()
}
