package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func rootHandler(w http.ResponseWriter, r *http.Request, con *TmplCon) {
	renderTemplate(w, "view", con)
}

func adminHandler(w http.ResponseWriter, r *http.Request, con *TmplCon) {
	values := r.URL.Query()
	if len(values) == 0 {
		if con.Student.User == "admin" {
			renderTemplate(w, "admin", con)
		} else if con.Student.User == "viesco" {
			renderTemplate(w, "viesco", con)
		}
		return
	}
	tmpl := values["tmpl"][0]
	//log out viesco user if it request an unauthorised template
	if con.Student.User == "viesco" && tmpl != "admin_lists" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	r.ParseForm()
	if len(r.PostForm) != 0 {
		if tmpl == "admin_students" {
			name := sanNames(r.Form.Get("name"))
			firstname := sanNames(r.Form.Get("firstname"))
			class := r.Form.Get("class")
			gender := r.Form.Get("gender")
			user := name + firstname
			user = sanUser(user)
			pass := genPassword()
			if len(name) != 0 &&
				len(firstname) != 0 &&
				len(strings.TrimSpace(class)) != 0 &&
				len(strings.TrimSpace(gender)) != 0 {
				addStudent(Student{user, name, firstname, class, gender, pass})
			}
			http.Redirect(w, r, "/admin?tmpl="+tmpl+"#"+user, http.StatusFound)
			return
		}
		if tmpl == "admin_classrooms" {
			id := strings.TrimSpace(r.Form.Get("id"))
			desc := strings.TrimSpace(r.Form.Get("desc"))
			capacity, _ := strconv.Atoi(strings.TrimSpace(r.Form.Get("cap")))
			gender := r.Form.Get("gender")
			group, _ := strconv.ParseBool(r.Form.Get("group"))
			name := id
			if group {
				name = name + " Etude en groupe"
			} else {
				name = name + " Etude individuelle"
			}
			if desc != "" {
				name = name + " " + desc
			}
			if len(id) != 0 && len(name) != 0 {
				//fmt.Println(ClassRoom{id, name, capacity, gender, group})
				addClassRoom(ClassRoom{id, name, capacity, gender, group})
			}
			http.Redirect(w, r, "/admin?tmpl="+tmpl, http.StatusFound)
			return
		}
		if tmpl == "admin_subjects" {
			name := sanNames(r.Form.Get("subject"))
			if len(name) != 0 {
				addSubject(name)
			}
			http.Redirect(w, r, "/admin?tmpl="+tmpl, http.StatusFound)
			return
		}
		if tmpl == "admin_classes" {
			name := sanClasses(r.Form.Get("class"))
			if len(name) != 0 && !contains(classes, name) {
				addClass(name)
			}
			http.Redirect(w, r, "/admin?tmpl="+tmpl, http.StatusFound)
			return
		}
		if tmpl == "admin_restrictedtime" {
			fromDay, _ := strconv.Atoi(r.Form.Get("fromday"))
			fromHour, _ := strconv.Atoi(r.Form.Get("fromhour"))
			fromMinute, _ := strconv.Atoi(r.Form.Get("fromminute"))
			toDay, _ := strconv.Atoi(r.Form.Get("today"))
			toHour, _ := strconv.Atoi(r.Form.Get("tohour"))
			toMinute, _ := strconv.Atoi(r.Form.Get("tominute"))
			rt := RestrictedTime{
				fromDay,
				fromHour,
				fromMinute,
				toDay,
				toHour,
				toMinute}
			if rt.valid() &&
				fromDay != -1 &&
				fromHour != -1 &&
				fromMinute != -1 &&
				toDay != -1 &&
				toHour != -1 &&
				toMinute != -1 {
				addRestrictedTime(rt)
			}
			http.Redirect(w, r, "/admin?tmpl="+tmpl, http.StatusFound)
			return
		}
	}
	action, action_ok := values["action"]
	if action_ok {
		if action[0] == "genpass" {
			student := students[values["param"][0]]
			student.Password = genPassword()
			remStudent(student.User)
			addStudent(student)
		}
		if action[0] == "remstudent" {
			remStudent(values["param"][0])
		}
		if action[0] == "remsubject" {
			remSubject(values["param"][0])
		}
		if action[0] == "remclass" {
			remClass(values["param"][0])
		}
		if action[0] == "remclassroom" {
			remClassRoom(values["param"][0])
			fmt.Println(classRooms)
		}
		if action[0] == "moveupclassroom" {
			moveUpClassRoom(values["param"][0])
			fmt.Println(classRooms)
		}
		if action[0] == "movedownclassroom" {
			moveDownClassRoom(values["param"][0])
			fmt.Println(classRooms)
		}
		if action[0] == "moveupclass" {
			moveUpClass(values["param"][0])
			fmt.Println(classes)
		}
		if action[0] == "movedownclass" {
			moveDownClass(values["param"][0])
			fmt.Println(classes)
		}
		if action[0] == "remrestrictedtime" {
			idx, _ := strconv.Atoi(values["param"][0])
			remRestrictedTime(idx)
			fmt.Println(restrictedHours)
		}
		if action[0] == "print" {
			room := (values["room"][0])
			day := (values["day"][0])
			pdfStudentList(w, room, day)
			return
		}
		if action[0] == "printnotice" {
			student := (values["param"][0])
			pdfStudentNotice(w, student)
			return
		}
		http.Redirect(w, r, "/admin?tmpl="+tmpl, http.StatusFound)
		return
	}

	renderTemplate(w, tmpl, con)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form) == 0 {
		var err error
		fmt.Println("Form empty, delete cookie")
		cookie := http.Cookie{Name: "session", Value: "deleted", HttpOnly: false, Path: "/"}
		http.SetCookie(w, &cookie)
		if !bookingsEnabled {
			err = errors.New("Le service est indisponible à cette heure-ci")
			renderTemplate(w, "login", &TmplCon{Errors: []error{err}})
		} else {
			renderTemplate(w, "login", &TmplCon{Errors: nil})
		}
		return
	}
	user := sanUser(r.Form["user"][0])
	pass := sanPass(r.Form["password"][0])
	fmt.Printf("formUser:%v formPass:%v\n", user, pass)
	_, admin_ok := admins[user]
	fmt.Printf("admin pass_ok:%v, admin user_ok:%v\n", admins[user].Password == pass, admin_ok)
	if sanPass(admins[user].Password) == pass && admin_ok {
		cookie := http.Cookie{Name: "session", Value: user + "/" + hash(user), HttpOnly: false, Path: "/"}
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	_, user_ok := students[user]
	fmt.Printf("pass_ok:%v, user_ok:%v\n", students[user].Password == pass, user_ok)
	if sanPass(students[user].Password) == pass && user_ok {
		cookie := http.Cookie{Name: "session", Value: user + "/" + hash(user), HttpOnly: false, Path: "/"}
		http.SetCookie(w, &cookie)
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return
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
		fmt.Printf("r.url:%v\n", r.URL.Path)
		_, isAdmin := admins[userhash[0]]
		if string(r.URL.Path) == "/admin" && !isAdmin {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if string(r.URL.Path) != "/admin" && isAdmin {
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		if string(r.URL.Path) != "/admin" && !bookingsEnabled {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		var student Student
		var studentSlice [][]string
		if isAdmin {
			student = Student{userhash[0], "", "", "", "", ""}
			studentSlice = adminStudentList()
		} else {
			student, _ = students[userhash[0]]
			studentSlice = studentList(student.User)
		}
		remainUpdate()
		con := &TmplCon{
			student,
			&idxDays,
			&idxWeek,
			&idxDates,
			&idxClassRooms,
			&subjects,
			&classRooms,
			&classes,
			&RemainSeats,
			&restrictedHours,
			occupancy(student.User),
			workList(student.User),
			[]error{},
			studentSlice,
			groupList(student.User),
			map[string][]string{}}
		fn(w, r, con)
	}
}
