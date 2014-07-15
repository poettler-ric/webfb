package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "os/exec"
//    "os/user"
    "html/template"
    "net/http"
    "path/filepath"
)

const RcFile = "webfbrc"

var config Configuration

type Configuration struct {
    DefaultActions []DefaultAction
}

type DefaultAction struct {
    Pattern string
    Command string
}

type Directory struct {
    Path string
    Separator string
    Entries []Dirent
}

type Dirent struct {
    Name string
    IsDirectory bool
    Viewed bool
}

func FileBrowserDefaultAction(w http.ResponseWriter, r *http.Request) {
    path := r.FormValue("path")
    // TODO: check for a valid path
    path = filepath.Clean(path)
    log.Printf("default action: %s\n", path)

    stat, err := os.Stat(path)
    if err != nil {
	log.Print(err)
    } else {
	if stat.IsDir() {
	    http.Redirect(w, r, "/list?path=" + path, http.StatusFound)
	    return
	}

	base := filepath.Base(path)
	for _, entry := range config.DefaultActions {
	    matched, err := filepath.Match(entry.Pattern, base)
	    if err != nil {
		log.Print(err)
		return
	    }
	    if matched {
		cmd := exec.Command(entry.Command, path)
		err := cmd.Start()
		if err != nil {
		    log.Print(err)
		}
		break
	    }
	}
    }
    http.Redirect(w, r, "/list?path=" + filepath.Dir(path), http.StatusFound)
}

func GetDirectory(path string) (*Directory, error) {
    result := &Directory{ Path: path, Separator: string(os.PathSeparator) }

    dirContent, err := ioutil.ReadDir(path)
    if err != nil {
	log.Print(err)
	return nil, err
    }

    result.Entries = make([]Dirent, len(dirContent))
    for i, val := range dirContent {
	result.Entries[i].Name = val.Name()
	result.Entries[i].IsDirectory = val.IsDir()
	result.Entries[i].Viewed = false
    }
    return result, nil
}

func FileBrowserListDirectory(w http.ResponseWriter, r *http.Request) {
    path := r.FormValue("path")
    // TODO: check for a valid path
    log.Printf("listing: %s\n", path)

    directory, err := GetDirectory(path)
    if err != nil {
	http.Redirect(w, r, "/list?path=" + filepath.Dir(path), http.StatusFound)
	return
    }
    
    template, err := template.ParseFiles("list")
    if err != nil {
	log.Fatal(err)
    }
    err = template.Execute(w, directory)
    if err != nil {
	log.Fatal(err)
    }
}

func main() {
    /*cwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
    if err != nil {
	log.Fatal(err)
    }
    fmt.Println("cwd:", cwd)*/

    /*user, err := user.Current()
    if err != nil {
	log.Fatal(err)
    }
    configFile := filepath.Join(user.HomeDir, RcFile)*/

    configFile := RcFile
    if _, err := os.Stat(configFile); err == nil {
	configContent, err := ioutil.ReadFile(configFile)
	if err != nil {
	    log.Fatal(err)
	}
	err = json.Unmarshal(configContent, &config)
	if err != nil {
	    log.Fatal(err)
	}
    }

    http.HandleFunc("/list", FileBrowserListDirectory)
    http.HandleFunc("/defaultaction", FileBrowserDefaultAction)
    http.ListenAndServe("localhost:4000", nil)

    fmt.Println("== done ==")
}

