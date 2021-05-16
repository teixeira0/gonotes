package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	"log"
	"net/http"
	"database/sql"
    "github.com/astaxie/beedb"
    _ "github.com/go-sql-driver/mysql"
    "golang.org/x/net/websocket"
)

var orm beedb.Model
var connections []*websocket.Conn

type Note struct {
	Id int `beedb:"PK"`
	Title string
	Content string
}

type Page struct {
    Title string
    Body  []byte
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    
    title := r.URL.Path[len("/"):]
    p, _ := loadPage(title)
    fmt.Fprintf(w, "<h1 align=\"center\">%s</h1><div>%s</div>", p.Title, p.Body)
}

func refreshClient(ws *websocket.Conn) {
	var allNotes []Note
	var msg string    
	var err error

    orm.FindAll(&allNotes)

    if len(allNotes) > 0 {
	    for _, note := range allNotes {
	    	if len(msg) > 0 {
	    		msg = msg + "-;;-"
	    	}
	    	msg = msg + strconv.Itoa(note.Id) + "-,,-" + note.Title + "-,,-" + note.Content
	    }
    } else {
    	msg = "empty"
    }
    if err = websocket.Message.Send(ws, msg); err != nil {
        fmt.Println("Can't send")
    }
}

func refreshAllClients() {
	var allNotes []Note
	var msg string
	var err error

    orm.FindAll(&allNotes)

    if len(allNotes) > 0 {
	    for _, note := range allNotes {
	    	if len(msg) > 0 {
	    		msg = msg + "-;;-"
	    	}
	    	msg = msg + strconv.Itoa(note.Id) + "-,,-" + note.Title + "-,,-" + note.Content
	    }
    } else {
    	msg = "empty"
    }
    for _, ws := range connections {
    	if err = websocket.Message.Send(ws, msg); err != nil {
        	fmt.Println("Can't send")
    	}
    }
}

func noteSocketHandler(ws *websocket.Conn) {
	var err error
	for {
		var reply string

        if err = websocket.Message.Receive(ws, &reply); err != nil {
        	fmt.Println("Can't receive")
            break
        }
	
		fmt.Println("Message received:", reply)
		if (reply == "Refresh") {
			connections = append(connections, ws)
			refreshClient(ws)
		} else if strings.HasPrefix(reply, "--Erase:") {
			pieces := strings.Split(reply, ":")
			id, _ := strconv.Atoi(pieces[1])

			fmt.Println("Remove ID:", id)

			orm.SetTable("note").Where(id).DeleteRow()

			refreshAllClients()
		} else {
			pieces := strings.Split(reply, "-,,-")
			var newNote Note
			newNote.Title = pieces[0]
			newNote.Content = pieces[1]
			orm.Save(&newNote)
			fmt.Println(newNote)

			refreshAllClients()
		}
	}
}

func (p *Page) save() error {
    filename := p.Title + ".html"
    return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
    filename := title + ".html"
    body, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return &Page{Title: title, Body: body}, nil
}

func main() {
    port := os.Getenv("PORT")

    if port == "" {
        log.Fatal("$PORT must be set")
    }

	db, err := sql.Open("mysql", "bfac529c4e00a7:2b5adff2@tcp(us-cdbr-east-03.cleardb.com:3306)/heroku_94ecc8b16e3723c")
	if err != nil {
		panic(err)
	}

	orm = beedb.New(db)
	
    http.HandleFunc("/gonotes", homeHandler)
    http.Handle("/new", websocket.Handler(noteSocketHandler))
    log.Fatal(http.ListenAndServe(":" + port, nil))
}
