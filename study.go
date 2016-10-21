package main

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Student struct {
	User      string
	Name      string
	FirstName string
	Class     string
	Gender    string
	Password  string
}
type Group struct {
	Lun [][]string
	Mar [][]string
	Mer [][]string
	Jeu [][]string
}
type ClassRoom struct {
	Id     string
	Name   string
	Cap    int
	Gender string
	Group  bool
}
type Booking struct {
	Day       string
	Student   string
	ClassRoom string
	Group     []string
	Work      map[string][]string
}
type Event struct {
	Type    string
	Date    time.Time
	Day     string
	Student string
	Value   string
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
	IdxSubjects   *[]string
	ClassRooms    *map[string]ClassRoom
	RemainSeats   *[4][5]int
	//RemainSeats *map[string]*[5]int
	Occupancy [4]string
	Work      map[string]map[string][]string
	Errors    []error
	Students  [][]string
	Group     map[string][][]string
	Values    map[string][]string
}

var eventsFile = "data/events.log"
var studentsFile = "data/students.log"
var students = map[string]Student{}

/*
var students = map[string]Student{
	"ingalls.albert": Student{"ingalls.albert", "Ingalls", "Albert", "TSTG", "G", ""},
	"plotte.camille": Student{"plotte.camille", "Plotte", "Camille", "TSTG", "F", ""},
	"jambon.chris":   Student{"jambon.chris", "Jambon", "Chris", "TSTG", "G", ""},
	"cooper.alice":   Student{"cooper.alice", "Cooper", "Alice", "TSTG", "F", ""}}
*/
var classRooms = map[string]ClassRoom{
	"210": ClassRoom{"210", "210 Etude individuelle", -1, "F", false},
	"216": ClassRoom{"216", "216 Etude individuelle", -1, "G", false},
	"219": ClassRoom{"219", "219 Etude en groupe", 2, "", true},
	"207": ClassRoom{"207", "207 Etude en groupe", 2, "", true},
	"CDI": ClassRoom{"CDI", "CDI Etude individuelle", 2, "", false}}
var idxDays = []string{"Lundi", "Mardi", "Mercredi", "Jeudi"}
var idxClassRooms = []string{"210", "216", "219", "207", "CDI"}
var idxSubjects = []string{"Français", "Mathématiques", "Histoire", "Anglais", "Masturbation"}

/*
var RemainSeats = map[string]*[5]int{
	"Lundi":    &[5]int{},
	"Mardi":    &[5]int{},
	"Mercredi": &[5]int{},
	"Jeudi":    &[5]int{}}
*/

var RemainSeats = [4][5]int{
	[5]int{},
	[5]int{},
	[5]int{},
	[5]int{}}
var events = []Event{}
var bookings = []Booking{}

var c = make(chan *EventCon)

var tmplDir = "tmpl/"

func hash(s string) string {
	h := fnv.New32a()
	h.Write([]byte("study/" + s))
	return fmt.Sprint(h.Sum32())
}

var templates = template.Must(template.ParseFiles(tmplDir+"picksubject.html", tmplDir+"pickgroup.html", tmplDir+"view.html", tmplDir+"login.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, c *TmplCon) {
	err := templates.ExecuteTemplate(w, tmpl+".html", c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func studentList(student string) [][]string {
	list := [][]string{}
	index := []string{}
	fLetter := ""
	for _, s := range students {
		index = append(index, s.User)
	}
	sort.Sort(sort.StringSlice(index))
	for _, i := range index {
		if fLetter != string(i[0]) {
			fLetter = string(i[0])
			index = append(index, fLetter)
		}
	}
	sort.Sort(sort.StringSlice(index))
	for _, s := range index {
		if s != student {
			if len(s) == 1 {
				list = append(list, []string{"index", strings.ToUpper(s), "", ""})
			} else {
				list = append(list, []string{students[s].User, students[s].Name, students[s].FirstName, students[s].Class})
			}
		}
	}
	return list
}
func groupList(s string) map[string][][]string {
	list := map[string][][]string{}
	for _, b := range bookings {
		if b.Student == s {
			for _, g := range b.Group {
				if roomByDay(s, b.Day) == roomByDay(g, b.Day) {
					list[b.Day] = append(list[b.Day], []string{g, "present"})
				} else {
					list[b.Day] = append(list[b.Day], []string{g, "absent"})
				}
			}
		}
	}
	return list
}
func workList(s string) map[string]map[string][]string {
	list := map[string]map[string][]string{}
	for i, b := range bookings {
		if b.Student == s {
			list[b.Day] = bookings[i].Work
		}
	}
	return list
}
func contains(slice []string, elem string) bool {
	for _, s := range slice {
		if s == elem {
			return true
		}
	}
	return false
}
func groupChange(s string, g []string, d string) bool {
	for _, b := range bookings {
		if b.Student == s && b.Day == d {
			for i := range b.Group {
				if b.Group[i] != g[i] {
					return true
				}
			}
		}
	}

	return false
}
func roomChange(s string, cr string, d string) bool {
	var idx int
	for i, day := range idxDays {
		if day == d {
			idx = i
		}
	}
	return occupancy(s)[idx] != cr
}
func roomByDay(s string, d string) string {
	for _, b := range bookings {
		if b.Student == s && b.Day == d {
			return b.ClassRoom
		}
	}
	return ""
}
func occupancy(s string) [4]string {
	occupancy := [4]string{}
	for _, b := range bookings {
		if b.Student == s {
			for j, d := range idxDays {
				if d == b.Day {
					occupancy[j] = b.ClassRoom
				}
			}
		}
	}
	return occupancy
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
func remainUpdate() {
	for i, d := range idxDays {
		for j, cr := range idxClassRooms {
			r := remaining(cr, d)
			RemainSeats[i][j] = r
		}
	}
}
func loadStudents() {
	f, err := os.Open(studentsFile)
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
		split := strings.Split(line, "_")
		s := Student{
			split[0],
			split[1],
			split[2],
			split[3],
			split[4],
			split[5]}
		students[split[0]] = s
	}
}

func genStudents() {
	names := map[string][]string{}
	files := []string{"data/names", "data/male_fnames", "data/female_fnames"}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			panic(err)
		}
		r := bufio.NewReader(f)
		for {
			line, err := r.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				panic(err)
			}
			line = strings.Trim(line, "\n")
			line = strings.TrimSpace(line)
			names[file] = append(names[file], line)
		}
		f.Close()
	}

	os.Remove(studentsFile)
	f, err := os.OpenFile(studentsFile, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	classes := []string{"2A", "2B", "2C", "2D", "2E", "1S1", "1S2", "1L", "1ES", "1STG", "TS1", "TS2", "TL", "TES", "TSTG", "2COM", "2SEC", "2ALIM", "1COM", "1SEC", "1ALIM", "TCOM", "TSEC", "TALIM"}
	genders := []string{"M", "F"}
	rand.Seed(time.Now().UTC().UnixNano())
	gen := 200
	users := []string{}
	for gen > 0 {
		name := names["data/names"][rand.Intn(len(names["data/names"]))]
		var firstname string
		gender := genders[rand.Intn(len(genders))]
		if gender == "M" {
			firstname = names["data/male_fnames"][rand.Intn(len(names["data/male_fnames"]))]
		} else if gender == "F" {
			firstname = names["data/female_fnames"][rand.Intn(len(names["data/female_fnames"]))]
		}
		user := strings.ToLower(name + "." + firstname)
		for contains(users, user) {
			user = user + "_"
		}
		users = append(users, user)
		class := classes[rand.Intn(len(classes))]
		f.WriteString(user + "_" + name + "_" + firstname + "_" + class + "_" + gender + "__\n")
		gen -= 1
	}
	f.Close()
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
						idClassRooms[rand.Intn(len(idClassRooms))]},
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

func main() {
	//genStudents()
	loadStudents()
	//resetEvents()
	loadEvents()
	//fmt.Println(bookings)

	go eventProcessor()
	http.Handle("/bs/", http.StripPrefix("/bs/", http.FileServer(http.Dir("bs/"))))
	http.HandleFunc("/login/", loginHandler)
	http.HandleFunc("/submit", makeHandler(submitHandler))
	http.HandleFunc("/", makeHandler(rootHandler))
	http.ListenAndServe(":8080", nil)
}
