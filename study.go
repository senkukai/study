package main

import (
	"bufio"
	"errors"
	"fmt"
	"hash/fnv"
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
type EventCon struct {
	Event Event
	Error chan error
}
type TmplCon struct {
	Student       Student
	IdxDays       *[]string
	IdxClassRooms *[]string
	ClassRooms    *map[string]ClassRoom
	RemainSeats   *[4][5]int
	Occupancy     *[4]string
	Errors        []error
}

var eventsFile = "data/events.log"
var students = map[string]Student{
	"albert.jean":  Student{"albert.jean", "Jean", "Albert", "G", "jeanjean"},
	"alice.cooper": Student{"alice.cooper", "Cooper", "Alice", "F", "caglisse"}}
var classRooms = map[string]ClassRoom{
	"210": ClassRoom{"210", "Etude individuelle Filles", 35, "F"},
	"216": ClassRoom{"216", "Etude individuelle Garcons", 35, "G"},
	"219": ClassRoom{"219", "Etude en groupe 1", 1, ""},
	"207": ClassRoom{"207", "Etude en groupe 2", 1, ""},
	"CDI": ClassRoom{"CDI", "CDI", 1, ""}}
var idxDays = []string{"Lundi", "Mardi", "Mercredi", "Jeudi"}
var idxClassRooms = []string{"210", "216", "219", "207", "CDI"}
var RemainSeats = [4][5]int{
	[5]int{},
	[5]int{},
	[5]int{},
	[5]int{}}
var Occupancy [4]string
var events = []Event{}
var bookings = []Booking{}

var c = make(chan *EventCon)

var tmplDir = "tmpl/"

func rootHandler(w http.ResponseWriter, r *http.Request, con *TmplCon) {
	renderTemplate(w, "view", con)
}
func hash(s string) string {
	h := fnv.New32a()
	h.Write([]byte("study/" + s))
	return fmt.Sprint(h.Sum32())
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form) == 0 {
		cookie := http.Cookie{Name: "session", Value: "deleted", HttpOnly: false, Path: "/"}
		http.SetCookie(w, &cookie)
		renderTemplate(w, "login", nil)
		return
	}
	user := r.Form["user"][0]
	pass := r.Form["password"][0]
	if students[user].Password == pass {
		cookie := http.Cookie{Name: "session", Value: user + "/" + hash(user), HttpOnly: false, Path: "/"}
		http.SetCookie(w, &cookie)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
func submitHandler(w http.ResponseWriter, r *http.Request, con *TmplCon) {
	r.ParseForm()
	fmt.Print(r.Form)
	for i, d := range idxDays {
		// only create an event if student has changed classroom
		if con.Occupancy[i] != r.Form[d][0] {
			comm := &EventCon{
				Event{
					"book",
					time.Now(),
					d,
					con.Student.User,
					r.Form[d][0],
					[]string{}},
				make(chan error)}
			c <- comm
			error := <-comm.Error
			if error != nil {
				con.Errors = append(con.Errors, error)
			}
		}
	}
	remainUpdate()
	updateOccupancy(con.Student.User)
	renderTemplate(w, "view", con)
}

var templates = template.Must(template.ParseFiles(tmplDir+"view.html", tmplDir+"login.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, c *TmplCon) {
	err := templates.ExecuteTemplate(w, tmpl+".html", c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, *TmplCon)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Header)
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		userhash := strings.Split(cookie.Value, "/")
		if len(userhash) == 1 || userhash[1] != hash(userhash[0]) {
			http.Redirect(w, r, "/login", http.StatusFound)
		}

		fmt.Printf("cookie:%v\n", cookie)

		student, _ := students[userhash[0]]
		remainUpdate()
		updateOccupancy(student.User)
		con := &TmplCon{student, &idxDays, &idxClassRooms, &classRooms, &RemainSeats, &Occupancy, []error{}}
		fn(w, r, con)
	}
}
func updateOccupancy(s string) {
	for _, b := range bookings {
		if b.Student == s {
			for j, d := range idxDays {
				if d == b.Day {
					Occupancy[j] = b.ClassRoom
				}
			}
		}
	}
}
func remaining(cr string, d string) int {
	rem := classRooms[cr].Cap
	if len(bookings) == 0 {
		return rem
	}
	for _, b := range bookings {
		if b.ClassRoom == cr && b.Day == d {
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
		return errors.New("Il n'y a plus de places " + e.Day + " dans la salle suivante : " + e.ClassRoom)
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
func eventProcessor() {
	for e := range c {
		events = append(events, e.Event)
		e.Event.log()
		e.Error <- e.Event.book()
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

func eventPopulate(c chan *EventCon) {
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
	routine := 10000
	for routine > 0 {
		go func(c chan *EventCon, loop int, days []string, idUsers []string, idClassRooms []string) {
			for loop > 0 {
				comm := &EventCon{
					Event{
						"book",
						time.Now(),
						days[rand.Intn(len(days))],
						idUsers[rand.Intn(len(idUsers))],
						idClassRooms[rand.Intn(len(idClassRooms))],
						[]string{}},
					make(chan error)}
				c <- comm
				if <-comm.Error == nil {
				}
				loop -= 1
			}
		}(c, 100, days, idUsers, idClassRooms)
		routine -= 1
	}
}
func remainUpdate() {
	for i, d := range idxDays {
		for j, cr := range idxClassRooms {
			r := remaining(cr, d)
			RemainSeats[i][j] = r
		}
	}
}

func main() {
	//resetEvents()
	loadEvents()

	//c := make(chan *EventCon)
	go eventProcessor()

	//eventPopulate(c)

	//time.Sleep(1 * time.Second)
	fmt.Printf("len events:%v\n", len(events))
	for _, v := range bookings {
		fmt.Println(v)
	}

	http.HandleFunc("/login/", loginHandler)
	http.HandleFunc("/submit", makeHandler(submitHandler))
	http.HandleFunc("/", makeHandler(rootHandler))
	http.ListenAndServe(":8080", nil)
	//http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil)
}
