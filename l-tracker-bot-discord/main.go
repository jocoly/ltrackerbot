//build
//export BOT_TOKEN=<TOKEN GOES HERE!!!>
//go run main.go -t $BOT_TOKEN
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

// Variables used for command line parameters
var Token string
var DB *sql.DB

const (
	username       = "root"
	password       = "rootpass"
	hostname       = "localhost"
	dbname         = "countDB"
	LMessage       = " has taken another L."
	WMessage       = " has secured another W."
	firstLMessage  = " has taken their first L."
	firstWMessage  = " has secured their first W."
	botsDontTakeLs = "Foolish mortal. I don't take L's; I give them."
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
	//defer db.Close()

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
	//defer db.Close()

	//db config options
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(time.Hour * 24)

	//verify db connection
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

	if m.Content == "!L" && m.Type == discordgo.MessageTypeReply {
		db, err := dbConnection()
		if err != nil {
			log.Printf("Error %s when getting database connection", err)
			return
		}
		defer db.Close()

		referencedAuthorUsername := m.ReferencedMessage.Author.Username
		referencedAuthorID := m.ReferencedMessage.Author.ID
		var WsForFirstL int = selectWs(db, referencedAuthorID)
		firstL := Ltracker{
			userID:   referencedAuthorID,
			username: referencedAuthorUsername,
			Ls:       1,
			Ws:       WsForFirstL,
		}
		//insert userid, username and first L into Ltracker if they are not already in it
		err1 := insertRow(db, firstL)
		if err1 != nil {
			log.Printf("Insert Ls failed with error %s", err)
			return
		}

		var Ls int = selectLs(db, referencedAuthorID)
		//SELECT Ls FROM countDB WHERE userID=referencedAuthorID
		if referencedAuthorUsername != "" {
			//bots dont take Ls
			if referencedAuthorID == s.State.User.ID {
				s.ChannelMessageSend(m.ChannelID, botsDontTakeLs)
			} else {
				if Ls < 1 {
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

	if m.Content == "!W" && m.Type == discordgo.MessageTypeReply {
		db, err := dbConnection()
		if err != nil {
			log.Printf("Error %s when getting database connection", err)
			return
		}
		defer db.Close()

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

		var Ws int = selectWs(db, referencedAuthorID)
		//SELECT Ws FROM countDB WHERE userID=referencedAuthorID
		if referencedAuthorUsername != "" {
			//bots do take Ws
			if referencedAuthorID == s.State.User.ID {
				s.ChannelMessageSend(m.ChannelID, "Thanks for the W! :) I work hard to keep track.")
			} else {
				if Ws < 1 {
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
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session, ", err)
		return
	}
	dg.Identify.Intents = discordgo.IntentsGuildMessages
	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error connecting to Discord session", err)
		return
	}

	db, err := dbConnection()
	if err != nil {
		log.Printf("Error %s when getting database connection", err)
		return
	}
	defer db.Close()
	log.Printf("Successfully connected to database")
	err = createLTrackerTable(db)
	if err != nil {
		log.Printf("Create LTrackerTable failed with error %s", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	// Cleanly close down the Discord session.
	dg.Close()
}
