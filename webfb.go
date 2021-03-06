package main

import (
    "encoding/json"
    "fmt"
    "html/template"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
)

const RcFile = "webfbrc"

var config Configuration

type Configuration struct {
    FileBrowser FileBrowserConfiguration
    DefaultActions []DefaultAction
}

type FileBrowserConfiguration struct {
    DefaultPath string
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
		break
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

    // by default we list the directory containing the file we tried to hanled
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

func GetDefaultLisingPath() string {
    // the the configured default path
    if config.FileBrowser.DefaultPath != "" {
	return config.FileBrowser.DefaultPath
    }

    // try the executeable's directory
    bindir, err := filepath.Abs(filepath.Dir(os.Args[0]))
    if err != nil {
	log.Printf("Error while determining the executeable's directory: %s\n", err)
    } else {
	return bindir
    }

    // use the current working directory
    return "."
}

func FileBrowserListDirectory(w http.ResponseWriter, r *http.Request) {
    path := r.FormValue("path")

    if path == "" {
	path = GetDefaultLisingPath()
    }
    absolute, err := filepath.Abs(path)
    if err != nil {
	fmt.Fprintf(w, "error while getting the absolute path for %s: %s\n", err)
	return
    }

    log.Printf("listing: %s\n", absolute)

    directory, err := GetDirectory(absolute)
    if err != nil {
	http.Redirect(w, r, "/list?path=" + filepath.Dir(absolute), http.StatusFound)
	return
    }
    
    template, err := template.ParseFiles("list")
    if err != nil {
	fmt.Fprintf(w, "error while parsing the template: %s\n", err)
	return
    }
    err = template.Execute(w, directory)
    if err != nil {
	fmt.Fprintf(w, "error while executing the template: %s\n", err)
	return
    }
}

func main() {
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

    http.HandleFunc("/", FileBrowserListDirectory)
    http.HandleFunc("/list", FileBrowserListDirectory)
    http.HandleFunc("/defaultaction", FileBrowserDefaultAction)
    http.ListenAndServe("localhost:4000", nil)
}

