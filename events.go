package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func (e Event) book() error {
	exist := false
	for i, b := range bookings {
		if b.Student == e.Student && b.Day == e.Day {
			if e.Type == "room" && bookings[i].ClassRoom != e.Value {
				if remaining(e.Value, e.Day) > 0 || classRooms[e.Value].Cap == -1 {
					bookings[i].ClassRoom = e.Value
					if classRooms[bookings[i].ClassRoom].Group {
						bookings[i].Group = []string{}
					}
				} else {
					return errors.New("Il n'y a plus de places " + e.Day + " dans la salle suivante : " + e.Value)
				}
			} else if e.Type == "addgroup" && len(bookings[i].Group) < 4 {
				fmt.Println("adding to group")
				for _, g := range bookings[i].Group {
					if g == e.Value {
						return errors.New("Cet étudiant fait déjà parti du groupe de travail")
					}
				}
				bookings[i].Group = append(bookings[i].Group, e.Value)
			} else if e.Type == "remgroup" {
				for j, g := range bookings[i].Group {
					if g == e.Value {
						bookings[i].Group = append(bookings[i].Group[:j], bookings[i].Group[j+1:]...)
					}
				}
			} else if e.Type == "addrevision" || e.Type == "addexercise" || e.Type == "addresearch" {
				e.Type = e.Type[3:]
				if !contains(bookings[i].Work[e.Type], e.Value) {
					bookings[i].Work[e.Type] = append(bookings[i].Work[e.Type], e.Value)
				}
			} else if e.Type == "remrevision" || e.Type == "remexercise" || e.Type == "remresearch" {
				e.Type = e.Type[3:]
				if contains(bookings[i].Work[e.Type], e.Value) {
					for j, w := range bookings[i].Work[e.Type] {
						if w == e.Value {
							bookings[i].Work[e.Type] = append(bookings[i].Work[e.Type][:j], bookings[i].Work[e.Type][j+1:]...)
						}
					}
				}
			}
			exist = true
		}
	}
	if !exist {
		bookings = append(bookings, Booking{e.Day, e.Student, e.Value, []string{}, map[string][]string{
			"revision": []string{},
			"exercise": []string{},
			"research": []string{}}})
	}
	return nil
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
	return fmt.Sprintf("%v_%v_%v_%v_%v_\n", e.Type, e.Date, e.Day, e.Student, e.Value)
}
func eventProcessor() {
	for e := range c {
		events = append(events, e.Event)
		e.Event.log()
		e.Error <- e.Event.book()
	}
}
func resetEvents() {
	fmt.Println("Emptying bookings")
	bookings = []Booking{}
	fmt.Println("Removing events file")
	os.Remove(eventsFile)
	fmt.Println("Creating new events file")
	f, err := os.OpenFile(eventsFile, os.O_CREATE, 0700)
	if err != nil {
		panic(err)
	}
	f.Close()
	for _, cr := range classRooms {
		for i, s := range students {
			if cr.Gender == s.Gender {
				for _, d := range idxDays {
					e := Event{"room", time.Now(), d, i, cr.Id}
					fmt.Println(e)
					e.log()
					events = append(events, e)
				}
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
		//convert string to time
		time, _ := time.Parse("2006-01-02 15:04:00.000000000 +0200 CEST", splitEvent[1])
		e := Event{
			splitEvent[0],
			time,
			splitEvent[2],
			splitEvent[3],
			splitEvent[4]}
		events = append(events, e)
		e.book()
	}
}
