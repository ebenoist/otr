package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell"
	"github.com/pkg/xattr"
	"github.com/rivo/tview"
	"howett.net/plist"
)

type Note struct {
	UpdatedAt time.Time
	Tags      []string
	Stared    bool
	Info      os.FileInfo
	Path      string
	Name      string
}

func main() {
	fmt.Println("oh hai 🦦")
	root := os.Args[1]

	var notes []*Note
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		var tags []string
		meta, err := xattr.Get(path, "com.apple.metadata:_kMDItemUserTags")
		_, err = plist.Unmarshal(meta, &tags)

		notes = append(notes, &Note{
			UpdatedAt: info.ModTime(),
			Path:      path,
			Name:      filepath.Base(path),
			Info:      info,
			Tags:      tags,
		})

		return nil
	})

	if err != nil {
		panic(err)
	}

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

	app := tview.NewApplication()
	list := tview.NewList()

	list.SetTitle("notes").SetBorder(true)

	preview := tview.NewTextView().SetDynamicColors(true)
	preview.SetTitle("preview").SetBorder(true)

	for _, note := range notes {
		updated := note.UpdatedAt.Format("Mon Jan _2 3:04PM")
		msg := fmt.Sprintf("%s\t\t%db\t\t%+v", updated, note.Info.Size(), note.Tags)
		list.AddItem(note.Name, msg, 0, nil)
	}

	list.AddItem("Quit", "Press to exit", 'q', func() {
		app.Stop()
	})

	cache := make(map[int]string)

	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= len(notes) {
			return
		}

		go func(idx int) {
			n := notes[idx]
			cmd := exec.Command("bat", "--color", "always", "--paging", "always", n.Path)
			cmd.Env = []string{"COLORTERM=truecolor"}
			out, err := cmd.Output()
			if err != nil {
				return
			}

			buff := &bytes.Buffer{}
			w := tview.ANSIWriter(buff)
			if _, err := io.Copy(w, bytes.NewReader(out)); err != nil {
				return
			}

			cache[idx] = buff.String()

			app.QueueUpdateDraw(func() {
				render, ok := cache[list.GetCurrentItem()]
				if ok {
					preview.SetTitle(notes[list.GetCurrentItem()].Name)
					preview.SetText(render)
				}
			})
		}(index)
	})

	flex := tview.NewFlex().
		AddItem(list, 0, 1, false).
		AddItem(preview, 0, 2, false)

	if err := app.SetRoot(flex, true).SetFocus(list).Run(); err != nil {
		panic(err)
	}
}
