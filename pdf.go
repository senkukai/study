package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/jung-kurt/gofpdf"
)

const margeCell = 2 // marge top/bottom of cell

/*
Table with one line by row
*/
func pdfStudentNotice(w io.Writer, student string) {
	var rows []string
	if student == "all" {
		for s := range students {
			rows = append(rows, s)
		}
		sort.Sort(sort.StringSlice(rows))
	}

	buf := bytes.NewBuffer(nil)
	f, _ := os.Open("data/texte_notice") // Error handling elided for brevity.
	io.Copy(buf, f)                      // Error handling elided for brevity.
	f.Close()
	notice := string(buf.Bytes())

	pdf := gofpdf.New("", "", "", "")
	utf := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 8)

	//	_, pageh := pdf.GetPageSize()
	pdf.SetAutoPageBreak(false, 20)
	_, lineHt := pdf.GetFontSize()

	if student != "all" {
		headers := fmt.Sprintf("%v %v %v    Identifiant: %v  Mot de passe: %v\n\n",
			students[student].Name,
			students[student].FirstName,
			students[student].Class,
			students[student].User,
			students[student].Password)
		pdf.SetFont("Arial", "B", 8)
		pdf.MultiCell(0, lineHt+margeCell, utf(headers), "", "", false)
		pdf.SetFont("Arial", "", 8)
		pdf.MultiCell(0, lineHt+margeCell, utf(notice), "", "", false)
	} else {
		for i, row := range rows {
			headers := fmt.Sprintf("%v %v %v    Identifiant: %v  Mot de passe: %v\n\n",
				students[row].Name,
				students[row].FirstName,
				students[row].Class,
				students[row].User,
				students[row].Password)

			pdf.SetFont("Arial", "B", 8)
			pdf.MultiCell(0, lineHt+margeCell, utf(headers), "", "", false)
			pdf.SetFont("Arial", "", 8)
			pdf.MultiCell(0, lineHt+margeCell, utf(notice), "", "", false)

			if i%2 == 0 {
				pdf.Ln(20)
			} else {
				pdf.AddPage()
			}

		}
	}
	pdf.Output(w)

}
func pdfStudentList(w io.Writer, room string, day string) {
	cols := []float64{60, 60, 20, 25, 25}
	datefull := idxDays[sliceElemId(idxDays, day)] + " " + idxDates[sliceElemId(idxDays, day)]
	rows := studentListByRoom(room, day)

	pdf := gofpdf.New("", "", "", "")
	utf := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)

	pdf.SetHeaderFunc(func() {
		pdf.SetY(10)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 0, fmt.Sprintf("%v %v", utf(datefull), utf(classRooms[room].Name)),
			"", 0, "", false, 0, "")
		pdf.Ln(10)
	})

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 0, fmt.Sprintf("Page %d", pdf.PageNo()),
			"", 0, "C", false, 0, "")
	})

	_, pageh := pdf.GetPageSize()
	_, _, _, mbottom := pdf.GetMargins()

	pdf.SetFont("Arial", "", 18)
	pdf.WriteAligned(0, 0, utf(datefull), "C")
	pdf.Ln(10)
	pdf.WriteAligned(0, 0, utf(classRooms[room].Name), "C")
	pdf.SetFont("Arial", "", 12)
	pdf.Ln(15)
	for idx, row := range rows {
		_, lineHt := pdf.GetFontSize()
		height := lineHt + margeCell

		x, y := pdf.GetXY()
		if idx == 0 {
			pdf.SetFont("Arial", "B", 12)
			for i, txt := range []string{"Nom", "Prénom", "Classe", "", ""} {
				width := cols[i]
				pdf.Rect(x, y, width, height, "")
				pdf.ClipRect(x, y, width, height, false)
				pdf.Cell(width, height, utf(txt))
				pdf.ClipEnd()
				x += width
			}
			pdf.Ln(-1)
			x, y = pdf.GetXY()
		}
		// add a new page if the height of the row doesn't fit on the page
		if y+height >= pageh-mbottom {
			pdf.AddPage()
			x, y = pdf.GetXY()
			pdf.SetFont("Arial", "B", 12)
			for i, txt := range []string{"Nom", "Prénom", "Classe", "", ""} {
				width := cols[i]
				pdf.Rect(x, y, width, height, "")
				pdf.ClipRect(x, y, width, height, false)
				pdf.Cell(width, height, utf(txt))
				pdf.ClipEnd()
				x += width
			}
			pdf.Ln(-1)
			x, y = pdf.GetXY()
		}
		pdf.SetFont("Arial", "", 12)
		for i, txt := range row {
			width := cols[i]
			pdf.Rect(x, y, width, height, "")
			pdf.ClipRect(x, y, width, height, false)
			pdf.Cell(width, height, utf(txt))
			pdf.ClipEnd()
			x += width
		}
		pdf.Ln(-1)
	}

	pdf.Ln(10)
	pdf.WriteAligned(0, 0, "Total: "+strconv.Itoa(len(rows)), "R")
	pdf.Output(w)
}
func genPdf() {

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	count := 1
	utf := pdf.UnicodeTranslatorFromDescriptor("")
	for _, s := range students {
		pdf.WriteAligned(0, 5, strconv.Itoa(count)+". "+utf(s.Name)+" "+utf(s.FirstName)+" "+s.Class, "")
		count += 1
		pdf.Ln(5)
	}
	pdf.OutputFileAndClose("/tmp/hello.pdf")
}
