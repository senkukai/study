package main

import (
	"bufio"
	"bytes"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
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
type Admin struct {
	User     string
	Password string
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
type RestrictedTime struct {
	FromDay    int
	FromHour   int
	FromMinute int
	ToDay      int
	ToHour     int
	ToMinute   int
}
type ResetTime struct {
	Day    int
	Hour   int
	Minute int
}
type TmplCon struct {
	Student        Student
	IdxDays        *[]string
	IdxWeek        *[]string
	IdxDates       *[4]string
	IdxClassRooms  *[]string
	Subjects       *[]string
	ClassRooms     *map[string]ClassRoom
	Classes        *[]string
	RemainSeats    *[4][]int
	RestrictedTime *[]RestrictedTime
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
var adminsFile = "data/admins.log"
var subjectsFile = "data/subjects.log"
var classesFile = "data/classes.log"
var classRoomsFile = "data/classrooms.log"
var idxClassRoomsFile = "data/idxclassrooms.log"
var restrictedTimeFile = "data/restrictedtime.log"
var Files = []string{eventsFile,
	studentsFile,
	adminsFile,
	subjectsFile,
	classesFile,
	classRoomsFile,
	idxClassRoomsFile,
	restrictedTimeFile}
var students = map[string]Student{}
var admins = map[string]Admin{}
var bookingsEnabled = true

var classRooms = map[string]ClassRoom{}
var idxDays = []string{"Lundi", "Mardi", "Mercredi", "Jeudi"}
var idxWeek = []string{"Lundi", "Mardi", "Mercredi", "Jeudi", "Vendredi", "Samedi", "Dimanche"}
var idxDates = [4]string{}
var idxClassRooms = []string{}
var subjects = []string{}
var classes = []string{}
var restrictedHours = []RestrictedTime{}
var resetTime = ResetTime{5, 0, 0}

/*
var RemainSeats = map[string]*[5]int{
	"Lundi":    &[5]int{},
	"Mardi":    &[5]int{},
	"Mercredi": &[5]int{},
	"Jeudi":    &[5]int{}}
*/

var RemainSeats = [4][]int{
	[]int{},
	[]int{},
	[]int{},
	[]int{}}
var events = []Event{}
var bookings = []Booking{}

var c = make(chan *EventCon)

var tmplDir = "tmpl/"
var templates = template.Must(template.ParseFiles(
	tmplDir+"admin.html",
	tmplDir+"admin_students.html",
	tmplDir+"admin_subjects.html",
	tmplDir+"admin_classes.html",
	tmplDir+"admin_classrooms.html",
	tmplDir+"admin_restrictedtime.html",
	tmplDir+"admin_lists.html",
	tmplDir+"picksubject.html",
	tmplDir+"pickgroup.html",
	tmplDir+"view.html",
	tmplDir+"login.html"))

func hash(s string) string {
	h := fnv.New32a()
	h.Write([]byte("study/" + students[s].Password))
	return fmt.Sprint(h.Sum32())
}

func renderTemplate(w http.ResponseWriter, tmpl string, c *TmplCon) {
	err := templates.ExecuteTemplate(w, tmpl+".html", c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
func (rt RestrictedTime) String() string {
	fromDay := strconv.Itoa(rt.FromDay)
	fromHour := strconv.Itoa(rt.FromHour)
	fromMinute := strconv.Itoa(rt.FromMinute)
	toDay := strconv.Itoa(rt.ToDay)
	toHour := strconv.Itoa(rt.ToHour)
	toMinute := strconv.Itoa(rt.ToMinute)
	return fmt.Sprintf("%v_%v_%v_%v_%v_%v_\n", fromDay, fromHour, fromMinute, toDay, toHour, toMinute)
}
func (rt RestrictedTime) valid() bool {
	if rt.FromDay == rt.ToDay {
		if rt.FromHour == rt.ToHour {
			if rt.FromMinute > rt.ToMinute {
				return false
			}
		} else if rt.FromHour > rt.ToHour {
			return false
		}
	} else if rt.FromDay > rt.ToDay {
		return false
	}
	return true
}
func sanNames(s string) string {
	s = strings.TrimSpace(s)
	if len(s) != 0 {
		s = strings.ToLower(s)
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}
func sanClasses(s string) string {
	s = strings.TrimSpace(s)
	if len(s) != 0 {
		s = strings.ToUpper(s)
	}
	return s
}
func sanFiles() {
	for _, file := range Files {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0700)
		if err != nil {
			fmt.Printf("File %v exists, cannot create\n", file)
		}
		f.Close()
	}
}
func studentListByRoom(room string, day string) [][]string {
	list := [][]string{}
	index := []string{}
	for _, b := range bookings {
		if b.ClassRoom == room && b.Day == day {
			index = append(index, b.Student)
		}
	}
	sort.Sort(sort.StringSlice(index))
	for _, s := range index {
		list = append(list, []string{
			students[s].Name,
			students[s].FirstName,
			students[s].Class,
			"",
			""})
	}
	return list
}
func adminStudentList() [][]string {
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
		if len(s) == 1 {
			list = append(list, []string{"index", strings.ToUpper(s), "", "", ""})
		} else {
			list = append(list, []string{
				students[s].User,
				students[s].Name,
				students[s].FirstName,
				students[s].Class,
				students[s].Password})
		}
	}
	return list
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
func sliceElemId(slice []string, elem string) int {
	for i, s := range slice {
		if s == elem {
			return i
		}
	}
	return 0
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
		RemainSeats[i] = []int{}
		for _, cr := range idxClassRooms {
			r := remaining(cr, d)
			//fmt.Printf("day:%v - idxclass:%v\n", d, cr)
			RemainSeats[i] = append(RemainSeats[i], r)
		}
	}
	//fmt.Println(RemainSeats)
}
func updateDate() {
	date := firstDayOfWeek()
	//fmt.Printf("date %v\n", date)
	for i := range idxDays {
		idxDates[i] = strconv.Itoa(date.Day()) + "/" +
			strconv.Itoa(int(date.Month())) + "/" +
			strconv.Itoa(date.Year())
		date = date.AddDate(0, 0, 1)
	}
}
func genPassword() string {
	rand.Seed(time.Now().UTC().UnixNano())
	elem := "ACDEFGHJKMNPQRSTUVWXYW2345679"
	length := 8
	password := make([]byte, length)
	for i := range password {
		password[i] = elem[rand.Intn(len(elem))]
	}
	return string(password)
}
func firstDayOfWeek() time.Time {
	date := time.Now()
	delta := -date.Day()
	//if the week is over(friday=4), add 1 week(7 days)
	if date.Day() > 4 {
		delta += 7
	}
	date = date.AddDate(0, 0, delta)
	return date
}

/*
func firstDayOfWeek() time.Time {
	year, week := time.Now().ISOWeek()
	//if the week is over(friday=4), select the next week
	if time.Now().Day() > 4 {
		week += 1
	}
	timezone, _ := time.LoadLocation("Europe/Paris")
	date := time.Date(year, 0, 0, 0, 0, 0, 0, timezone)
	isoYear, isoWeek := date.ISOWeek()

	// iterate back to Monday
	for date.Weekday() != time.Monday {
		date = date.AddDate(0, 0, -1)
		isoYear, isoWeek = date.ISOWeek()
	}

	// iterate forward to the first day of the first week
	for isoYear < year {
		date = date.AddDate(0, 0, 7)
		isoYear, isoWeek = date.ISOWeek()
	}

	// iterate forward to the first day of the given week
	for isoWeek < week {
		date = date.AddDate(0, 0, 7)
		isoYear, isoWeek = date.ISOWeek()
	}

	return date
}
*/
func addSubject(s string) {
	subjects = append(subjects, s)
	sort.Sort(sort.StringSlice(subjects))
	f, err := os.OpenFile(subjectsFile, os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(s + "\n")

}
func remSubject(s string) {
	for i, subject := range subjects {
		if subject == s {
			subjects = append(subjects[:i], subjects[i+1:]...)
		}
	}
	os.Remove(subjectsFile)
	f, err := os.OpenFile(subjectsFile, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, s := range subjects {
		f.WriteString(s + "\n")
	}
}
func addClass(s string) {
	classes = append(classes, s)
	//sort.Sort(sort.StringSlice(classes))
	f, err := os.OpenFile(classesFile, os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(s + "\n")

}
func remClass(s string) {
	for i, class := range classes {
		if class == s {
			classes = append(classes[:i], classes[i+1:]...)
		}
	}
	os.Remove(classesFile)
	f, err := os.OpenFile(classesFile, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, s := range classes {
		f.WriteString(s + "\n")
	}
}
func addClassRoom(cr ClassRoom) {
	_, exists := classRooms[cr.Id]
	if exists {
		remClassRoom(cr.Id)
	}
	classRooms[cr.Id] = cr
	idxClassRooms = append(idxClassRooms, cr.Id)
	saveIdxClassRooms()
	f, err := os.OpenFile(classRoomsFile, os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(cr.Id + "_" +
		cr.Name + "_" +
		strconv.Itoa(cr.Cap) + "_" +
		cr.Gender + "_" +
		strconv.FormatBool(cr.Group) + "_" + "\n")

}
func remClassRoom(id string) {
	delete(classRooms, id)
	for i, idcr := range idxClassRooms {
		if id == idcr {
			idxClassRooms = append(idxClassRooms[:i], idxClassRooms[i+1:]...)
			saveIdxClassRooms()
		}
	}
	os.Remove(classRoomsFile)
	f, err := os.OpenFile(classRoomsFile, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, cr := range classRooms {
		f.WriteString(cr.Id + "_" +
			cr.Name + "_" +
			strconv.Itoa(cr.Cap) + "_" +
			cr.Gender + "_" +
			strconv.FormatBool(cr.Group) + "_" + "\n")
	}
}
func moveUpClass(class string) {
	idx := 0
	for i, c := range classes {
		if c == class {
			idx = i
		}
	}
	if idx != 0 {
		classes = append(classes[:idx], classes[idx+1:]...)
		classes = append(classes[:idx-1],
			append([]string{class}, classes[idx-1:]...)...)
	}
	saveClasses()
}
func moveDownClass(class string) {
	idx := len(classes) - 1
	for i, c := range classes {
		if c == class {
			idx = i
		}
	}
	if idx != len(classes)-1 {
		classes = append(classes[:idx], classes[idx+1:]...)
		classes = append(classes[:idx+1],
			append([]string{class}, classes[idx+1:]...)...)
	}
	saveClasses()
}
func moveUpClassRoom(room string) {
	idx := 0
	for i, cr := range idxClassRooms {
		if classRooms[cr].Id == room {
			idx = i
		}
	}
	if idx != 0 {
		idxClassRooms = append(idxClassRooms[:idx], idxClassRooms[idx+1:]...)
		idxClassRooms = append(idxClassRooms[:idx-1],
			append([]string{room}, idxClassRooms[idx-1:]...)...)
	}
	saveIdxClassRooms()
}
func moveDownClassRoom(room string) {
	idx := len(idxClassRooms) - 1
	for i, cr := range idxClassRooms {
		if classRooms[cr].Id == room {
			idx = i
		}
	}
	if idx != len(idxClassRooms)-1 {
		idxClassRooms = append(idxClassRooms[:idx], idxClassRooms[idx+1:]...)
		idxClassRooms = append(idxClassRooms[:idx+1],
			append([]string{room}, idxClassRooms[idx+1:]...)...)
	}
	saveIdxClassRooms()
}
func addStudent(s Student) {
	students[s.User] = s
}
func remStudent(s string) error {
	var err error
	for _, b := range bookings {
		for _, g := range b.Group {
			if g == s {
				comm := &EventCon{
					Event{"remgroup", time.Now(), b.Day, b.Student, s},
					make(chan error)}
				c <- comm
				err = <-comm.Error
			}
		}
	}
	delete(students, s)
	return err
}
func addRestrictedTime(rt RestrictedTime) {
	restrictedHours = append(restrictedHours, rt)
	f, err := os.OpenFile(restrictedTimeFile, os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(rt.String())
	//fmt.Println(rt.String())
}
func remRestrictedTime(idx int) {
	for i := range restrictedHours {
		if idx == i {
			restrictedHours = append(restrictedHours[:i], restrictedHours[i+1:]...)
		}
	}
	os.Remove(restrictedTimeFile)
	f, err := os.OpenFile(restrictedTimeFile, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, rt := range restrictedHours {
		f.WriteString(rt.String())
	}
}
func loadClassRooms() {
	f, err := os.Open(classRoomsFile)
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
		capacity, _ := strconv.Atoi(split[2])
		group, _ := strconv.ParseBool(split[4])
		cr := ClassRoom{
			split[0],
			split[1],
			capacity,
			split[3],
			group}
		classRooms[split[0]] = cr
		//idxClassRooms = append(idxClassRooms, split[0])
	}
}
func saveClasses() {
	os.Remove(classesFile)
	f, err := os.OpenFile(classesFile, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, i := range classes {
		f.WriteString(i + "\n")
	}
}
func loadIdxClassRooms() {
	f, err := os.Open(idxClassRoomsFile)
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
		line = strings.Replace(line, "\n", "", -1)
		idxClassRooms = append(idxClassRooms, line)
	}
	if len(idxClassRooms) == 0 && len(classRooms) != 0 {
		for cr := range classRooms {
			idxClassRooms = append(idxClassRooms, cr)
		}
	}
}
func saveIdxClassRooms() {
	os.Remove(idxClassRoomsFile)
	f, err := os.OpenFile(idxClassRoomsFile, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, i := range idxClassRooms {
		f.WriteString(i + "\n")
	}
}

func loadSubjects() {
	f, err := os.Open(subjectsFile)
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
		line = strings.Replace(line, "\n", "", -1)
		subjects = append(subjects, line)
		sort.Sort(sort.StringSlice(subjects))
	}
}
func loadClasses() {
	f, err := os.Open(classesFile)
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
		line = strings.Replace(line, "\n", "", -1)
		classes = append(classes, line)
		sort.Sort(sort.StringSlice(classes))
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

func loadAdmins() {
	f, err := os.Open(adminsFile)
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
		s := Admin{
			split[0],
			split[1]}
		admins[split[0]] = s
	}
}
func loadRestrictedTime() {
	f, err := os.Open(restrictedTimeFile)
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
		line = strings.Replace(line, "\n", "", -1)
		split := strings.Split(line, "_")
		fromDay, _ := strconv.Atoi(split[0])
		fromHour, _ := strconv.Atoi(split[1])
		fromMinute, _ := strconv.Atoi(split[2])
		toDay, _ := strconv.Atoi(split[3])
		toHour, _ := strconv.Atoi(split[4])
		toMinute, _ := strconv.Atoi(split[5])
		rt := RestrictedTime{
			fromDay,
			fromHour,
			fromMinute,
			toDay,
			toHour,
			toMinute}
		restrictedHours = append(restrictedHours, rt)
	}
}

/*
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
	//classes := []string{"2A", "2B", "2C", "2D", "2E", "1S1", "1S2", "1L", "1ES", "1STG", "TS1", "TS2", "TL", "TES", "TSTG", "2COM", "2SEC", "2ALIM", "1COM", "1SEC", "1ALIM", "TCOM", "TSEC", "TALIM"}
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
*/
func genStudents() {
	students := map[string]Student{}
	files := []string{"data/Internes filles LEGT.csv",
		"data/Internes filles LP.csv",
		"data/Internes garçons LEGT.csv",
		"data/Internes garçons LP.csv"}
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
			//fmt.Println(line)
			line = toUtf8(line)
			//fmt.Println(line)
			split := strings.Split(line, ";")
			name := sanNames(split[0])
			firstname := sanNames(split[1])
			user := ""
			if name != "" || firstname != "" {
				user = name + firstname
				user = sanUser(user)
			}
			gender := strings.ToUpper(split[2])
			class := sanClasses(split[3])
			password := genPassword()
			//if user already exists(homonym), append a number and increment it if multiple cases
			i := 1
			_, user_exists := students[user]
			for user_exists {
				if i == 1 {
					user = user + strconv.Itoa(i)
				} else {
					user = user[len(user)-1:] + strconv.Itoa(i)
				}
				_, user_exists = students[user]
				i += 1
			}
			//fmt.Printf("%v %v %v %v %v %v\n", user, name, firstname, gender, class, password)
			if name != "" && firstname != "" && user != "" && class != "" && gender != "" {
				students[user] = Student{user, name, firstname, class, gender, password}
			}
		}
		f.Close()
	}

	os.Remove(studentsFile)
	f, err := os.OpenFile(studentsFile, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	for s := range students {
		f.WriteString(students[s].User + "_" +
			students[s].Name + "_" +
			students[s].FirstName + "_" +
			students[s].Class + "_" +
			students[s].Gender + "_" +
			students[s].Password + "_\n")
	}
	f.Close()
}

func sanPass(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	return s
}
func sanUser(s string) string {
	var buf bytes.Buffer
	var c string
	s = strings.ToLower(s)
	convTable := [][]string{
		[]string{"é", "e"},
		[]string{"è", "e"},
		[]string{"ë", "e"},
		[]string{"ê", "e"},
		[]string{"æ", "ae"},
		[]string{"œ", "oe"},
		[]string{"ï", "i"},
		[]string{"ç", "c"},
		[]string{"ö", "o"},
		[]string{"ù", "u"},
		[]string{"ü", "u"},
		[]string{" ", ""},
		[]string{"-", ""},
		[]string{".", ""},
		[]string{"'", ""},
		[]string{"_", ""}}
	for _, runeValue := range s {
		c = string(runeValue)
		for j := range convTable {
			if c == convTable[j][0] {
				c = convTable[j][1]
			}
		}
		buf.WriteString(c)
	}
	return buf.String()
}
func toUtf8(s string) string {
	var buf bytes.Buffer
	for _, b := range []byte(s) {
		r := rune(b)
		buf.WriteRune(r)
	}
	return string(buf.Bytes())
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
func timeProcessor() {
	for {
		timezone, _ := time.LoadLocation("Europe/Paris")
		now := time.Now()
		first := firstDayOfWeek()
		bookingsEnabled = true
		//block access during these hours
		for _, restrictedHour := range restrictedHours {
			start := time.Date(
				first.Year(),
				first.Month(),
				first.Day(),
				restrictedHour.FromHour,
				restrictedHour.FromMinute,
				0,
				0,
				timezone)
			start = start.AddDate(0, 0, restrictedHour.FromDay)
			end := time.Date(
				first.Year(),
				first.Month(),
				first.Day(),
				restrictedHour.ToHour,
				restrictedHour.ToMinute,
				0,
				0,
				timezone)
			end = end.AddDate(0, 0, restrictedHour.ToDay)
			if now.After(start) && now.Before(end) {
				bookingsEnabled = false
			}
		}
		//reset all events and bookings for the next week
		if now.Day() == resetTime.Day &&
			now.Hour() == resetTime.Hour &&
			now.Minute() == resetTime.Minute &&
			now.Second() == 0 {
			fmt.Println("Events reset!")
			resetEvents()
			loadEvents()
			remainUpdate()
			updateDate()
		}
		time.Sleep(900 * time.Millisecond)
	}
}

func main() {
	sanFiles()
	//genStudents()
	loadAdmins()
	loadStudents()
	loadSubjects()
	loadClasses()
	loadClassRooms()
	loadIdxClassRooms()
	//resetEvents()
	loadEvents()
	loadRestrictedTime()
	updateDate()
	remainUpdate()
	//fmt.Println(bookings)

	//tableClip()

	go eventProcessor()
	go timeProcessor()
	http.Handle("/bs/", http.StripPrefix("/bs/", http.FileServer(http.Dir("bs/"))))
	http.HandleFunc("/login/", loginHandler)
	http.HandleFunc("/admin", makeHandler(adminHandler))
	http.HandleFunc("/submit", makeHandler(submitHandler))
	http.HandleFunc("/", makeHandler(rootHandler))
	http.ListenAndServe(":8080", nil)
}
