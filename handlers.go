package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func rootHandler(w http.ResponseWriter, r *http.Request, con *TmplCon) {
	renderTemplate(w, "view", con)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form) == 0 {
		fmt.Println("Form empty, delete cookie")
		cookie := http.Cookie{Name: "session", Value: "deleted", HttpOnly: false, Path: "/"}
		http.SetCookie(w, &cookie)
		renderTemplate(w, "login", nil)
		return
	}
	user := r.Form["user"][0]
	pass := r.Form["password"][0]
	fmt.Printf("formUser:%v formPass:%v\n", user, pass)
	_, user_ok := students[user]
	fmt.Printf("pass_ok:%v, user_ok:%v\n", students[user].Password == pass, user_ok)
	if students[user].Password == pass && user_ok {
		cookie := http.Cookie{Name: "session", Value: user + "/" + hash(user), HttpOnly: false, Path: "/"}
		http.SetCookie(w, &cookie)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func submitHandler(w http.ResponseWriter, r *http.Request, con *TmplCon) {
	values := r.URL.Query()
	eventType := values["t"][0]
	day := values["d"][0]
	if eventType == "pickgroup" || eventType == "picksubject" {
		con.Values = values
		renderTemplate(w, eventType, con)
		return
	}
	value := values["id"][0]
	comm := &EventCon{
		Event{
			eventType,
			time.Now(),
			day,
			con.Student.User,
			value},
		make(chan error)}
	c <- comm
	err := <-comm.Error
	//fmt.Println(comm)
	//fmt.Println(err)
	if err != nil {
		con.Errors = append(con.Errors, err)
		renderTemplate(w, "view", con)
		return
	} else {
		http.Redirect(w, r, "/#"+day, http.StatusFound)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, *TmplCon)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		userhash := strings.Split(cookie.Value, "/")
		if len(userhash) == 1 || userhash[1] != hash(userhash[0]) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		student, _ := students[userhash[0]]
		remainUpdate()
		con := &TmplCon{
			student,
			&idxDays,
			&idxClassRooms,
			&idxSubjects,
			&classRooms,
			&RemainSeats,
			occupancy(student.User),
			workList(student.User),
			[]error{},
			studentList(student.User),
			groupList(student.User),
			map[string][]string{}}
		fn(w, r, con)
	}
}
