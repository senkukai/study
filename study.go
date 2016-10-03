package main

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type Student struct {
	User      string
	Name      string
	FirstName string
	Gender    string
	Password  string
}
type Group struct {
	Lun [5]*Student
	Mar [5]*Student
	Mer [5]*Student
	Jeu [5]*Student
}
type ClassRoom struct {
	Id     string
	Name   string
	Cap    int
	Gender string
}
type Booking struct {
	Day       string
	Student   string
	ClassRoom string
	Group     []string
}
type Event struct {
	Type      string
	Date      time.Time
	Day       string
	Student   string
	ClassRoom string
	Group     []string
}
type Context struct {
	Student   Student
	ClassRoom map[string]ClassRoom
}

var eventsFile = "data/events.log"
var students = map[string]Student{
	"albert.jean":  Student{"albert.jean", "Jean", "Albert", "G", "jeanjean"},
	"alice.cooper": Student{"alice.cooper", "Cooper", "Alice", "G", "caglisse"}}
var classRooms = map[string]ClassRoom{
	"210": ClassRoom{"210", "Etude individuelle Filles", 35, "F"},
	"216": ClassRoom{"216", "Etude individuelle Garcons", 35, "G"},
	"219": ClassRoom{"219", "Etude en groupe 1", 35, ""},
	"207": ClassRoom{"207", "Etude en groupe 2", 35, ""},
	"CDI": ClassRoom{"CDI", "CDI", 0, ""}}
var events = []Event{}
var bookings = []Booking{}

var tmplDir = "tmpl/"

func rootHandler(w http.ResponseWriter, r *http.Request, c *Context) {
	renderTemplate(w, "view", c)
}
func submitHandler(w http.ResponseWriter, r *http.Request, c *Context) {
	r.ParseForm()
	fmt.Print(r.Form)
	events = append(
		events,
		Event{
			"book",
			time.Now(),
			"Lundi",
			c.Student.User,
			r.Form["Lundi"][0],
			[]string{}})
	events = append(events, Event{"book", time.Now(), "Mardi", c.Student.User, r.Form["Mardi"][0], []string{}})
	events = append(events, Event{"book", time.Now(), "Mercredi", c.Student.User, r.Form["Mercredi"][0], []string{}})
	events = append(events, Event{"book", time.Now(), "Jeudi", c.Student.User, r.Form["Jeudi"][0], []string{}})
	http.Redirect(w, r, "/", http.StatusFound)
}

var templates = template.Must(template.ParseFiles(tmplDir + "view.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, c *Context) {
	err := templates.ExecuteTemplate(w, tmpl+".html", c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, *Context)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		user, pass, _ := r.BasicAuth()
		student, ok := students[user]
		if !ok || student.Password != pass {
			http.Error(w, "Unauthorized.", 401)
			return
		}

		fmt.Print(student)
		fmt.Print("\n")
		c := &Context{student, classRooms}
		fn(w, r, c)
	}
}

func remaining(s string, j string) int {
	rem := classRooms[s].Cap
	if len(bookings) == 0 {
		return rem
	}
	for _, b := range bookings {
		if b.ClassRoom == s && b.Day == j {
			rem -= 1
		}
	}
	return rem
}
func (e Event) book() error {
	exist := false
	if remaining(e.ClassRoom, e.Day) > 0 {
		for i, b := range bookings {
			if b.Student == e.Student && b.Day == e.Day {
				bookings[i].ClassRoom = e.ClassRoom
				exist = true
			}
		}
		if !exist {
			bookings = append(bookings, Booking{e.Day, e.Student, e.ClassRoom, []string{}})
		}
		return nil
	} else {
		return errors.New("Il n'y a plus de places dans la salle suivante : " + e.ClassRoom)
	}
}
func (e Event) log() {
	f, err := os.OpenFile(eventsFile, os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(e.String())
}
func (e Event) String() string {
	return fmt.Sprintf("%v_%v_%v_%v_%v_%v\n", e.Type, e.Date, e.Day, e.Student, e.ClassRoom, e.Group)
}
func eventProcessor(c chan Event) error {
	for {
		e := <-c
		events = append(events, e)
		e.log()
		err := e.book()
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}
func resetEvents() {
	os.Remove(eventsFile)
	f, err := os.OpenFile(eventsFile, os.O_CREATE, 0700)
	if err != nil {
		panic(err)
	}
	f.Close()
	for _, cr := range classRooms {
		for i, s := range students {
			if cr.Gender == s.Gender {
				e := Event{"book", time.Now(), "Lundi", i, cr.Id, []string{}}
				e.log()
				events = append(events, e)
				e = Event{"book", time.Now(), "Mardi", i, cr.Id, []string{}}
				e.log()
				events = append(events, e)
				e = Event{"book", time.Now(), "Mercredi", i, cr.Id, []string{}}
				e.log()
				events = append(events, e)
				e = Event{"book", time.Now(), "Jeudi", i, cr.Id, []string{}}
				e.log()
				events = append(events, e)
			}
		}
	}
}
func loadEvents() {
	f, err := os.Open(eventsFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		splitEvent := strings.Split(line, "_")
		time, _ := time.Parse("2006-01-02 15:04:00.000000000 +0200 CEST", splitEvent[1])
		e := Event{
			splitEvent[0],
			time,
			splitEvent[2],
			splitEvent[3],
			splitEvent[4],
			[]string{}}
		events = append(events, e)
		e.book()
	}
}

func eventPopulate(c chan Event) {
	days := []string{"Lundi", "Mardi", "Mercredi", "Jeudi"}
	idUsers := []string{}
	for i := range students {
		idUsers = append(idUsers, i)
	}
	idClassRooms := []string{}
	for i := range classRooms {
		idClassRooms = append(idClassRooms, i)
	}
	rand.Seed(10)
	routine := 100
	for routine > 0 {
		go func(c chan Event, loop int, days []string, idUsers []string, idClassRooms []string) {
			for loop > 0 {
				c <- Event{
					"book",
					time.Now(),
					days[rand.Intn(len(days))],
					idUsers[rand.Intn(len(idUsers))],
					idClassRooms[rand.Intn(len(idClassRooms))],
					[]string{}}
				loop -= 1
			}
		}(c, 100, days, idUsers, idClassRooms)
		routine -= 1
	}
}

func main() {
	//resetEvents()
	loadEvents()

	c := make(chan Event)
	go eventProcessor(c)

	//eventPopulate(c)

	//time.Sleep(1 * time.Second)
	fmt.Printf("len events:%v\n", len(events))
	for _, v := range bookings {
		fmt.Println(v)
	}
	//http.HandleFunc("/submit", makeHandler(submitHandler))
	//http.HandleFunc("/", makeHandler(rootHandler))
	//http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil)
}
